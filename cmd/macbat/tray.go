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

	"github.com/getlantern/systray"
)

// updateMenu обновляет состояние меню в трее
var updateMu sync.Mutex // защита от параллельного вызова

func updateMenu(mCurrent, mMin, mMax, mCycles, mHealth *systray.MenuItem, conf *config.Config) {
	updateMu.Lock()
	defer updateMu.Unlock()

	info, err := battery.GetBatteryInfo()
	if err != nil {
		mCurrent.SetTitle("Ошибка получения данных")
		return
	}

	// Обновляем заголовок с иконкой батареи
	icon := getBatteryIcon(info.CurrentCapacity, info.IsCharging)
	mCurrent.SetTitle(fmt.Sprintf("%-18s %4d%% %s", "Текущий заряд:", info.CurrentCapacity, icon))

	// Получаем пороги из конфигурации
	minThreshold := 20 // Значение по умолчанию
	maxThreshold := 80 // Значение по умолчанию
	if conf != nil {
		minThreshold = conf.MinThreshold
		maxThreshold = conf.MaxThreshold
	}

	// Обновляем информацию в меню
	// Используем форматирование для выравнивания значений по правому краю.
	// Самая длинная метка - "Здоровье батареи:" (17 символов).
	// Мы делаем левую часть (метку) шириной 18 символов, а правую (значение) - 4 символа.
	mMin.SetTitle(fmt.Sprintf("%-18s %4d%%", "Мин. порог:", minThreshold))
	mMax.SetTitle(fmt.Sprintf("%-18s %4d%%", "Макс. порог:", maxThreshold))
	mCycles.SetTitle(fmt.Sprintf("%-18s %4d", "Циклов заряда:", info.CycleCount))
	mHealth.SetTitle(fmt.Sprintf("%-18s %4d%%", "Здоровье батареи:", info.HealthPercent))
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

// onReady инициализирует иконку в трее
func onReady() {
	iconData := getAppIconFromFile()
	// Используем цветную иконку, а не шаблонную (template), чтобы macOS не перекрашивал её.
	systray.SetIcon(iconData)
	systray.SetTitle("MBT")
	systray.SetTooltip("MacBat - Управление батареей")

	// Добавляем элементы меню
	// mBattery := systray.AddMenuItem("Загрузка...", "")
	// mBattery.Disable()

	systray.AddSeparator()

	mCurrent := systray.AddMenuItem("Текущий заряд: --%", "")
	mCurrent.Disable()

	mMin := systray.AddMenuItem("Мин. порог: --%", "")
	mMin.Disable()

	mMax := systray.AddMenuItem("Макс. порог: --%", "")
	mMax.Disable()

	mCycles := systray.AddMenuItem("Циклов заряда: --", "")
	mCycles.Disable()

	mHealth := systray.AddMenuItem("Здоровье батареи: --%", "")
	mHealth.Disable()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход", "Завершить работу приложения")

	// Создаем логгер для получения конфигурации
	log := logger.New(paths.LogPath(), 100, true, false)

	// Создаем менеджер конфигурации
	// Загружаем конфигурацию для отображения порогов
	cfgManager, _ := config.New(log, paths.ConfigPath())
	conf, _ := cfgManager.Load()

	// Переносим первое обновление меню на короткую задержку,
	// чтобы гарантировать завершение инициализации GUI и избежать блокировки.
	go func() {
		time.Sleep(100 * time.Millisecond)
		updateMenu(mCurrent, mMin, mMax, mCycles, mHealth, conf)
	}()

	// Запускаем тикер для обновления меню каждые 30 секунд
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateMenu(mCurrent, mMin, mMax, mCycles, mHealth, conf)
		}
	}()

	go func() {
		<-mQuit.ClickedCh
		// Завершаем фоновый процесс, запущенный с --background
		killBackground()
		systray.Quit()
		// Допустим, systray.Run() иногда не завершает процесс мгновенно,
		// поэтому завершаем его явно.
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}

//go:embed sys-tray-icon.png
var iconData []byte

func getAppIconFromFile() []byte {
	return iconData
}

func killBackground() {

	pidPath := paths.PIDPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	} // файла нет – процесса нет
	pid, _ := strconv.Atoi(string(data))
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Signal(syscall.SIGTERM) // корректное завершение
	}
	_ = os.Remove(pidPath)
}
