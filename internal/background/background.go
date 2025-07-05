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
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"macbat/internal/logger"
	"macbat/internal/paths"
)

//================================================================================
// СТРУКТУРЫ ДАННЫХ
//================================================================================

// Manager управляет фоновыми процессами приложения.
// @property log - логгер для записи событий.
type Manager struct {
	log      *logger.Logger // Логгер для вывода сообщений.
	stopChan chan struct{}   // Канал для graceful shutdown.
}

// New создает новый экземпляр Manager.
//
// @param log *logger.Logger - логгер для записи событий.
// @return *Manager - новый экземпляр Manager.
func New(log *logger.Logger) *Manager {
	return &Manager{
		log:      log,
		stopChan: make(chan struct{}),
	}
}

//================================================================================
// ОСНОВНЫЕ МЕТОДЫ
//================================================================================

// LaunchDetached запускает новый экземпляр приложения в отсоединенном режиме.
//
// @param processType Строковый флаг, указывающий, какой процесс запустить (например, "--background").
func (m *Manager) LaunchDetached(processType string) {
	binPath := paths.BinaryPath()
	if binPath == paths.AppName {
		// Это означает, что os.Executable() вернул ошибку, и было возвращено имя по умолчанию.
		m.log.Error(fmt.Sprintf("Не удалось получить полный путь к исполняемому файлу, используется '%s'. Убедитесь, что он находится в PATH.", binPath))
	}

	cmd := exec.Command(binPath, processType)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Отсоединяем от текущей сессии

	if err := cmd.Start(); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось запустить отсоединенный процесс '%s': %v", processType, err))
		return
	}

	m.log.Info(fmt.Sprintf("Процесс '%s' успешно запущен в фоновом режиме с PID %d.", processType, cmd.Process.Pid))
	// Важно: не ждем завершения процесса, чтобы родитель мог завершиться.
	_ = cmd.Process.Release()
}

// Run выполняет задачу, удерживая блокировку для указанного типа процесса.
// Этот метод является блокирующим и завершится только после выполнения переданной задачи.
//
// @param processType Строковый идентификатор процесса (например, "--background").
// @param task Функция, содержащая основную логику процесса.
// @return Ошибка, если процесс уже запущен или не удалось создать блокировку.
func (m *Manager) Run(processType string, task func()) error {
	// 1. Попытка заблокировать lock-файл.
	lockFile, err := m.lock(processType)
	if err != nil {
		return fmt.Errorf("процесс '%s' уже запущен или произошла ошибка блокировки: %w", processType, err)
	}
	// Гарантируем разблокировку и очистку при выходе из функции.
	defer m.unlock(lockFile)

	// 2. Запись PID.
	if err := m.writePID(processType); err != nil {
		// Не фатально, но стоит залогировать.
		m.log.Info(fmt.Sprintf("Не удалось записать PID-файл для '%s': %v", processType, err))
	}
	// Гарантируем удаление PID-файла при выходе.
	defer m.removePID(processType)

	m.log.Info(fmt.Sprintf("Процесс '%s' успешно запущен и заблокирован.", processType))

	// 3. Установка обработчика сигналов для корректного завершения.
	m.handleSignals(processType)

	// 4. Выполнение основной задачи, переданной в параметре.
	go func() {
		defer func() {
			// После завершения задачи отправляем сигнал в stopChan, если он еще не закрыт.
			// Используем select для неблокирующей проверки, чтобы избежать паники при двойном закрытии.
			select {
			case <-m.stopChan:
				// Канал уже закрыт, ничего не делаем.
			default:
				close(m.stopChan)
			}
		}()
		if task != nil {
			task()
		}
	}()

	// Ожидаем сигнала о завершении (от задачи или от обработчика сигналов).
	<-m.stopChan
	m.log.Info(fmt.Sprintf("Задача процесса '%s' завершена. Снятие блокировки.", processType))

	return nil
}

// IsRunning проверяет, запущен ли процесс указанного типа, путем проверки lock-файла.
//
// @param processType Строковый идентификатор процесса (например, "--background").
// @return true, если процесс запущен, иначе false.
func (m *Manager) IsRunning(processType string) bool {
	lockPath := paths.LockPath(processType)
	file, err := os.Open(lockPath)
	if err != nil {
		return false // Файла нет, значит, процесс не запущен.
	}
	defer file.Close()

	// Пытаемся заблокировать файл. Если удалось (err == nil), значит, он не заблокирован другим процессом.
	// В этом случае процесс не запущен, и мы тут же снимаем блокировку.
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		return false
	}

	return true
}

// Kill отправляет сигнал завершения процессу по его PID из PID-файла.
//
// @param processType Строковый идентификатор процесса.
// @return Ошибка, если не удалось прочитать PID или отправить сигнал.
func (m *Manager) Kill(processType string) error {
	pidPath := paths.PIDPath(processType)
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать PID-файл для '%s': %w", processType, err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return fmt.Errorf("некорректный PID в файле '%s': %w", pidPath, err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		// Процесс может не существовать, если он уже завершился.
		// Это не всегда ошибка, но мы возвращаем ее для информации.
		return fmt.Errorf("не удалось найти процесс с PID %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Если процесс уже завершен, os.FindProcess его находит, но Signal возвращает ошибку.
		// Мы можем проверить ошибку, чтобы не считать это сбоем.
		if strings.Contains(err.Error(), "process already finished") {
			m.log.Info(fmt.Sprintf("Процесс '%s' (PID: %d) уже был завершен.", processType, pid))
			// Очищаем файлы, так как процесс мертв
			m.removePID(processType)
			lockPath := paths.LockPath(processType)
			_ = os.Remove(lockPath)
			return nil
		}
		return fmt.Errorf("не удалось отправить сигнал завершения процессу с PID %d: %w", pid, err)
	}

	m.log.Info(fmt.Sprintf("Сигнал завершения отправлен процессу '%s' (PID: %d).", processType, pid))
	return nil
}

//================================================================================
// ВНУТРЕННИЕ МЕТОДЫ
//================================================================================

// writePID записывает PID текущего процесса в файл.
func (m *Manager) writePID(processType string) error {
	pidPath := paths.PIDPath(processType)
	pid := os.Getpid()
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		m.log.Error(fmt.Sprintf("Не удалось записать PID-файл в '%s': %v", pidPath, err))
		return err
	}
	m.log.Info(fmt.Sprintf("PID %d записан в %s", pid, pidPath))
	return nil
}

// removePID удаляет PID-файл.
func (m *Manager) removePID(processType string) {
	pidPath := paths.PIDPath(processType)
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		m.log.Info(fmt.Sprintf("Не удалось удалить PID-файл '%s': %v", pidPath, err))
	} else {
		m.log.Info(fmt.Sprintf("PID-файл '%s' удален.", pidPath))
	}
}

// lock пытается создать и заблокировать lock-файл.
func (m *Manager) lock(processType string) (*os.File, error) {
	lockPath := paths.LockPath(processType)
	file, err := os.Create(lockPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать lock-файл '%s': %w", lockPath, err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("не удалось заблокировать lock-файл '%s', возможно, процесс уже запущен: %w", lockPath, err)
	}

	return file, nil
}

// unlock снимает блокировку и удаляет lock-файл.
func (m *Manager) unlock(file *os.File) {
	if file == nil {
		return
	}
	lockPath := file.Name() // Получаем путь из самого файла
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось разблокировать lock-файл '%s': %v", lockPath, err))
	}
	if err := file.Close(); err != nil {
		m.log.Error(fmt.Sprintf("Не удалось закрыть lock-файл '%s': %v", lockPath, err))
	}
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		m.log.Error(fmt.Sprintf("Не удалось удалить lock-файл '%s': %v", lockPath, err))
	}
}

// handleSignals настраивает обработку системных сигналов для graceful shutdown.
func (m *Manager) handleSignals(processType string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		m.log.Info(fmt.Sprintf("Получен сигнал '%v' для процесса '%s'. Завершение...", sig, processType))
		// Используем select для неблокирующей проверки, чтобы избежать паники при двойном закрытии.
		select {
		case <-m.stopChan:
			// Канал уже закрыт, ничего не делаем.
		default:
			close(m.stopChan)
		}
	}()
}
