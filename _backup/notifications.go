package notification

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"
)

//================================================================================
// ИНТЕРФЕЙСЫ И ИХ РЕАЛИЗАЦИИ
//================================================================================

type Notifier interface {
	Notify(message string)
	ShowHighBatteryNotification(message string) error
	ShowLowBatteryNotification(message string) error
}

/**
 * @struct FileNotifier
 * @brief Реализация Notifier, которая пишет в файл с поддержкой ротации.
 * @details Этот уведомитель отслеживает количество строк в текущем лог-файле.
 * Когда число строк достигает максимума, файл переименовывается,
 * а запись продолжается в новый пустой файл.
 */
type FileNotifier struct {
	filePath     string
	maxLines     int
	currentLines int
}

/**
 * @brief Создает новый экземпляр FileNotifier.
 * @details При создании он проверяет, существует ли уже лог-файл, и если да,
 * подсчитывает количество строк в нем, чтобы корректно продолжить работу.
 * @param filePath Основной путь к лог-файлу.
 * @param maxLines Максимальное количество строк до ротации.
 * @return Указатель на новый экземпляр FileNotifier.
 */
func NewFileNotifier(filePath string, maxLines int) *FileNotifier {
	// Подсчитываем начальное количество строк, если файл уже существует.
	lines := 0
	f, err := os.Open(filePath)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines++
		}
		f.Close()
	}

	return &FileNotifier{
		filePath:     filePath,
		maxLines:     maxLines,
		currentLines: lines,
	}
}

/**
 * @brief Записывает сообщение в лог и выполняет ротацию при необходимости.
 * @param message Текст сообщения.
 */
func (fn *FileNotifier) Notify(message string) {
	// Шаг 1: Проверяем, не пора ли выполнять ротацию.
	if fn.currentLines >= fn.maxLines {
		fmt.Printf("[Log Manager] Достигнут лимит в %d строк. Выполняется ротация...\n", fn.maxLines)
		err := fn.rotate()
		if err != nil {
			log.Printf("Критическая ошибка: не удалось выполнить ротацию лога: %v", err)
			// Даже если ротация не удалась, продолжаем писать в старый файл.
		}
	}

	// Шаг 2: Записываем сообщение в файл.
	f, err := os.OpenFile(fn.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Критическая ошибка: не удалось открыть лог-файл %s: %v", fn.filePath, err)
		return
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	logger.Println(message)
	fmt.Printf("УВЕДОМЛЕНИЕ: %s\n", message)

	// Шаг 3: Увеличиваем счетчик строк.
	fn.currentLines++
}

/**
 * @brief Выполняет ротацию лог-файла.
 * @details Переименовывает текущий лог-файл, добавляя к имени временную метку.
 * @private
 */
func (fn *FileNotifier) rotate() error {
	timestamp := time.Now().Format("2006-01-02T15_04_05")
	newName := fmt.Sprintf("%s_%s.log", fn.filePath, timestamp)

	err := os.Rename(fn.filePath, newName)
	if err != nil {
		return err
	}
	fmt.Printf("[Log Manager] Файл '%s' переименован в '%s'\n", fn.filePath, newName)

	// Сбрасываем счетчик, так как следующий вызов Notify создаст новый пустой файл.
	fn.currentLines = 0
	return nil
}

/**
 * @brief Отправляет уведомление о низком заряде батареи. Переопределенный метод интерфейса Notifier.
 * @param message Текст сообщения.
 * @return Ошибку, если отправка не удалась.
 */
func (fn *FileNotifier) ShowHighBatteryNotification(message string) error {
	return ShowDialogNotification("Внимание: Высокий заряд батареи", message)
}

/**
 * @brief Отправляет уведомление о низком заряде батареи. Переопределенный метод интерфейса Notifier.
 * @param message Текст сообщения.
 * @return Ошибку, если отправка не удалась.
 */
func (fn *FileNotifier) ShowLowBatteryNotification(message string) error {
	return ShowDialogNotification("Внимание: Низкий заряд батареи", message)
}
