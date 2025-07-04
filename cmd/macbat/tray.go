// Пакет main содержит реализацию иконки в системном трее
package main

import (
	_ "embed"
	"fmt"
	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/caseymrm/menuet"
)

// Глобальные переменные для хранения состояния приложения
var (
	appLog      *logger.Logger
	batteryInfo *battery.BatteryInfo
	appConfig   *config.Config
	updateMu    sync.Mutex // защита от параллельного вызова
)

// initTray инициализирует меню в трее macOS
func initTray() {
	// Создаем логгер
	appLog = logger.New(paths.LogPath(), 100, true, false)

	// Инициализируем конфигурацию
	cfgManager, _ := config.New(appLog, paths.ConfigPath())
	appConfig, _ = cfgManager.Load()

	// Инициализируем меню
	ap := menuet.App()
	ap.Name = "MacBat"
	ap.Label = "MacBat"

	// Настраиваем состояние меню и обработчики
	menuState := &menuet.MenuState{
		Title: "MBat",
		// Указываем путь к иконке, если бы она была в ресурсах
		// Встроенный бинарный ресурс не поддерживается напрямую, нужно сохранить
	}
	ap.SetMenuState(menuState)

	// Установка функции для генерации меню
	ap.Children = menuItems

	// Запускаем автоматическое обновление статуса
	go startStatusUpdater()

	// Запускаем GUI-цикл
	ap.RunApplication()
}

// menuItems создает элементы меню
func menuItems() []menuet.MenuItem {
	updateMu.Lock()
	defer updateMu.Unlock()

	// Получаем информацию о батарее
	info, err := battery.GetBatteryInfo()
	if err != nil {
		return []menuet.MenuItem{
			{Text: "⚠️ Ошибка получения данных"},
			{Type: menuet.Separator},
			{Text: "Выход", Clicked: exitApp},
		}
	}

	// Сохраняем для использования в других функциях
	batteryInfo = info

	// Определяем пороги из конфигурации или значения по умолчанию
	minThreshold := 20
	maxThreshold := 80
	if appConfig != nil {
		minThreshold = appConfig.MinThreshold
		maxThreshold = appConfig.MaxThreshold
	}

	// Определяем иконку для текущего заряда
	icon := getBatteryIcon(info.CurrentCapacity, info.IsCharging)

	// Используем выравнивание с пробелами для создания двух колонок
	// Выравнивание в столбик не идеально, но это лучшее, что можно сделать с menuet
	currentStatusText := fmt.Sprintf("%d%% %s", info.CurrentCapacity, icon)

	return []menuet.MenuItem{
		{
			// Заголовок пункта можно использовать как статус
			Text:     currentStatusText,
			FontSize: 14,
			State:    true, // Отмечен
		},
		{Type: menuet.Separator}, // Разделитель
		{
			// Используем форматирование строк для создания колонок
			// Первая колонка 20 символов, вторая выровнена по правому краю
			Text:     fmt.Sprintf("%-20s %4d%%", "Текущий заряд:", info.CurrentCapacity),
			FontSize: 13,
		},
		{
			Text:     fmt.Sprintf("%-20s %4d%%", "Мин. порог:", minThreshold),
			FontSize: 13,
		},
		{
			Text:     fmt.Sprintf("%-20s %4d%%", "Макс. порог:", maxThreshold),
			FontSize: 13,
		},
		{
			Text:     fmt.Sprintf("%-20s %4d", "Циклов заряда:", info.CycleCount),
			FontSize: 13,
		},
		{
			Text:     fmt.Sprintf("%-20s %4d%%", "Здоровье батареи:", info.HealthPercent),
			FontSize: 13,
		},
		{Type: menuet.Separator}, // Разделитель
		{Text: "Выход", Clicked: exitApp},
	}
}

// startStatusUpdater запускает периодическое обновление данных в меню
func startStatusUpdater() {
	// Первое обновление сразу после запуска
	menuet.App().MenuChanged()

	// Затем периодические обновления
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		menuet.App().MenuChanged()
	}
}

// getBatteryIcon возвращает иконку батареи в зависимости от уровня заряда
func getBatteryIcon(percent int, isCharging bool) string {
	switch {
	case percent <= 10:
		if isCharging {
			return "🔌⚡"
		}
		return "🔴"
	case percent <= 30:
		if isCharging {
			return "🔋⚡"
		}
		return "🟠"
	case percent <= 60:
		if isCharging {
			return "🔋⚡"
		}
		return "🟡"
	default:
		if isCharging {
			return "🔋⚡"
		}
		return "🟢"
	}
}

// exitApp обрабатывает клик на пункте "Выход"
func exitApp() {
	// Завершаем фоновый процесс
	killBackground()

	// Завершаем работу программы
	time.Sleep(100 * time.Millisecond)
	os.Exit(0)
}

// killBackground завершает фоновый процесс мониторинга батареи
func killBackground() {
	pidPath := paths.PIDPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return // файла нет – процесса нет
	}

	pid, _ := strconv.Atoi(string(data))
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Signal(syscall.SIGTERM) // корректное завершение
	}
	_ = os.Remove(pidPath)
}

//go:embed sys-tray-icon.png
var iconData []byte

// getAppIconFromFile возвращает данные иконки для отображения в трее
func getAppIconFromFile() []byte {
	return iconData
}
