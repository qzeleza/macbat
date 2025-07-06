package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qzeleza/macbat/internal/logger"
)

// addToPath добавляет директорию в переменную PATH в файле конфигурации оболочки
// и обновляет текущую сессию
// AddToPath добавляет директорию в переменную PATH в файле конфигурации оболочки
// и обновляет текущую сессию
func AddToPath(path string, log *logger.Logger) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("не удалось определить домашнюю директорию: %v", err)
	}

	// Определяем файл конфигурации оболочки
	var configFile string
	var shellName string
	shell := os.Getenv("SHELL")

	switch filepath.Base(shell) {
	case "zsh":
		configFile = filepath.Join(homeDir, ".zshrc")
		shellName = "zsh"
	case "bash":
		configFile = filepath.Join(homeDir, ".bash_profile")
		shellName = "bash"
	default:
		// По умолчанию используем .zshrc для macOS
		configFile = filepath.Join(homeDir, ".zshrc")
		shellName = "zsh"
	}

	// Проверяем, существует ли файл конфигурации
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Создаем файл, если он не существует
		if err := os.WriteFile(configFile, []byte{}, 0644); err != nil {
			return fmt.Errorf("не удалось создать файл конфигурации: %v", err)
		}
	}

	// Читаем содержимое файла
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл конфигурации: %v", err)
	}

	// Проверяем, не добавлен ли уже путь
	pathEntry := fmt.Sprintf("\nexport PATH=\"$PATH:%s\"\n", path)
	pathAdded := false

	if !strings.Contains(string(data), pathEntry) {
		// Добавляем путь в конец файла
		f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("не удалось открыть файл конфигурации для записи: %v", err)
		}

		if _, err := f.WriteString(fmt.Sprintf("\n# Добавлено macbat\nexport PATH=\"$PATH:%s\"\n", path)); err != nil {
			f.Close()
			return fmt.Errorf("не удалось записать в файл конфигурации: %v", err)
		}
		f.Close()
		pathAdded = true
	}

	// Обновляем PATH в текущей сессии
	if pathAdded {
		// Добавляем путь в текущий PATH
		currentPath := os.Getenv("PATH")
		if !strings.Contains(currentPath, path) {
			os.Setenv("PATH", currentPath+":"+path)
		}

		// Выполняем source для обновления сессии
		var cmd *exec.Cmd
		switch shellName {
		case "zsh":
			cmd = exec.Command("zsh", "-c", "source "+configFile+" && exec zsh -i")
		case "bash":
			cmd = exec.Command("bash", "-c", "source "+configFile+" && exec bash -i")
		}

		// Запускаем в фоновом режиме, чтобы не блокировать
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			mess := fmt.Sprintf("Не удалось обновить текущую сессию: %v", err)
			log.Info(mess)
		}
	}

	return nil
}

// removeFromPath удаляет директорию из переменной PATH в файле конфигурации оболочки
// RemoveFromPath удаляет директорию из переменной PATH в файле конфигурации оболочки
func RemoveFromPath(path string, log *logger.Logger) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("не удалось определить домашнюю директорию: %v", err)
	}

	// Проверяем все возможные файлы конфигурации
	configFiles := []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".bash_profile"),
		filepath.Join(homeDir, ".bashrc"),
	}

	for _, configFile := range configFiles {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			continue
		}

		// Читаем содержимое файла
		data, err := os.ReadFile(configFile)
		if err != nil {
			continue
		}

		// Удаляем запись о пути
		content := string(data)
		pathEntry := fmt.Sprintf("\nexport PATH=\"$PATH:%s\"\n", path)
		content = strings.ReplaceAll(content, pathEntry, "\n")

		// Удаляем комментарий, если он есть
		content = strings.ReplaceAll(content, "\n# Добавлено macbat\n\n", "\n")

		// Записываем обновленное содержимое обратно в файл
		if err := os.WriteFile(configFile, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
			return fmt.Errorf("не удалось обновить файл конфигурации %s: %v", configFile, err)
		}
	}

	return nil
}

// updateShell обновляет текущую сессию оболочки
// UpdateShell обновляет текущую сессию оболочки
func UpdateShell(log *logger.Logger) error {
	// Получаем путь к текущей оболочке
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh" // Значение по умолчанию для macOS
	}

	// Определяем, какую команду source использовать в зависимости от оболочки
	var sourceCmd string
	switch filepath.Base(shell) {
	case "zsh":
		sourceCmd = "source ~/.zshrc"
	case "bash":
		sourceCmd = "source ~/.bash_profile || source ~/.bashrc"
	default:
		sourceCmd = "source ~/.zshrc || source ~/.bash_profile || source ~/.bashrc"
	}

	// Формируем команду для выполнения в новой оболочке
	cmd := exec.Command(shell, "-c", sourceCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Запускаем команду
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("не удалось обновить PATH: %v", err)
	}

	return nil
}
