package main

import (
	"bufio"
	"fmt"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"os/exec"
	"strings"
)

func isAppInstalled(log *logger.Logger) bool {

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

// IsAgentRunning проверяет, запущен ли агент в системе.
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
	output, err := cmd.Output()
	if err != nil && !strings.Contains(err.Error(), "Could not find service") {
		log.Error(fmt.Sprintf("Ошибка при проверке статуса агента: %v", err.Error()))
		return false
	}
	isRunning := strings.Contains(string(output), agentID)
	log.Debug(fmt.Sprintf("Агент %s", map[bool]string{true: "запущен", false: "не запущен"}[isRunning]))
	return isRunning
}
