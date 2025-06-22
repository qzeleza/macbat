package main

import (
	"fmt"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Install устанавливает приложение и регистрирует его как агент launchd.
//
// @param log *logger.Logger - логгер
// @return *appConfig.Config - конфигурация приложения
// @return error - ошибка, если не удалось установить приложение
func Install(log *logger.Logger, cfg *config.Config) error {

	log.Info("Начало установки приложения")

	// Копирование бинарника
	binPath := paths.BinaryPath()
	binDir := filepath.Dir(binPath)
	currentBin, err := os.Executable()
	if err != nil {
		mess := fmt.Sprintf("не удалось определить путь к бинарнику: %v", err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}

	// Список файлов приложения
	filesToRemove := []string{
		paths.LogPath(),
		paths.ErrorLogPath(),
		paths.PlistPath(),
		binPath,
	}
	// Удаляем файлы приложения предыдущей версии
	for _, path := range filesToRemove {
		log.Debug(fmt.Sprintf("Удаление файла: %s", path))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Error(fmt.Sprintf("Не удалось удалить %s: %v", path, err))
			// Продолжаем удаление других файлов
		} else if err == nil {
			log.Debug(fmt.Sprintf("Файл успешно удален: %s", path))
		}
	}

	// Создаем директорию для логов, если её нет
	logDir := filepath.Dir(paths.LogPath())
	log.Debug(fmt.Sprintf("Создание директории для логов: %s", logDir))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Error(fmt.Sprintf("Ошибка создания директории для логов: %v", err))
		return fmt.Errorf("не удалось создать директорию для логов: %v", err)
	}
	log.Debug("Директория для логов успешно создана")

	// Создаем директорию для бинарника, если её нет
	if err := os.MkdirAll(binDir, 0755); err != nil {
		mess := fmt.Sprintf("не удалось создать директорию %s: %v", binDir, err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}

	// Проверяем права на запись
	if err := CheckWriteAccess(binDir, log); err != nil {
		mess := fmt.Sprintf("нет прав на запись в %s: %v", binDir, err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}

	// Копируем бинарник
	log.Debug(fmt.Sprintf("Копирование бинарника из %s в %s", currentBin, binPath))
	data, err := os.ReadFile(currentBin)
	if err != nil {
		mess := fmt.Sprintf("не удалось прочитать бинарник: %v", err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	if err := os.WriteFile(binPath, data, 0755); err != nil {
		mess := fmt.Sprintf("не удалось записать бинарник в %s: %v", binPath, err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	log.Debug(fmt.Sprintf("Бинарник успешно записан: %s", binPath))

	// Добавляем директорию в PATH
	if err := addToPath(binDir, log); err != nil {
		// Не считаем это фатальной ошибкой, продолжаем установку
		mess := fmt.Sprintf("Предупреждение: не удалось добавить директорию в PATH: %v\n", err)
		log.Info(mess)
	} else {
		// Пытаемся обновить PATH в текущей оболочке
		if err := updateShell(log); err != nil {
			mess_1 := fmt.Sprintf("Предупреждение: не удалось обновить PATH в текущей сессии: %v\n", err)
			log.Info(mess_1)
			mess_2 := "Выполните вручную: source ~/.zshrc (или source ~/.bash_profile)"
			log.Info(mess_2)
		}
	}

	// Создаем plist файл для агента
	if err := createPlistFile(binPath, log, cfg); err != nil {
		mess := fmt.Sprintf("не удалось создать plist: %v", err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}

	// Загружаем агента при помощи launchd
	if state, err := Load(log); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось загрузить агента: %v", err)
			log.Error(mess)
			return fmt.Errorf("%s", mess)
		}
	}

	return nil
}

// createPlistFile создает файл конфигурации для launchd в формате plist.
//
// Функция генерирует XML-файл, который содержит настройки для запуска агента,
// включая путь к исполняемому файлу, параметры запуска и пути к логам.
//
// @param binPath string Абсолютный путь к исполняемому файлу агента
// @return error Ошибка, если не удалось создать или записать файл конфигурации
//
// Пример использования:
//
//	if err := createPlistFile("/usr/local/bin/macbat"); err != nil {
//	    log.Fatalf("Ошибка создания plist: %v", err)
//	}
//
// Примечания:
// - Автоматически создает необходимые директории
// - Устанавливает права доступа 0644 на созданный файл
// - Использует настройки из загруженной конфигурации
func createPlistFile(binPath string, log *logger.Logger, cfg *config.Config) error {

	// Создаем plist-файл для агента
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
	<key>EnvironmentVariables</key>
	<dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>`, paths.AgentIdentifier(), binPath, paths.LogPath(), paths.ErrorLogPath())

	plistPath := paths.PlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		mess := fmt.Sprintf("не удалось создать директорию для plist: %v", err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	if err := CheckWriteAccess(filepath.Dir(plistPath), log); err != nil {
		mess := fmt.Sprintf("нет прав на запись в %s: %v", filepath.Dir(plistPath), err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		mess := fmt.Sprintf("не удалось записать plist: %v", err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	return nil
}

// Uninstall выполняет полное удаление приложения из системы.
//
// Процесс удаления включает:
// 1. Остановку и выгрузку агента из launchd
// 2. Удаление plist-файла конфигурации
// 3. Удаление исполняемого файла
// 4. Удаление логов и временных файлов
// 5. Обновление переменной окружения PATH
//
// @return error Ошибка, если процесс удаления не был завершен успешно
//
// Пример использования:
//
//	if err := service.Uninstall(); err != nil {
//	    log.Fatalf("Ошибка удаления: %v", err)
//	}
//
// Примечания:
// - Требует прав администратора
// - Не удаляет пользовательские конфигурации
// - Автоматически обновляет PATH в текущей сессии
func Uninstall(log *logger.Logger) error {
	log.Info("Начало удаления приложения")
	// Получаем путь к директории с бинарником перед удалением
	binDir := filepath.Dir(paths.BinaryPath())

	// Выгружаем агент, если он запущен
	log.Info("Выгрузка агента...")
	if _, err := Unload(log); err != nil {
		log.Error(fmt.Sprintf("Ошибка выгрузки агента: %v", err))
		return fmt.Errorf("не удалось выгрузить агент: %v", err)
	}
	log.Info("Агент успешно выгружен")

	// Удаляем plist файл
	plistPath := paths.PlistPath()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		mess := fmt.Sprintf("не удалось удалить файл %s: %v", plistPath, err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}

	// Удаляем бинарник
	binPath := paths.BinaryPath()
	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		mess := fmt.Sprintf("не удалось удалить бинарник %s: %v", binPath, err)
		log.Error(mess)
		return fmt.Errorf("%s", mess)
	}

	// Удаляем директорию из PATH
	if err := removeFromPath(binDir, log); err != nil {
		// Не считаем это фатальной ошибкой, продолжаем удаление
		mess := fmt.Sprintf("Предупреждение: не удалось удалить директорию из PATH: %v\n", err)
		log.Info(mess)
	} else {
		// Пытаемся обновить PATH в текущей оболочке
		if err := removeFromPath(binDir, log); err != nil {
			mess_1 := fmt.Sprintf("Предупреждение: не удалось обновить PATH в текущей сессии: %v\n", err)
			log.Info(mess_1)
			mess_2 := "Выполните вручную: source ~/.zshrc (или source ~/.bash_profile)"
			log.Info(mess_2)
		}
	}

	// Удаляем файлы
	paths := []string{
		paths.BinaryPath(),
		paths.ConfigPath(),
		paths.LogPath(),
		paths.ErrorLogPath(),
		paths.PlistPath(),
	}

	for _, path := range paths {
		log.Info(fmt.Sprintf("Удаление файла: %s", path))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Error(fmt.Sprintf("Не удалось удалить %s: %v", path, err))
			// Продолжаем удаление других файлов
		} else if err == nil {
			log.Info(fmt.Sprintf("Файл успешно удален: %s", path))
		}
	}
	log.Info("Удаление приложения завершено")
	return nil
}

// Load загружает и активирует агент в системе launchd.
//
// Функция регистрирует агента в launchd, что позволяет ему автоматически
// запускаться при загрузке системы и перезапускаться при сбоях.
//
// @return bool Флаг успешного выполнения операции
// @return error Ошибка, если не удалось загрузить агента
//
// Пример использования:
//
//	if ok, err := service.Load(); !ok {
//	    log.Fatalf("Не удалось загрузить агента: %v", err)
//	}
//
// Примечания:
// - Требует прав администратора
// - Автоматически проверяет, не загружен ли агент
// - Использует идентификатор пользователя для загрузки в правильный домен
func Load(log *logger.Logger) (bool, error) {

	if !IsAgentRunning(log) {
		cmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), paths.PlistPath())
		if err := cmd.Run(); err != nil {
			mess := fmt.Sprintf("не удалось загрузить агента: %v", err)
			log.Error(mess)
			return false, fmt.Errorf("%s", mess)
		}
		return true, nil
	}
	mess := "Агент уже загружен посредством launchctl"
	log.Debug(mess)
	return true, fmt.Errorf("%s", mess)
}

// Unload останавливает и выгружает агент из системы launchd.
//
// Функция останавливает выполнение агента и удаляет его из списка
// автоматически запускаемых при загрузке системы.
//
// @return bool Флаг успешного выполнения операции
// @return error Ошибка, если не удалось выгрузить агента
//
// Пример использования:
//
//	if ok, err := service.Unload(); !ok {
//	    log.Fatalf("Не удалось выгрузить агента: %v", err)
//	}
//
// Примечания:
//   - Требует прав администратора
//   - Игнорирует ошибку "Input/output error", которая может возникать
//     при попытке выгрузки уже остановленного агента
func Unload(log *logger.Logger) (bool, error) {
	if IsAgentRunning(log) {
		cmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d", os.Getuid()), paths.PlistPath())
		if err := cmd.Run(); err != nil && !strings.Contains(err.Error(), "Boot-out failed: 5: Input/output error") {
			mess := fmt.Sprintf("не удалось выгрузить агент: %v", err)
			log.Error(mess)
			return false, fmt.Errorf("%s", mess)
		}
		return true, nil
	}
	mess := "Агент уже выгружен посредством launchctl"
	log.Debug(mess)
	return true, fmt.Errorf("%s", mess)
}
