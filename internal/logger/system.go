package logger

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

/**
 * @brief Отправить системное уведомление в macOS
 * @param title Заголовок уведомления
 * @param message Текст сообщения
 * @return Ошибку, если отправка не удалась
 */
func (l *Logger) ShowDialogNotification(title, message string) error {
	l.Debug(fmt.Sprintf("Попытка отправить уведомление.\nЗаголовок: '%s'\nСообщение: '%s'", title, message))

	if !l.isNotificationAvailable() {
		errMsg := "система уведомлений недоступна"
		l.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Проверяем, что заголовок и сообщение не пустые
	if title == "" {
		title = "MacBat"
		l.Debug("Использован заголовок по умолчанию: MacBat")
	}
	if message == "" {
		errMsg := "текст уведомления не может быть пустым"
		l.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	script := fmt.Sprintf(`display dialog "%s" with title "%s" with icon caution buttons {"OK"} default button "OK" giving up after 7`,
		strings.ReplaceAll(message, `"`, `\"`),
		strings.ReplaceAll(title, `"`, `\"`))

	// Устанавливаем таймаут на выполнение команды
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Выполняем команду osascript
	l.Debug("Выполнение команды osascript для отображения уведомления")
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("не удалось отправить уведомление: %v, stderr: %s", err, stderr.String())
		l.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	l.Debug("Уведомление успешно отправлено")
	l.Info(message)
	return nil
}

/**
 * @brief Проверить доступность системы уведомлений
 * @return true если система доступна
 */
func (l *Logger) isNotificationAvailable() bool {
	l.Debug("Проверка доступности системы уведомлений...")

	// Проверяем доступность утилиты osascript
	cmd := exec.Command("which", "osascript")
	if err := cmd.Run(); err != nil {
		l.Error("osascript не найден: " + err.Error())
		return false
	}

	// Проверяем, что мы можем отправить тестовое уведомление
	testCmd := exec.Command("osascript", "-e", `display notification "" with title "MacBat Test"`)
	if err := testCmd.Run(); err != nil {
		l.Error("Не удалось отправить тестовое уведомление: " + err.Error())
		return false
	}

	l.Debug("Система уведомлений доступна")
	return true
}
