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

	script := fmt.Sprintf(`display dialog "%s" with title "%s" with icon caution buttons {"OK"} default button "OK" giving up after 7`,
		strings.ReplaceAll(message, `"`, `\"`),
		strings.ReplaceAll(title, `"`, `\"`))

	// Устанавливаем таймаут на выполнение команды
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Выполняем команду osascript
	log.Debug("Выполнение команды osascript для отображения уведомления")
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("не удалось отправить уведомление: %v, stderr: %s", err, stderr.String())
		log.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	log.Debug("Уведомление успешно отправлено")
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

	// Проверяем, что мы можем отправить тестовое уведомление
	testCmd := exec.Command("osascript", "-e", `display notification "" with title "MacBat Test"`)
	if err := testCmd.Run(); err != nil {
		log.Error("Не удалось отправить тестовое уведомление: " + err.Error())
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
