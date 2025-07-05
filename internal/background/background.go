package background

import (
	"fmt"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/monitor"
	"macbat/internal/paths"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//================================================================================
// ЭКСПОРТИРУЕМЫЕ ФУНКЦИИ
//================================================================================

// IsGUIRunning проверяет, запущен ли GUI процесс, по lock-файлу.
func IsGUIRunning(log *logger.Logger) bool {
	lockFile := paths.GUILockPath()
	pidBytes, err := os.ReadFile(lockFile)
	if err != nil {
		return false // Файл не найден, GUI не запущен.
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		log.Info(fmt.Sprintf("Поврежденный lock-файл GUI, удаляем: %s", lockFile))
		_ = os.Remove(lockFile) // Поврежденный файл, удаляем.
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Info(fmt.Sprintf("Процесс GUI не найден, удаляем устаревший lock-файл: %s", lockFile))
		_ = os.Remove(lockFile) // Процесс не найден, устаревший файл.
		return false
	}
	// Сигнал 0 проверяет существование процесса.
	if err = process.Signal(syscall.Signal(0)); err == nil {
		return true // Процесс существует.
	}
	log.Info(fmt.Sprintf("Процесс GUI не отвечает, удаляем устаревший lock-файл: %s", lockFile))
	_ = os.Remove(lockFile) // Процесс не существует, устаревший файл.
	return false
}

// LaunchDetached запускает копию приложения с указанным флагом в отсоединенном режиме.
func LaunchDetached(flag string, log *logger.Logger) {
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

// Run - это основная логика приложения, которая работает в фоне.
func Run(log *logger.Logger, cfg *config.Config, cfgManager *config.Manager, mode string) {
	log.Info("Фоновый процесс проверки заряда батареи начал работу.")

	// Настраиваем канал для обработки сигналов завершения.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создаем и запускаем монитор.
	appMonitor := monitor.NewMonitor(cfg, cfgManager, log)

	monitorStarted := make(chan struct{})

	go func() {
		close(monitorStarted) // Сообщаем, что монитор запускается.
		appMonitor.Start(mode)
	}()

	// Ждем, пока монитор действительно начнет работу.
	<-monitorStarted
	log.Info("Монитор успешно запущен (appMonitor.Start).")

	sig := <-sigChan
	log.Info(fmt.Sprintf("Получен сигнал %v, завершаю работу...", sig))

	// Корректно останавливаем монитор.
	appMonitor.Stop()

	// Даем время на завершение всех горутин.
	time.Sleep(500 * time.Millisecond)
}

// IsRunning проверяет, запущен ли фоновый процесс, по PID-файлу.
func IsRunning(log *logger.Logger) bool {
	pidPath := paths.PIDBackgoundPath()
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		// Если файл не читается, считаем, что процесс не запущен.
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		log.Info(fmt.Sprintf("Поврежденный PID-файл, удаляем: %s", pidPath))
		_ = os.Remove(pidPath)
		return false
	}

	// Проверяем, существует ли процесс с таким PID.
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Info(fmt.Sprintf("Процесс не найден, удаляем устаревший PID-файл: %s", pidPath))
		_ = os.Remove(pidPath)
		return false
	}

	// Отправка сигнала 0 - это стандартный способ проверить существование процесса в Unix-системах.
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		log.Info(fmt.Sprintf("Процесс не отвечает, удаляем устаревший PID-файл: %s", pidPath))
		_ = os.Remove(pidPath)
	}
	return err == nil
}

// WritePID записывает PID текущего процесса в файл.
func WritePID(log *logger.Logger) error {
	pidPath := paths.PIDBackgoundPath()
	pid := os.Getpid()
	log.Info(fmt.Sprintf("Запись PID %d в файл: %s", pid, pidPath))

	// Создаем директорию, если она не существует.
	dir := filepath.Dir(pidPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("не удалось создать директорию для PID файла: %w", err)
		}
	}

	// Записываем PID в файл.
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("не удалось записать в PID файл: %w", err)
	}
	return nil
}

// Kill находит и завершает указанный процесс (background или gui).
// В качестве аргумента принимает логгер и тип процесса: "background" или "gui".
func Kill(log *logger.Logger, processType string) {
	var pidPath string

	// Определяем путь к PID/Lock файлу в зависимости от типа процесса.
	switch processType {
	case "--background":
		pidPath = paths.PIDBackgoundPath()
	case "--gui-agent":
		pidPath = paths.GUILockPath()
	default:
		log.Error(fmt.Sprintf("Неизвестный тип процесса для завершения: %s", processType))
		return
	}

	log.Info(fmt.Sprintf("Попытка завершить процесс типа '%s', используя PID файл: %s", processType, pidPath))

	// Читаем PID из файла.
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info(fmt.Sprintf("PID файл для процесса '%s' не найден. Процесс, вероятно, не запущен.", processType))
		} else {
			log.Error(fmt.Sprintf("Не удалось прочитать PID файл для процесса '%s': %v", processType, err))
		}
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		log.Error(fmt.Sprintf("Неверный формат PID в файле для процесса '%s': %v", processType, err))
		_ = os.Remove(pidPath)
		return
	}

	// Находим процесс по PID.
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Error(fmt.Sprintf("Не удалось найти процесс '%s' с PID %d: %v", processType, pid, err))
		_ = os.Remove(pidPath)
		return
	}

	// Отправляем сигнал завершения.
	log.Info(fmt.Sprintf("Отправка сигнала SIGTERM процессу '%s' с PID %d", processType, pid))
	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Error(fmt.Sprintf("Не удалось отправить сигнал SIGTERM процессу %d: %v", pid, err))
	} else {
		log.Info(fmt.Sprintf("Сигнал SIGTERM успешно отправлен процессу %d", pid))
	}

	// Удаляем PID файл после отправки сигнала.
	if err := os.Remove(pidPath); err != nil {
		log.Error(fmt.Sprintf("Не удалось удалить PID файл для процесса '%s': %v", processType, err))
	} else {
		log.Info(fmt.Sprintf("PID файл для процесса '%s' успешно удален.", processType))
	}
}
