// internal/interfaces/interfaces.go
// Файл содержит все интерфейсы приложения для обеспечения dependency injection,
// улучшения тестируемости и создания слабо связанной архитектуры
package interfaces

import (
	"context"

	"github.com/qzeleza/macbat/internal/config"
)

// ------------------------------------------------------------------
// Интерфейс для логирования
// ------------------------------------------------------------------

// Logger определяет контракт для системы логирования
// Позволяет легко заменять реализацию логгера и создавать моки для тестов
type Logger interface {
	// Debug записывает отладочное сообщение
	Debug(message string)
	
	// Info записывает информационное сообщение
	Info(message string)
	
	// Error записывает сообщение об ошибке
	Error(message string)
	
	// Fatal записывает критическое сообщение и завершает программу
	Fatal(message string)
	
	// Line добавляет разделительную линию в лог
	Line()
	
	// SetLevel устанавливает уровень логирования
	SetLevel(level string)
	
	// Close закрывает лог-файл и освобождает ресурсы
	Close() error
}

// ------------------------------------------------------------------
// Интерфейс для управления фоновыми процессами
// ------------------------------------------------------------------

// BackgroundManager определяет контракт для управления фоновыми процессами
// Инкапсулирует логику запуска, остановки и мониторинга процессов
type BackgroundManager interface {
	// IsRunning проверяет, запущен ли процесс с заданным именем
	IsRunning(processName string) bool
	
	// LaunchDetached запускает процесс в отсоединенном режиме
	LaunchDetached(processName string) error
	
	// Run запускает задачу в фоновом режиме с управлением жизненным циклом
	Run(processName string, task func()) error
	
	// Lock создает блокировку для предотвращения множественного запуска
	Lock(processName string) error
	
	// Unlock освобождает блокировку процесса
	Unlock(processName string)
	
	// WritePID записывает PID процесса в файл
	WritePID(processName string) error
	
	// HandleSignals настраивает обработку системных сигналов
	HandleSignals(processName string)
	
	// Kill принудительно завершает процесс
	Kill(processName string) error
	
	// GetPID возвращает PID процесса
	GetPID(processName string) (int, error)
}

// ------------------------------------------------------------------
// Интерфейс для управления конфигурацией
// ------------------------------------------------------------------

// ConfigManager определяет контракт для управления конфигурацией приложения
// Обеспечивает загрузку, сохранение и мониторинг изменений конфигурации
type ConfigManager interface {
	// Load загружает конфигурацию из файла
	Load() (*config.Config, error)
	
	// Save сохраняет конфигурацию в файл
	Save(config *config.Config) error
	
	// Reload перезагружает конфигурацию из файла
	Reload() (*config.Config, error)
	
	// Watch запускает мониторинг изменений файла конфигурации
	Watch(ctx context.Context, callback func(*config.Config)) error
	
	// Validate проверяет корректность конфигурации
	Validate(config *config.Config) error
	
	// GetPath возвращает путь к файлу конфигурации
	GetPath() string
}

// ------------------------------------------------------------------
// Интерфейс для мониторинга батареи
// ------------------------------------------------------------------

// Monitor определяет контракт для мониторинга состояния батареи
// Инкапсулирует логику отслеживания заряда и отправки уведомлений
type Monitor interface {
	// Start запускает мониторинг батареи
	Start(mode string, started chan<- struct{})
	
	// Stop останавливает мониторинг батареи
	Stop() error
	
	// GetStatus возвращает текущее состояние батареи
	GetStatus() (string, error)
	
	// IsHealthy проверяет, работает ли монитор корректно
	IsHealthy() bool
	
	// GetBatteryLevel возвращает текущий уровень заряда батареи
	GetBatteryLevel() (int, error)
	
	// SetThresholds устанавливает пороговые значения для уведомлений
	SetThresholds(low, critical int) error
}

// ------------------------------------------------------------------
// Интерфейс для установки/удаления приложения
// ------------------------------------------------------------------

// Installer определяет контракт для установки и удаления приложения
// Обеспечивает управление жизненным циклом приложения в системе
type Installer interface {
	// Install выполняет установку приложения в систему
	Install(ctx context.Context) error
	
	// Uninstall выполняет удаление приложения из системы
	Uninstall(ctx context.Context) error
	
	// IsInstalled проверяет, установлено ли приложение
	IsInstalled() bool
	
	// GetInstallationPath возвращает путь установки приложения
	GetInstallationPath() string
	
	// CreatePlist создает файл конфигурации для launchd
	CreatePlist() error
	
	// RemovePlist удаляет файл конфигурации launchd
	RemovePlist() error
}

// ------------------------------------------------------------------
// Интерфейс для GUI приложения в системном трее
// ------------------------------------------------------------------

// TrayApp определяет контракт для GUI приложения в системном трее
// Управляет отображением и взаимодействием с пользователем через трей
type TrayApp interface {
	// Start запускает GUI приложение в системном трее
	Start() error
	
	// Stop останавливает GUI приложение
	Stop() error
	
	// UpdateIcon обновляет иконку в трее в соответствии с уровнем батареи
	UpdateIcon(batteryLevel int) error
	
	// ShowNotification отображает уведомление пользователю
	ShowNotification(message string) error
	
	// SetMenu устанавливает контекстное меню для иконки в трее
	SetMenu(items []MenuItem) error
	
	// IsVisible возвращает true, если иконка видна в трее
	IsVisible() bool
}

// ------------------------------------------------------------------
// Вспомогательные типы для интерфейсов
// ------------------------------------------------------------------

// MenuItem представляет элемент контекстного меню в системном трее
type MenuItem struct {
	// Title - заголовок элемента меню
	Title string
	
	// Action - функция, вызываемая при нажатии на элемент
	Action func() error
	
	// Enabled - доступен ли элемент для взаимодействия
	Enabled bool
	
	// Separator - является ли элемент разделителем
	Separator bool
}

// ------------------------------------------------------------------
// Интерфейс для работы с системными путями
// ------------------------------------------------------------------

// PathProvider определяет контракт для получения системных путей
// Абстрагирует платформо-зависимую логику работы с путями
type PathProvider interface {
	// GetConfigPath возвращает путь к файлу конфигурации
	GetConfigPath() string
	
	// GetLogPath возвращает путь к файлу логов
	GetLogPath() string
	
	// GetBinaryPath возвращает путь к исполняемому файлу
	GetBinaryPath() string
	
	// GetPlistPath возвращает путь к файлу plist для launchd
	GetPlistPath() string
	
	// GetTempPath возвращает путь к временной директории
	GetTempPath() string
	
	// EnsureDirectories создает необходимые директории, если они не существуют
	EnsureDirectories() error
}

// ------------------------------------------------------------------
// Интерфейс для управления средой выполнения
// ------------------------------------------------------------------

// EnvironmentManager определяет контракт для управления переменными окружения
// Обеспечивает кроссплатформенную работу с переменными среды
type EnvironmentManager interface {
	// AddToPath добавляет директорию в переменную PATH
	AddToPath(directory string) error
	
	// RemoveFromPath удаляет директорию из переменной PATH
	RemoveFromPath(directory string) error
	
	// UpdateShell обновляет текущую сессию shell
	UpdateShell() error
	
	// GetEnv возвращает значение переменной окружения
	GetEnv(key string) string
	
	// SetEnv устанавливает значение переменной окружения
	SetEnv(key, value string) error
}
