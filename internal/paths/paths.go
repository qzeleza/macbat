// paths/paths.go
// Модуль для получения путей к часто используемым файлам.

package paths

import (
	"os"
	"path/filepath"
)

const AppName = "macbat"

// BinaryPath возвращает путь к бинарному файлу приложения.
// @return string - путь к бинарнику
func BinaryPath() string {
	return filepath.Join(os.Getenv("HOME"), "bin", AppName)
}

// ConfigPath возвращает путь к файлу конфигурации.
// @return string - путь к config.json
func ConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", AppName, "config.json")
}

// LogDir возвращает путь к директории логов.
// @return string - путь к директории логов
func LogDir() string {
	return "/tmp"
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
	return filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com."+AppName+".agent.plist")
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
