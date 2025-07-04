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
	"golang.org/x/term"

	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
)

var log *logger.Logger

// launchDetached запускает копию приложения с указанным флагом в отсоединенном режиме.
func launchDetached(flag string) {
	log.Info(fmt.Sprintf("Запуск отсоединенного процесса с флагом: %s", flag))

	cmd := exec.Command(paths.BinaryPath(), flag)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		log.Fatal(fmt.Sprintf("Не удалось запустить процесс с флагом %s: %v", flag, err))
	}
	log.Info(fmt.Sprintf("Процесс с флагом %s успешно запущен с PID: %d", flag, cmd.Process.Pid))
}

// isGUIRunning проверяет, запущен ли GUI процесс, по lock-файлу.
func isGUIRunning() bool {
	lockFile := paths.GUILockPath()
	pidBytes, err := os.ReadFile(lockFile)
	if err != nil {
		return false // Файл не найден, GUI не запущен.
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		_ = os.Remove(lockFile) // Поврежденный файл, удаляем.
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(lockFile) // Процесс не найден, устаревший файл.
		return false
	}
	// Сигнал 0 проверяет существование процесса.
	if err = process.Signal(syscall.Signal(0)); err == nil {
		return true // Процесс существует.
	}
	_ = os.Remove(lockFile) // Процесс не существует, устаревший файл.
	return false
}

func main() {
	// Инициализация логгера
	log = logger.New(paths.LogPath(), 100, true, false)

	// --- Инициализация конфигурации ---
	cfgManager, err := config.New(log, paths.ConfigPath())
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка инициализации менеджера конфигурации: %v", err))
	}
	conf, err := cfgManager.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка загрузки конфигурации: %v", err))
	}

	// --- Обработка флагов командной строки ---
	installFlag := flag.Bool("install", false, "Установить приложение и агент launchd")
	uninstallFlag := flag.Bool("uninstall", false, "Удалить приложение и агент launchd")
	backgroundFlag := flag.Bool("background", false, "Запуск фонового процесса мониторинга")
	guiAgentFlag := flag.Bool("gui-agent", false, "Внутренний флаг для запуска GUI агента")
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

	// --- Логика фонового процесса ---
	if *backgroundFlag {
		if isBackgroundRunning() {
			log.Info("Фоновый процесс уже запущен. Выход.")
			return
		}
		if term.IsTerminal(int(os.Stdout.Fd())) {
			launchDetached("--background")
			log.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")
			return
		}
		if err := writePID(); err != nil {
			log.Error(fmt.Sprintf("Не удалось записать PID файла: %v", err))
		}
		log.Info("Запуск в фоновом режиме...")
		runBackgroundMainTask(conf, cfgManager, "run")
		_ = os.Remove(paths.PIDPath())
		return
	}

	// --- Логика GUI Агента ---
	if *guiAgentFlag {
		log.Info("Запуск агента GUI (иконка в трее)...")
		// Создаем lock-файл, так как этот процесс теперь главный для GUI.
		_ = os.WriteFile(paths.GUILockPath(), []byte(strconv.Itoa(os.Getpid())), 0644)

		// Запускаем фоновый процесс, если он еще не запущен
		if !isBackgroundRunning() {
			log.Info("Запуск фонового процесса мониторинга батареи...")
			launchDetached("--background")
		} else {
			log.Info("Фоновый процесс уже запущен.")
		}
		// Запускаем блокирующий цикл GUI
		systray.Run(onReady, onExit)
		return
	}

	// --- Логика Лаунчера (запуск без флагов) ---
	log.Info("Запуск приложения (режим лаунчера)...")
	if isGUIRunning() {
		log.Info("Приложение уже запущено. Выход.")
		return
	}

	log.Info("Запуск GUI агента...")
	launchDetached("--gui-agent")
	log.Info("Приложение успешно запущено в фоновом режиме. Лаунчер завершает работу.")
}


func onExit() {
	// Здесь можно выполнить очистку перед выходом
	log.Info("Выход из приложения")
	// Удаляем lock-файл GUI перед выходом
	_ = os.Remove(paths.GUILockPath())
	os.Exit(0)
}

// isBackgroundRunning проверяет, запущен ли фоновый процесс, по PID-файлу.
func isBackgroundRunning() bool {
	pidPath := paths.PIDPath()
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		// Если файл не читается, считаем, что процесс не запущен.
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		// Некорректный PID в файле.
		return false
	}

	// Проверяем, существует ли процесс с таким PID.
	// Отправка сигнала 0 - это стандартный способ проверить существование процесса в Unix-системах.
	process, err := os.FindProcess(pid)
	if err != nil {
		// Процесс не найден.
		return false
	}

	// Если err == nil, сигнал был успешно отправлен (или у нас нет прав, но процесс существует).
	// Если err == os.ErrProcessDone, процесс уже завершился.
	err = process.Signal(syscall.Signal(0))
	return err == nil
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
