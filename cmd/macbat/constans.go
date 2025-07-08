// cmd/macbat/cli/constants.go
package main

import "time"

// Константы приложения
const (
	// Основные параметры приложения
	AppName        = "macbat"
	AppUsage       = "Утилита мониторинга батареи для macOS"
	AppDescription = "Приложение для мониторинга состояния батареи Mac с возможностью установки пороговых значений и отправки уведомлений"

	// Параметры производительности
	AppStartupTimeout = 30 * time.Second // Максимальное время запуска приложения
	MaxLogSizeMB      = 100              // Максимальный размер лог-файла в МБ
	DebugMode         = true             // Режим отладки

	// Имена процессов для менеджера фоновых процессов
	BackgroundProcessName = "--background"
	GUIAgentProcessName   = "--gui-agent"

	// Редактор по умолчанию для конфигурации
	DefaultEditor = "nano"

	// Форматирование вывода
	SeparatorLength = 100
	SeparatorChar   = "-"
)
