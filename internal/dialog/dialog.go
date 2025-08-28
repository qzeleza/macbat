package dialog

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/qzeleza/macbat/internal/logger"
)

/**
 * @brief Отправить системное уведомление в macOS
 * @param title Заголовок уведомления
 * @param message Текст сообщения
 * @return Ошибку, если отправка не удалась
 */
func ShowDialogNotification(title, message string, log *logger.Logger) error {
	log.Debug(fmt.Sprintf("Попытка отправить уведомление.\nЗаголовок: '%s'\nСообщение: '%s'", title, message))

	if !IsNotificationAvailable(log) {
		errMsg := "система уведомлений недоступна"
		log.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Проверяем, что заголовок и сообщение не пустые
	if title == "" {
		title = "MacBat"
		log.Debug("Использован заголовок по умолчанию: MacBat")
	}
	if message == "" {
		errMsg := "текст уведомления не может быть пустым"
		log.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Сначала отправляем notification center уведомление
	if err := showNotificationCenter(title, message, log); err != nil {
		log.Error(fmt.Sprintf("Не удалось отправить notification center: %v", err))
	}

	// Затем показываем модальный диалог
	dialogScript := fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"OK"} default button "OK" with icon caution giving up after 10`,
		strings.ReplaceAll(message, `"`, `\"`),
		strings.ReplaceAll(title, `"`, `\"`))

	// Устанавливаем таймаут на выполнение команды
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Выполняем команду osascript для диалога
	log.Debug("Выполнение команды osascript для отображения диалога")
	cmd := exec.CommandContext(ctx, "osascript", "-e", dialogScript)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("не удалось отправить диалог: %v, stderr: %s", err, stderr.String())
		log.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	log.Debug("Диалог успешно отправлен")
	log.Info(message)
	return nil
}

/**
 * @brief Проверить доступность системы уведомлений
 * @return true если система доступна
 */
func IsNotificationAvailable(log *logger.Logger) bool {
	log.Debug("Проверка доступности системы уведомлений...")

	// Проверяем доступность утилиты osascript
	cmd := exec.Command("which", "osascript")
	if err := cmd.Run(); err != nil {
		log.Error("osascript не найден: " + err.Error())
		return false
	}

	// Проверяем базовую функциональность osascript без отправки уведомления
	// Используем простую команду для проверки доступности AppleScript
	testCmd := exec.Command("osascript", "-e", `tell application "System Events" to get name`)
	if err := testCmd.Run(); err != nil {
		log.Error("Система уведомлений недоступна: " + err.Error())
		return false
	}

	log.Debug("Система уведомлений доступна")
	return true
}

//================================================================================
// МЕТОДЫ СИСТЕМНЫХ УВЕДОМЛЕНИЙ
//================================================================================

// ShowHighBatteryNotification отправляет уведомление о высоком заряде батареи.
func ShowHighBatteryNotification(message string, log *logger.Logger) error {
	log.Info(fmt.Sprintf("Отправка уведомления о высоком заряде: %s", message))
	return ShowDialogNotification("Внимание: Высокий заряд батареи", message, log)
}

// ShowLowBatteryNotification отправляет уведомление о низком заряде батареи.
func ShowLowBatteryNotification(message string, log *logger.Logger) error {
	log.Info(fmt.Sprintf("Отправка уведомления о низком заряде: %s", message))
	return ShowDialogNotification("Внимание: Низкий заряд батареи", message, log)
}

/**
 * @brief Отправляет уведомление через Notification Center
 * @param title Заголовок уведомления
 * @param message Текст сообщения
 * @param log Логгер
 * @return Ошибка если отправка не удалась
 */
func showNotificationCenter(title, message string, log *logger.Logger) error {
	log.Debug("Отправка notification center уведомления")
	
	// AppleScript для отправки уведомления в Notification Center
	notificationScript := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`,
		strings.ReplaceAll(message, `"`, `\"`),
		strings.ReplaceAll(title, `"`, `\"`))

	// Устанавливаем таймаут на выполнение команды
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Выполняем команду osascript для notification center
	cmd := exec.CommandContext(ctx, "osascript", "-e", notificationScript)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("не удалось отправить notification center: %v, stderr: %s", err, stderr.String())
	}
	
	log.Debug("Notification center уведомление успешно отправлено")
	return nil
}
