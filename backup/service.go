package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"path/filepath"
)

// Install устанавливает приложение и регистрирует его как агент launchd.
//
// @param log *logger.Logger - логгер
// @return *config.Config - конфигурация приложения
// @return error - ошибка, если не удалось установить приложение
func Install(log *logger.Logger) (*config.Config, error) {

	log.Info("Начало установки приложения")
	// Копирование бинарника
	binPath := paths.BinaryPath()
	binDir := filepath.Dir(binPath)
	currentBin, err := os.Executable()
	if err != nil {
		mess := fmt.Sprintf("не удалось определить путь к бинарнику: %v", err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
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
		return nil, fmt.Errorf("не удалось создать директорию для логов: %v", err)
	}
	log.Debug("Директория для логов успешно создана")

	// Создаем директорию для бинарника, если её нет
	if err := os.MkdirAll(binDir, 0755); err != nil {
		mess := fmt.Sprintf("не удалось создать директорию %s: %v", binDir, err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
	}

	// Проверяем права на запись
	if err := config.CheckWriteAccess(binDir); err != nil {
		mess := fmt.Sprintf("нет прав на запись в %s: %v", binDir, err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
	}

	// Копируем бинарник
	log.Debug(fmt.Sprintf("Копирование бинарника из %s в %s", currentBin, binPath))
	data, err := os.ReadFile(currentBin)
	if err != nil {
		mess := fmt.Sprintf("не удалось прочитать бинарник: %v", err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
	}
	if err := os.WriteFile(binPath, data, 0755); err != nil {
		mess := fmt.Sprintf("не удалось записать бинарник в %s: %v", binPath, err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
	}
	log.Debug(fmt.Sprintf("Бинарник успешно записан: %s", binPath))

	// Добавляем директорию в PATH
	if err := addToPath(binDir); err != nil {
		// Не считаем это фатальной ошибкой, продолжаем установку
		mess := fmt.Sprintf("Предупреждение: не удалось добавить директорию в PATH: %v\n", err)
		log.Info(mess)
	} else {
		// Пытаемся обновить PATH в текущей оболочке
		if err := updateShell(); err != nil {
			mess_1 := fmt.Sprintf("Предупреждение: не удалось обновить PATH в текущей сессии: %v\n", err)
			log.Info(mess_1)
			mess_2 := "Выполните вручную: source ~/.zshrc (или source ~/.bash_profile)"
			log.Info(mess_2)
		}
	}

	// Загружаем конфигурацию из файла
	cfg, err := config.LoadConfig()
	if err != nil {
		mess := fmt.Sprintf("не удалось загрузить конфигурацию: %v", err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
	}

	// Создаем plist файл для агента
	if err := createPlistFile(binPath, log, cfg); err != nil {
		mess := fmt.Sprintf("не удалось создать plist: %v", err)
		log.Error(mess)
		return nil, fmt.Errorf("%s", mess)
	}

	// Загружаем агента при помощи launchd
	if state, err := Load(); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось загрузить агента: %v", err)
			log.Error(mess)
			return nil, fmt.Errorf("%s", mess)
		}
	}

	return cfg, nil
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
	if err := config.CheckWriteAccess(filepath.Dir(plistPath)); err != nil {
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
	if _, err := Unload(); err != nil {
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
	if err := removeFromPath(binDir); err != nil {
		// Не считаем это фатальной ошибкой, продолжаем удаление
		mess := fmt.Sprintf("Предупреждение: не удалось удалить директорию из PATH: %v\n", err)
		log.Info(mess)
	} else {
		// Пытаемся обновить PATH в текущей оболочке
		if err := updateShell(); err != nil {
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
func Load() (bool, error) {

	if !IsAgentRunning() {
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
func Unload() (bool, error) {
	if IsAgentRunning() {
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

// Reload выполняет перезагрузку агента в системе launchd.
//
// Функция последовательно вызывает Unload() и Load() для применения
// изменений конфигурации без перезагрузки системы.
//
// @return bool Флаг успешного выполнения операции
// @return error Ошибка, если не удалось перезагрузить агента
//
// Пример использования:
//
//	if ok, err := service.Reload(); !ok {
//	    log.Fatalf("Не удалось перезагрузить агента: %v", err)
//	}
//
// Примечания:
// - Требует прав администратора
// - Полезно после изменения конфигурации агента
// - Сохраняет текущее состояние агента
func Reload() (bool, error) {
	log.Info("Перезагрузка агента...")
	if _, err := Unload(); err != nil {
		log.Error(fmt.Sprintf("Ошибка при выгрузке агента перед перезагрузкой: %v", err))
		return false, err
	}
	log.Info("Агент выгружен, начинаем загрузку...")
	return Load()
}

// IsAgentRunning проверяет, запущен ли агент в системе.
//
// Функция использует команду launchctl для проверки состояния агента
// в контексте текущего пользователя.
//
// @return bool true, если агент запущен и активен, иначе false
//
// Пример использования:
//
//	if service.IsAgentRunning() {
//	    log.Println("Агент запущен")
//	} else {
//	    log.Println("Агент не запущен")
//	}
//
// Примечания:
// - Не требует прав администратора
// - Возвращает false в случае ошибки выполнения команды
// - Проверяет только активные процессы, не статус загрузки
func IsAgentRunning() bool {
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/"+paths.AgentIdentifier(), os.Getuid()))
	output, err := cmd.Output()
	if err != nil {
		log.Error(fmt.Sprintf("Ошибка при проверке статуса агента: %v", err))
		return false
	}
	isRunning := strings.Contains(string(output), paths.AgentIdentifier())
	log.Debug(fmt.Sprintf("Агент %s", map[bool]string{true: "запущен", false: "не запущен"}[isRunning]))
	return isRunning
}

// ReadLogs читает и возвращает содержимое лог-файлов приложения с возможностью фильтрации по уровню.
//
// Функция последовательно читает оба файла логов (стандартный вывод и ошибки)
// и возвращает их содержимое в виде среза строк, отфильтрованное по указанному уровню.
//
// @param level Уровень логирования для фильтрации ("debug", "info", "error").
//
//	Если пустая строка, возвращаются все логи.
//
// @return []string Содержимое лог-файлов, объединенное в один срез и отфильтрованное
// @return error Ошибка, если не удалось прочитать файлы логов
//
// Пример использования:
//
//	// Получить все логи
//	logs, err := service.ReadLogs("")
//
//	// Получить только логи уровня error
//	errorLogs, err := service.ReadLogs("error")
//
// Примечания:
// - Возвращает пустой срез, если файлы логов не существуют
// - Игнорирует ошибки отсутствия файлов логов
// - Регистр уровня не имеет значения ("Error", "ERROR", "error" обрабатываются одинаково)
func ReadLogs(level string) ([]string, error) {
	logPaths := []string{paths.LogPath(), paths.ErrorLogPath()}
	var logs []string
	level = strings.ToUpper(level) // Приводим к верхнему регистру для сравнения

	// Отладочная информация
	if level != "" {
		log.Info(fmt.Sprintf("Чтение логов с уровнем: '%s'", level))
	}
	log.Info(fmt.Sprintf("Пути к логам: %v", logPaths))

	for _, path := range logPaths {
		log.Info(fmt.Sprintf("Проверка файла: %s", path))

		// Проверяем существование файла
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Info(fmt.Sprintf("Файл лога не существует: %s", path))
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			mess := fmt.Sprintf("не удалось прочитать лог %s: %v", path, err)
			log.Error(mess)
			return nil, fmt.Errorf("%s", mess)
		}

		log.Info(fmt.Sprintf("Прочитано %d байт из %s", len(data), path))

		if len(data) > 0 {
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			lineNum := 0
			matchedLines := 0

			for scanner.Scan() {
				line := scanner.Text()
				lineNum++

				// Пропускаем пустые строки
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Если уровень не указан или строка содержит указанный уровень
				if level == "" {
					logs = append(logs, line)
					matchedLines++
				} else {
					// Проверяем формат: [дата-время] LEVEL: сообщение
					// Ищем закрывающую квадратную скобку и двоеточие после уровня
					timeEnd := strings.Index(line, "]")
					if timeEnd > 0 && len(line) > timeEnd+2 {
						// Пропускаем пробелы после закрывающей скобки
						levelStart := timeEnd + 1
						for levelStart < len(line) && line[levelStart] == ' ' {
							levelStart++
						}

						// Ищем конец уровня (двоеточие)
						levelEnd := strings.Index(line[levelStart:], ":")
						if levelEnd > 0 {
							logLevel := strings.TrimSpace(line[levelStart : levelStart+levelEnd])
							if strings.EqualFold(logLevel, level) {
								logs = append(logs, line)
								matchedLines++
							}
						}
					}
				}
			}

			log.Info(fmt.Sprintf("Проверено %d строк, найдено %d совпадений", lineNum, matchedLines))
		}
	}

	log.Info(fmt.Sprintf("Всего найдено записей: %d", len(logs)))
	return logs, nil
}
