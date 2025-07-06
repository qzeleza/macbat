package monitor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/qzeleza/macbat/internal/logger"
	"github.com/qzeleza/macbat/internal/paths"
)

// IsAppInstalled проверяет, корректно ли установлено приложение, включая все его компоненты.
func IsAppInstalled(log *logger.Logger) bool {

	// Удаляем файлы предыдущей версии
	if err := removeOldFiles(log); err != nil {
		log.Error(fmt.Sprintf("Не удалось удалить файлы предыдущей версии: %v", err))
		return false
	}

	log.Line()
	log.Info("Запуск проверки наличия установленных файлов...")
	log.Line()

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
	agentID := paths.AgentIdentifier()
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/"+agentID, os.Getuid()))
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "Could not find service") {
		log.Debug(fmt.Sprintf("Агент не запущен, ошибка: %v", err))
		return false
	}
	// Если в выводе содержится "Could not find service", значит агент не запущен, но ошибки нет
	if strings.Contains(string(output), "Could not find service") {
		log.Debug("Агент не запущен.")
		return false
	}
	log.Debug("Агент запущен.")
	// Проверяем, что PID существует и процесс не является \"Could not find service\"
	return strings.Contains(string(output), agentID)
}

// removeOldFiles удаляет старые файлы конфигурации и логов.
//
// Функция использует paths.LogPath() и paths.ErrorLogPath() для получения
// путей к файлам, которые необходимо удалить.
//
// @param log *logger.Logger - логгер
// @return error - ошибка, если удаление файла не удалось
func removeOldFiles(log *logger.Logger) error {
	filesToRemove := []string{
		paths.LogPath(),
		paths.ErrorLogPath(),
	}

	for _, path := range filesToRemove {
		log.Debug(fmt.Sprintf("Удаление файла: %s", path))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Error(fmt.Sprintf("Не удалось удалить %s: %v", path, err))
		}
	}
	log.Line()
	log.Debug(fmt.Sprintf("Файлы %v успешно удалены.", filesToRemove))
	return nil
}

// loadAgent является вспомогательной функцией для загрузки агента.
// LoadAndEnableAgent загружает и подключает агента в системе launchd.
//
// Функция выполняет две основные операции: загрузку агента с помощью
// команды "bootstrap" и его подключение с помощью команды "enable".
// Ожидается, что агент будет успешно загружен и подключен, чтобы
// обеспечивать свою работоспособность в системе.
//
// @param log *logger.Logger - логгер для записи сообщений о ходе выполнения
//
// @return error - ошибка, если не удалось загрузить или подключить агента
func LoadAndEnableAgent(log *logger.Logger) error {
	if state, err := CommandAgentService(log, "bootstrap"); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось загрузить агента: %v", err)
			log.Error(mess)
			return fmt.Errorf("%s", mess)
		}
	}
	if state, err := CommandAgentService(log, "enable"); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось подключить агента: %v", err)
			log.Error(mess)
			return fmt.Errorf("%s", mess)
		}
	}
	log.Debug("Агент успешно загружен и подключен.")
	return nil
}

// UnloadAndDisableAgent отключает и выгружает агент из launchd.
//
// Функция отключает и выгружает агент, если он запущен. Если агент не
// запущен, функция просто возвращает true, не выполняя никаких действий.
//
// @param log *logger.Logger - логгер для записи сообщений о ходе выполнения
//
// @return error - ошибка, если не удалось отключить или выгрузить агента
func UnloadAndDisableAgent(log *logger.Logger) error {
	if state, err := CommandAgentService(log, "disable"); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось отключить агента: %v", err)
			log.Error(mess)
			return fmt.Errorf("%s", mess)
		}
	}
	if state, err := CommandAgentService(log, "bootout"); err != nil {
		if !state {
			mess := fmt.Sprintf("не удалось выгрузить агента: %v", err)
			log.Error(mess)
			return fmt.Errorf("%s", mess)
		}
	}
	log.Debug("Агент успешно выгружен и отключен.")
	return nil
}

// ControlAgentService управляет состоянием агента в системе.
//
// Функция принимает 2 параметра: логгер для записи сообщений о ходе выполнения
// и строку, указывающую на действие, которое нужно выполнить (например, "bootstrap"
// или "bootout"). Если агент не запущен, функция выполняет команду launchctl
// с указанным действием для изменения состояния агента в системе.
// Если агент уже запущен, функция возвращает ошибку.
//
// @param log *logger.Logger - логгер для записи сообщений о ходе выполнения
// @param action string - действие, которое нужно выполнить (например, "bootstrap"
//
//	или "bootout")
//
// @return bool true, если команда выполнена успешно, иначе false
// @return error ошибка выполнения команды, если она произошла
func ControlAgentService(log *logger.Logger, action string) (bool, error) {
	if !IsAgentRunning(log) {
		cmd := exec.Command("launchctl", action, fmt.Sprintf("gui/%d/%s", os.Getuid(), paths.AgentIdentifier()))
		if err := cmd.Run(); err != nil && strings.Contains(err.Error(), "Could not find service") {
			act := "загрузить"
			if action == "disable" {
				act = "выгрузить"
			}
			mess := fmt.Sprintf("не удалось %s агента: %v", act, err)
			log.Error(mess)
			return false, fmt.Errorf("%s", mess)
		}
		return true, nil
	}
	mess := "Агент уже загружен посредством launchctl"
	log.Debug(mess)
	return true, fmt.Errorf("%s", mess)
}

// CommandAgentService управляет состоянием агента в системе через launchctl.
//
// Функция выполняет команду launchctl с указанным действием (например, "bootstrap" или "bootout")
// для изменения состояния агента в системе. Это позволяет загружать или выгружать агента
// в зависимости от переданного параметра action.
//
// @param log *logger.Logger - логгер для записи сообщений о ходе выполнения
// @param action string - действие, которое нужно выполнить (например, "bootstrap" или "bootout")
//
// @return bool true, если команда выполнена успешно, иначе false
// @return error ошибка выполнения команды, если она произошла
//
// Примечания:
// - Требует прав администратора
// - Возвращает false и сообщение об ошибке, если служба не найдена
func CommandAgentService(log *logger.Logger, action string) (bool, error) {
	cmd := exec.Command("launchctl", action, fmt.Sprintf("gui/%d/%s", os.Getuid(), paths.AgentIdentifier()))
	if err := cmd.Run(); err != nil && strings.Contains(err.Error(), "Could not find service") {
		act := "bootstrap"
		if action == "bootout" {
			act = "bootout"
		}
		mess := fmt.Sprintf("не удалось %s агента: %v", act, err)
		log.Error(mess)
		return false, fmt.Errorf("%s", mess)
	}
	return true, nil
}
