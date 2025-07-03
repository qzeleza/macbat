package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/getlantern/systray"
	"golang.org/x/term" // Для определения, запущены ли мы из терминала

	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
)

var log *logger.Logger

func main() {
	// Инициализация логгера
	log = logger.New(paths.LogPath(), 100, true, false)

	// --- Инициализация конфигурации ---
	cfgManager, err := config.New(log, paths.ConfigPath())
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка загрузки конфигурации: %v", err))
	}
	conf, err := cfgManager.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка загрузки конфигурации: %v", err))
	}

	// --- Обработка флагов командной строки ---
	installFlag := flag.Bool("install", false, "Установить приложение и агент launchd")
	uninstallFlag := flag.Bool("uninstall", false, "Удалить приложение и агент launchd")
	backgroundFlag := flag.Bool("background", false, "Запуск в фоновом режиме")
	flag.Parse()

	// --- Логика установки/удаления ---
	if *installFlag {
		log.Info("Запрошена установка приложения...")
		if err := Install(log, conf); err != nil {
			log.Fatal(fmt.Sprintf("Ошибка во время установки: %v", err))
		}
		log.Info("Установка успешно завершена.")
		return
	}

	if *uninstallFlag {
		log.Info("Запрошено удаление приложения...")
		if err := Uninstall(log); err != nil {
			log.Fatal(fmt.Sprintf("Ошибка во время удаления: %v", err))
		}
		log.Info("Удаление успешно завершено.")
		return
	}

	// --- Логика фонового режима ---
	if *backgroundFlag {
		// Если мы запускаемся из терминала, нужно отсоединиться, чтобы не блокировать его.
		// Это делает запуск `./macbat --background` более удобным для пользователя.
		if term.IsTerminal(int(os.Stdout.Fd())) {
			// Запускаем себя же в фоне и выходим
			launchInBackground()
			log.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")
			return
		}

		// Записываем PID фонового процесса для корректного завершения из GUI
		if err := writePID(); err != nil {
			log.Error(fmt.Sprintf("Не удалось записать PID файла: %v", err))
		}

		log.Info("Запуск в фоновом режиме...")
		runBackgroundMainTask(conf, cfgManager, "run")

		// Удаляем PID-файл перед выходом, так как процесс завершается
		_ = os.Remove(paths.PIDPath())
		return
	}

	// --- Логика для GUI (иконка в трее) ---
	log.Info("Запуск иконки в трее...")

	// Перед запуском нового фонового процесса принудительно завершаем старый,
	// если он остался от предыдущего сбоя. Это гарантирует, что мы не создадим зомби.
	killBackgroundGo()

	// Запускаем фоновый процесс, если он еще не запущен
	if !isProcessRunning("macbat --background") {
		log.Info("Запуск фонового процесса мониторинга батареи...")
		launchInBackground()
	} else {
		log.Info("Фоновый процесс уже запущен.")
	}

	// Просто запускаем GUI. systray.Run() - блокирующая операция,
	// она будет держать процесс активным.
	// Убрана сложная и подверженная ошибкам логика daemonizeGUI.
	systray.Run(onReady, onExit)
}

func onExit() {
	// Здесь можно выполнить очистку перед выходом
	log.Info("Выход из приложения")
	// Удаляем PID-файл перед выходом
	_ = os.Remove(paths.PIDPath())
	os.Exit(0)
}

// isProcessRunning проверяет, запущен ли уже процесс с указанным именем.
// Использует `pgrep` для поиска процесса.
func isProcessRunning(pattern string) bool {
	// Команда ищет процесс по имени, исключая сам процесс pgrep, чтобы избежать ложных срабатываний.
	cmd := exec.Command("sh", "-c", fmt.Sprintf("pgrep -f '%s' | grep -v pgrep", pattern))
	output, err := cmd.Output()

	if err != nil {
		// Если err != nil, значит pgrep ничего не нашел, что является ожидаемым поведением.
		return false
	}

	// Если вывод не пустой, значит процесс найден.
	return len(output) > 0
}

// writePID записывает PID текущего процесса в файл.
// Это используется для того, чтобы GUI мог найти и завершить фоновый процесс.
func writePID() error {
	pidPath := paths.PIDPath()
	pid := os.Getpid()
	log.Info(fmt.Sprintf("Запись PID %d в файл: %s", pid, pidPath))

	// Создаем директорию, если она не существует
	dir := filepath.Dir(pidPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("не удалось создать директорию для PID файла: %w", err)
		}
	}

	// Записываем PID в файл
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("не удалось записать в PID файл: %w", err)
	}
	return nil
}

// killBackgroundGo находит и завершает фоновый процесс.
func killBackgroundGo() {
	pidPath := paths.PIDPath()
	log.Info(fmt.Sprintf("Попытка завершить фоновый процесс через PID файл: %s", pidPath))

	// Проверяем, существует ли PID файл
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		log.Info("PID файл не найден. Возможно, фоновый процесс не запущен или уже завершен.")
		return
	}

	// Читаем PID из файла
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		log.Error(fmt.Sprintf("Не удалось прочитать PID файл: %v", err))
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		log.Error(fmt.Sprintf("Не удалось преобразовать PID из файла: %v", err))
		return
	}

	// Находим процесс по PID
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Error(fmt.Sprintf("Не удалось найти процесс с PID %d: %v", pid, err))
		return
	}

	// Отправляем сигнал завершения
	log.Info(fmt.Sprintf("Отправка сигнала SIGTERM процессу с PID %d", pid))
	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Error(fmt.Sprintf("Не удалось отправить сигнал SIGTERM процессу %d: %v", pid, err))
	} else {
		log.Info(fmt.Sprintf("Сигнал SIGTERM успешно отправлен процессу %d", pid))
	}

	// Удаляем PID файл после попытки завершения
	if err := os.Remove(pidPath); err != nil {
		log.Error(fmt.Sprintf("Не удалось удалить PID файл: %v", err))
	}
}
