// paths/paths.go
// Модуль для получения путей к часто используемым файлам.

package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// AppName - это константа, определяющая имя приложения.
const AppName = "macbat"

// BinaryPath возвращает путь к бинарному файлу приложения.
// @return string - путь к бинарнику
func BinaryPath() string {
	// os.Executable() возвращает полный путь к текущему исполняемому файлу.
	// Это именно то, что нужно для запуска копии процесса.
	binPath, err := os.Executable()
	if err != nil {
		// В случае ошибки возвращаем базовое имя, предполагая, что оно в PATH.
		return AppName
	}
	return binPath
}

// ConfigPath возвращает путь к файлу конфигурации.
// Мы будем хранить config.json рядом с исполняемым файлом для портативности.
// @return string - путь к config.json
func ConfigPath() string {
	// Получаем путь к исполняемому файлу
	exePath, err := os.Executable()
	if err != nil {
		// В случае ошибки, возвращаем путь в домашней директории как запасной вариант
		return filepath.Join(os.Getenv("HOME"), ".config", AppName, "config.json")
	}
	// Возвращаем путь к config.json в той же директории, что и бинарник
	return filepath.Join(filepath.Dir(exePath), "config.json")
}

// LogDir возвращает путь к директории логов.
// @return string - путь к директории логов
func LogDir() string {
	// Для macOS предпочтительнее использовать ~/Library/Logs
	logDir := filepath.Join(os.Getenv("HOME"), "Library", "Logs", AppName)
	_ = os.MkdirAll(logDir, 0755)
	return logDir
}

// ensureLogDir создает директорию логов, если она не существует.
// @return error - ошибка, если не удалось создать директорию
func ensureLogDir() error {
	logDir := LogDir()
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return os.MkdirAll(logDir, 0755)
	}
	return nil
}

// LogPath возвращает путь к файлу логов.
// @return string - путь к macbat.log
func LogPath() string {
	_ = ensureLogDir() // Игнорируем ошибку, так как это вызовется при каждой записи в лог
	return filepath.Join(LogDir(), AppName+".log")
}

// PlistPath возвращает путь к файлу plist для launchd.
// @return string - путь к com.macbat.plist
func PlistPath() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", AgentIdentifier()+".plist")
}

// ErrorLogPath возвращает путь к файлу ошибок.
// @return string - путь к macbat.err
func ErrorLogPath() string {
	return "/tmp/" + AppName + ".err"
}

// AgentIdentifier возвращает идентификатор агента для launchd.
// @return string - идентификатор агента
func AgentIdentifier() string {
	return "com." + AppName + ".agent"
}

// PIDPath возвращает путь к файлу PID для указанного типа процесса.
// @param processType - тип процесса (например, "--background" или "--gui-agent").
// @return string - путь к PID-файлу.
func PIDPath(processType string) string {
	// Удаляем префиксы, чтобы имя файла было чище
	cleanProcessType := strings.TrimPrefix(processType, "--")
	return filepath.Join(os.TempDir(), AppName+"."+cleanProcessType+".pid")
}

// LockPath возвращает путь к файлу блокировки для указанного типа процесса.
// @param processType - тип процесса (например, "--background" или "--gui-agent").
// @return string - путь к lock-файлу.
func LockPath(processType string) string {
	// Удаляем префиксы, чтобы имя файла было чище
	cleanProcessType := strings.TrimPrefix(processType, "--")
	return filepath.Join(os.TempDir(), AppName+"."+cleanProcessType+".lock")
}
