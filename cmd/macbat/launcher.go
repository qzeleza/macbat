// cmd/macbat/launcher.go
package main

import (
	"fmt"

	"github.com/qzeleza/macbat/internal/monitor"
)

// RunLauncher запускает приложение в режиме лаунчера (оптимизированная версия)
func RunLauncher(deps *Dependencies) error {
	log := deps.Logger
	bgManager := deps.BgManager

	// Быстрая проверка установки без тяжелых операций
	if !monitor.IsAppInstalled(log) {
		log.Line()
		log.Info("Приложение не установлено. Выполняем установку...")

		if err := performInstallation(deps); err != nil {
			return fmt.Errorf("ошибка во время установки: %w", err)
		}

		log.Info("Установка успешно завершена.")
	}

	log.Line()
	log.Info("Запускаем приложение (режим лаунчера)...")

	// Оптимизированная проверка запущенного GUI агента
	if bgManager.IsRunning(GUIAgentProcessName) {
		log.Info("Приложение уже запущено. Выход.")
		return nil
	}

	// Асинхронный запуск GUI агента для быстрого завершения лаунчера
	log.Info("Запускаем GUI агента...")
	bgManager.LaunchDetached(GUIAgentProcessName)
	log.Info("Приложение успешно запущено в фоновом режиме. Лаунчер завершает работу.")
	log.Line()

	return nil
}

// performInstallation выполняет установку с улучшенной обработкой ошибок
func performInstallation(deps *Dependencies) error {
	// Здесь должен быть вызов функции Install с лучшей обработкой ошибок
	// return Install(deps.Logger, deps.Config)
	return fmt.Errorf("функция установки не реализована") // placeholder
}
