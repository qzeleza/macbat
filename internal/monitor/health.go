package monitor

import (
	"bufio"
	"fmt"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"os/exec"
	"strings"
)

// IsAppInstalled проверяет, корректно ли установлено приложение, включая все его компоненты.
func IsAppInstalled(log *logger.Logger) bool {

	// Удаляем файлы предыдущей версии
	if err := removeOldFiles(log); err != nil {
		log.Error(fmt.Sprintf("Не удалось удалить файлы предыдущей версии: %v", err))
		return false
	}

	log.Info("Запуск проверки наличия установленных файлов...")

	// Определяем задачи для проверки в виде карты.
	// Каждому файлу соответствует свой список искомых строк.
	// Для бинарного файла список строк пуст - проверяем только его наличие.
	searchTasks := map[string][]string{
		paths.BinaryPath(): {},
		paths.PlistPath(): {
			"ProgramArguments",
			paths.AgentIdentifier(),
			"RunAtLoad",
			"KeepAlive",
		},
		paths.ConfigPath(): {
			"min_threshold", // Ищем ошибки в системном логе
			"max_threshold",
			"notification_interval",
			"max_notifications",
			"log_file_path",
			"log_rotation_lines",
			"check_interval_charging",
			"check_interval_discharging",
			"log_enabled",
			"debug_enabled",
		},
	}

	// Вызываем нашу обновленную функцию
	ok, err := checkFilesAndContent(searchTasks, log)
	if err != nil {
		log.Fatal(fmt.Sprintf("Критическая ошибка при проверке файлов: %v", err))
	}

	// Обрабатываем простой булев результат.
	if ok {
		log.Debug("Проверка пройдена успешно: все файлы существуют и содержат все необходимые строки.")
	} else {
		// Используем Warn, так как это не критическая ошибка, а результат проверки.
		log.Debug("Проверка НЕ пройдена. Детали смотрите в логах выше. Продолжение работы...")
		// В реальном приложении здесь можно было бы выйти: os.Exit(1)
	}

	if ok {
		// Проверяем запущен ли агент
		if IsAgentRunning(log) {
			log.Debug("Агент запущен...")
		} else {
			log.Debug("Агент не запущен. Запуск...")
			if err := loadAgent(log); err != nil {
				log.Fatal(fmt.Sprintf("Ошибка во время загрузки агента: %v", err))
				return false
			}
			return true
		}
	}
	// Возвращаем результат проверки
	return ok
}

/**
 * @brief Выполняет строгую проверку: все ли файлы существуют и содержат ли все необходимые строки.
 *
 * @param filesToSearch Карта, где ключ - это путь к файлу, а значение - срез строк,
 * которые необходимо найти в этом конкретном файле.
 *
 * @return bool `true` только если ВСЕ файлы существуют и КАЖДЫЙ из них содержит ВСЕ указанные строки.
 * В противном случае `false`.
 * @return err Ошибка, если возникли проблемы с доступом к файлам (кроме их отсутствия).
 */
func checkFilesAndContent(filesToSearch map[string][]string, log *logger.Logger) (bool, error) {
	// Итерируем по карте "файл -> список строк для поиска".
	for filePath, requiredStrings := range filesToSearch {
		// Шаг 1: Проверяем, существует ли файл.
		// os.Stat возвращает информацию о файле или ошибку.
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			// Если файла нет - это провал всей проверки.
			log.Debug(fmt.Sprintf("Проверка не пройдена: файл '%s' не найден.", filePath))
			return false, nil // Ошибки нет, результат проверки - ложь.
		} else if err != nil {
			// Другая ошибка (например, нет прав) - это системная ошибка.
			return false, fmt.Errorf("ошибка при доступе к файлу %s: %w", filePath, err)
		}

		log.Debug(fmt.Sprintf("Файл '%s' найден, проверяю наличие всех строк: %v", filePath, requiredStrings))

		// Шаг 2: Проверяем, что в файле есть ВСЕ необходимые строки.
		allStringsFound, err := allStringsExistInFile(filePath, requiredStrings, log)
		if err != nil {
			// Если при чтении файла возникла ошибка, возвращаем ее.
			return false, err
		}

		if !allStringsFound {
			// Если хотя бы одна строка не найдена - это провал.
			log.Debug(fmt.Sprintf("Проверка не пройдена: в файле '%s' найдены не все требуемые строки.", filePath))
			return false, nil
		}

		log.Debug(fmt.Sprintf("В файле '%s' найдены все требуемые строки.", filePath))
	}

	// Если цикл завершился, значит все проверки пройдены успешно.
	return true, nil
}

/**
 * @brief Вспомогательная функция, которая проверяет, что ВСЕ строки из среза есть в файле.
 */
func allStringsExistInFile(filePath string, requiredStrings []string, log *logger.Logger) (bool, error) {
	// Если искать ничего не нужно, то результат - успех.
	if len(requiredStrings) == 0 {
		return true, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	// Создаем карту для отслеживания найденных строк.
	foundTracker := make(map[string]bool)
	for _, s := range requiredStrings {
		foundTracker[s] = false
	}
	foundCount := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		for _, reqStr := range requiredStrings {
			// Ищем только те строки, которые еще не были найдены
			if !foundTracker[reqStr] && strings.Contains(line, reqStr) {
				foundTracker[reqStr] = true
				foundCount++
			}
		}
		// Оптимизация: если уже нашли все, что искали, выходим из цикла досрочно.
		if foundCount == len(requiredStrings) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("ошибка при чтении файла %s: %w", filePath, err)
	}

	// Возвращаем true, только если количество найденных уникальных строк равно количеству искомых.
	return foundCount == len(requiredStrings), nil
}

// isAgentRunning проверяет, запущен ли агент в системе.
//
// Функция использует команду launchctl для проверки состояния агента
// в контексте текущего пользователя.
//
// @param log *logger.Logger - логгер
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
func IsAgentRunning(log *logger.Logger) bool {
	log.Debug("Проверка статуса агента...")
	agentID := paths.AgentIdentifier()
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/"+agentID, os.Getuid()))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug(fmt.Sprintf("Агент не запущен или произошла ошибка: %v", err))
		return false
	}

	log.Debug(fmt.Sprintf("Вывод launchctl: %s", string(output)))

	// Проверяем, что PID существует и процесс не является \"Could not find service\"
	return strings.Contains(string(output), agentID) && !strings.Contains(string(output), "Could not find service")
}

// removeOldFiles удаляет старые файлы конфигурации и логов.
func removeOldFiles(log *logger.Logger) error {
	filesToRemove := []string{
		paths.PlistPath(),
		paths.LogPath(),
		paths.ErrorLogPath(),
	}

	for _, path := range filesToRemove {
		log.Debug(fmt.Sprintf("Удаление файла: %s", path))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Error(fmt.Sprintf("Не удалось удалить %s: %v", path, err))
			// Продолжаем удаление других файлов
		} else if err == nil {
			log.Debug(fmt.Sprintf("Файл успешно удален: %s", path))
		}
	}
	return nil
}

// loadAgent является вспомогательной функцией для загрузки агента.
func loadAgent(log *logger.Logger) error {
	if state, err := LoadAgent(log); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось загрузить агента: %v", err)
			log.Error(mess)
			return fmt.Errorf("%s", mess)
		}
	}
	return nil
}

// LoadAgent загружает и активирует агент в системе launchd.
//
// Функция регистрирует агента в launchd, что позволяет ему автоматически
// запускаться при загрузке системы и перезапускаться при сбоях.
//
// @return bool Флаг успешного выполнения операции
// @return error Ошибка, если не удалось загрузить агента
func LoadAgent(log *logger.Logger) (bool, error) {

	if !IsAgentRunning(log) {
		cmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), paths.PlistPath())
		if err := cmd.Run(); err != nil && strings.Contains(err.Error(), "Could not find service") {
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

// UnloadAgent останавливает и выгружает агент из системы launchd.
//
// Функция останавливает выполнение агента и удаляет его из списка
// автоматически запускаемых при загрузке системы.
//
// @return bool Флаг успешного выполнения операции
// @return error Ошибка, если не удалось выгрузить агента
func UnloadAgent(log *logger.Logger) (bool, error) {
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
