/**
 * @file background.go
 * @brief Управление фоновыми процессами и их жизненным циклом.
 *
 * Этот пакет предоставляет инструменты для запуска, остановки и проверки состояния
 * фоновых процессов приложения, таких как мониторинг батареи или GUI-агент.
 * Он использует lock-файлы для предотвращения одновременного запуска нескольких
 * экземпляров одного и того же процесса.
 *
 * @author Zeleza
 * @date 2025-07-20
 */

package background

import (
	"fmt"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

// Manager управляет фоновыми процессами, обеспечивая их уникальность с помощью lock-файлов.
// Он также обрабатывает сигналы для корректного завершения.
type Manager struct {
	log      *logger.Logger
	lockFile *os.File // Хранит дескриптор lock-файла
}

// New создает новый экземпляр Manager.
func New(log *logger.Logger) *Manager {
	return &Manager{
		log: log,
	}
}

// LaunchDetached запускает текущее приложение в фоновом режиме с указанными аргументами.
func (m *Manager) LaunchDetached(args ...string) {
	binaryPath := paths.BinaryPath()
	// paths.BinaryPath() возвращает имя приложения, если не может получить путь.
	// Проверяем, что путь абсолютный, чтобы убедиться, что мы получили корректный путь.
	if !filepath.IsAbs(binaryPath) {
		m.log.Fatal(fmt.Sprintf("Не удалось получить абсолютный путь к исполняемому файлу, получен: '%s'. Убедитесь, что приложение находится в PATH или запускается с указанием полного пути.", binaryPath))
	}

	cmd := exec.Command(binaryPath, args...)
	// Отсоединяем процесс от текущего терминала
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось запустить отсоединенный процесс с аргументами %v: %v", args, err))
		return
	}

	m.log.Info(fmt.Sprintf("Процесс успешно запущен в фоновом режиме с PID %d и аргументами: %v", cmd.Process.Pid, args))
	// Освобождаем ресурсы, связанные с дочерним процессом, в родительском процессе
	_ = cmd.Process.Release()
}

// Run выполняет задачу, удерживая блокировку. Этот метод является блокирующим.
func (m *Manager) Run(mode string, task func()) error {
	if err := m.Lock(mode); err != nil {
		return fmt.Errorf("процесс '%s' уже запущен или произошла ошибка блокировки: %w", mode, err)
	}
	defer m.Unlock(mode)

	if err := m.WritePID(mode); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось записать PID для режима %s: %v", mode, err))
	}
	defer removePID(mode)

	m.HandleSignals(mode)

	task() // Блокирующий вызов

	return nil
}

// IsRunning проверяет, запущен ли процесс, путем проверки lock-файла.
func (m *Manager) IsRunning(mode string) bool {
	lockPath := paths.LockPath(mode)
	file, err := os.Open(lockPath)
	if err != nil {
		return false // Файла нет, значит не запущен
	}
	defer file.Close()

	// Пытаемся заблокировать файл. Если не получается, значит он уже заблокирован другим процессом.
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return true // Не удалось заблокировать, значит процесс запущен
	}

	// Если удалось заблокировать, значит процесс не был запущен. Сразу же разблокируем.
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return false
}

// Kill завершает процесс, идентифицированный по его типу (режиму).
func (m *Manager) Kill(mode string) error {
	pidPath := paths.PIDPath(mode)
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			// PID файла нет, возможно, процесс уже завершился. Попробуем удалить lock-файл на всякий случай.
			m.log.Info(fmt.Sprintf("PID-файл для '%s' не найден, процесс, вероятно, не запущен. Попытка очистки...", mode))
			m.Unlock(mode)
			return nil
		}
		return fmt.Errorf("не удалось прочитать PID-файл для '%s': %w", mode, err)
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return fmt.Errorf("неверный формат PID в файле для '%s': %w", mode, err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		// Если процесс не найден, возможно, он уже завершился. Просто чистим файлы.
		m.log.Info(fmt.Sprintf("Процесс с PID %d для '%s' не найден, возможно, он уже завершен. Очистка...", pid, mode))
		m.Unlock(mode)
		removePID(mode)
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Проверяем, не является ли ошибка следствием того, что процесс уже завершен
		if err == os.ErrProcessDone {
			m.log.Info(fmt.Sprintf("Процесс с PID %d для '%s' уже был завершен. Очистка...", pid, mode))
			m.Unlock(mode)
			removePID(mode)
			return nil
		}
		return fmt.Errorf("не удалось отправить сигнал завершения процессу с PID %d для '%s': %w", pid, mode, err)
	}

	m.log.Info(fmt.Sprintf("Сигнал завершения отправлен процессу '%s' (PID: %d)", mode, pid))
	// Файлы будут удалены обработчиком сигнала в самом процессе.
	return nil
}

// Lock создает и блокирует lock-файл, сохраняя его дескриптор.
func (m *Manager) Lock(mode string) error {
	lockPath := paths.LockPath(mode)
	file, err := os.Create(lockPath)
	if err != nil {
		return fmt.Errorf("не удалось создать lock-файл '%s': %w", lockPath, err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close() // Закрываем файл перед возвратом ошибки
		return fmt.Errorf("не удалось заблокировать lock-файл '%s', возможно, процесс уже запущен: %w", lockPath, err)
	}

	m.lockFile = file // Сохраняем дескриптор
	return nil
}

// Unlock снимает блокировку, закрывает и удаляет lock-файл.
func (m *Manager) Unlock(mode string) {
	if m.lockFile == nil {
		// Если lock-файл не был установлен (например, при вызове Kill из другого процесса),
		// попытаемся удалить его по пути. Это не гарантирует разблокировку, но удаляет файл.
		lockPath := paths.LockPath(mode)
		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			m.log.Error(fmt.Sprintf("Не удалось удалить lock-файл '%s' (без дескриптора): %v", lockPath, err))
		}
		return
	}

	lockPath := m.lockFile.Name()
	// Сначала снимаем блокировку
	if err := syscall.Flock(int(m.lockFile.Fd()), syscall.LOCK_UN); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось разблокировать lock-файл '%s': %v", lockPath, err))
	}
	// Затем закрываем файл
	if err := m.lockFile.Close(); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось закрыть lock-файл '%s': %v", lockPath, err))
	}
	// Наконец, удаляем файл
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		m.log.Error(fmt.Sprintf("Не удалось удалить lock-файл '%s': %v", lockPath, err))
	}

	m.lockFile = nil // Сбрасываем дескриптор
}

// WritePID записывает PID текущего процесса в PID-файл.
func (m *Manager) WritePID(mode string) error {
	pidPath := paths.PIDPath(mode)
	pid := os.Getpid()
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		m.log.Error(fmt.Sprintf("Не удалось записать PID-файл в '%s': %v", pidPath, err))
		return err
	}
	m.log.Info(fmt.Sprintf("PID %d записан в %s", pid, pidPath))
	return nil
}

// HandleSignals обрабатывает сигналы завершения для корректного освобождения ресурсов.
func (m *Manager) HandleSignals(mode string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Эта горутина будет ждать сигнала от ОС
		sig := <-sigChan
		m.log.Info(fmt.Sprintf("Получен сигнал '%v' для процесса '%s'. Завершение...", sig, mode))
		m.Unlock(mode)
		removePID(mode)
		os.Exit(0)
	}()
}

// removePID удаляет PID-файл.
func removePID(processType string) {
	pidPath := paths.PIDPath(processType)
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		// В этой функции логгер недоступен.
		fmt.Fprintf(os.Stderr, "Не удалось удалить PID-файл '%s': %v\n", pidPath, err)
	}
}
