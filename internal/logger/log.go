// Package log предоставляет комплексное решение для логирования с поддержкой уровней,
// ротации файлов и системных уведомлений.

package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

//================================================================================
// ОСНОВНАЯ СТРУКТУРА ЛОГГЕРА
//================================================================================

// Logger - это основной объект для управления логированием.
// Он инкапсулирует всю конфигурацию и состояние, включая ротацию файлов.
type Logger struct {
	mu             sync.Mutex // Для обеспечения потокобезопасности
	filePath       string
	maxLines       int
	currentLines   int
	isLogEnabled   bool
	isDebugEnabled bool
}

// New создает и инициализирует новый экземпляр Logger.
//
// @param filePath - Основной путь к лог-файлу.
// @param maxLines - Максимальное количество строк до ротации.
// @param logEnabled - Включает или отключает логирование в файл.
// @param debugEnabled - Включает или отключает логирование уровня DEBUG.
// @return *Logger - Указатель на новый экземпляр логгера.
func New(filePath string, maxLines int, logEnabled bool, debugEnabled bool) *Logger {

	// Удаляем старый лог-файл.
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		// Если файл не существует, это не ошибка. В иных случаях - выводим предупреждение.
		log.Printf("Предупреждение: не удалось удалить старый лог-файл: %v", err)
	}

	// Задаем начальное количество строк.
	lines := 0

	return &Logger{
		filePath:       filePath,
		maxLines:       maxLines,
		isLogEnabled:   logEnabled,
		isDebugEnabled: debugEnabled,
		currentLines:   lines,
	}
}

//================================================================================
// МЕТОДЫ ЛОГИРОВАНИЯ
//================================================================================

// EnableLogging включает или отключает логирование
// @param enabled bool - true для включения, false для отключения
func (l *Logger) EnableLogging(enabled bool) {
	l.isLogEnabled = enabled
}

// Info записывает информационное сообщение в лог.
func (l *Logger) Test(message string) {
	if l.isLogEnabled {
		l.logMessage("TEST", message)
	}
}

// Info записывает информационное сообщение в лог.
func (l *Logger) Info(message string) {
	if l.isLogEnabled {
		l.logMessage("INFO", message)
	}
}

// Info записывает информационное сообщение в лог.
func (l *Logger) Fatal(message string) {
	if l.isLogEnabled {
		log.Fatal(message)
	}
}

// Check записывает специальное сообщение о проверке состояния.
func (l *Logger) Check(message string) {
	if l.isLogEnabled {
		l.logMessage("CHECK", message)
	}
}

// Debug записывает отладочное сообщение в лог.
func (l *Logger) Debug(message string) {
	if l.isLogEnabled && l.isDebugEnabled {
		l.logMessage("DEBUG", message)
	}
}

// Error записывает сообщение об ошибке в лог.
func (l *Logger) Error(message string) {
	if l.isLogEnabled {
		// Ошибки можно писать в тот же файл с уровнем ERROR
		l.logMessage("ERROR", message)
	}
}

// logMessage - это внутренний метод для записи сообщений в файл.
// Он управляет ротацией и форматированием строк.
func (l *Logger) logMessage(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Шаг 1: Проверяем, не пора ли выполнять ротацию.
	if l.currentLines >= l.maxLines {
		fmt.Printf("[Log Manager] Достигнут лимит в %d строк. Выполняется ротация...\n", l.maxLines)
		if err := l.rotate(); err != nil {
			// Выводим критическую ошибку в стандартный вывод, так как запись в файл может быть невозможна.
			log.Printf("Критическая ошибка: не удалось выполнить ротацию лога: %v", err)
		}
	}

	// Шаг 2: Открываем файл для добавления записи.
	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Критическая ошибка: не удалось открыть лог-файл %s: %v", l.filePath, err)
		return
	}
	defer f.Close()

	// Шаг 3: Форматируем и записываем сообщение.
	timeFormat := "02-01-2006 15:04:05"
	logEntry := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format(timeFormat), level, strings.TrimSpace(message))

	if _, err := f.WriteString(logEntry); err != nil {
		log.Printf("Критическая ошибка: не удалось записать в лог: %v", err)
	}

	// Шаг 4: Увеличиваем счетчик строк.
	l.currentLines++
}

// rotate выполняет ротацию лог-файла.
func (l *Logger) rotate() error {
	timestamp := time.Now().Format("2006-01-02T15_04_05")
	// Формируем новое имя с расширением .log
	newName := fmt.Sprintf("%s_%s.log", strings.TrimSuffix(l.filePath, ".log"), timestamp)

	// Проверяем, существует ли файл, прежде чем переименовывать
	if _, err := os.Stat(l.filePath); os.IsNotExist(err) {
		// Файла нет, нечего ротировать. Просто сбрасываем счетчик.
		l.currentLines = 0
		return nil
	}

	err := os.Rename(l.filePath, newName)
	if err != nil {
		return err
	}
	fmt.Printf("[Log Manager] Файл '%s' переименован в '%s'\n", l.filePath, newName)

	// Сбрасываем счетчик, так как следующая запись создаст новый пустой файл.
	l.currentLines = 0
	return nil
}

//================================================================================
// МЕТОДЫ СИСТЕМНЫХ УВЕДОМЛЕНИЙ
//================================================================================

// ShowHighBatteryNotification отправляет уведомление о высоком заряде батареи.
func (l *Logger) ShowHighBatteryNotification(message string) error {
	l.Info(fmt.Sprintf("Отправка уведомления о высоком заряде: %s", message))
	return l.ShowDialogNotification("Внимание: Высокий заряд батареи", message)
}

// ShowLowBatteryNotification отправляет уведомление о низком заряде батареи.
func (l *Logger) ShowLowBatteryNotification(message string) error {
	l.Info(fmt.Sprintf("Отправка уведомления о низком заряде: %s", message))
	return l.ShowDialogNotification("Внимание: Низкий заряд батареи", message)
}
