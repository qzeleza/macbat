// Пакет tray содержит реализацию иконки в системном трее
package tray

import (
	_ "embed"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qzeleza/macbat/internal/background"
	"github.com/qzeleza/macbat/internal/battery"
	"github.com/qzeleza/macbat/internal/config"
	"github.com/qzeleza/macbat/internal/logger"
	"github.com/qzeleza/macbat/internal/monitor"
	"github.com/qzeleza/macbat/internal/paths"
	"github.com/qzeleza/macbat/internal/utils"
	"github.com/qzeleza/macbat/internal/version"

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
)

// Tray управляет иконкой и меню в системном трее.
type Tray struct {
	log               *logger.Logger
	cfg               *config.Config
	cfgManager        *config.Manager
	bgManager         *background.Manager
	mChargeMode       *systray.MenuItem
	mCurrent          *systray.MenuItem
	mMin              *systray.MenuItem
	mMax              *systray.MenuItem
	mCycles           *systray.MenuItem
	mHealth           *systray.MenuItem
	mCheckCharging    *systray.MenuItem
	mCheckDischarging *systray.MenuItem
	mMaxNotifications *systray.MenuItem
	mSettings         *systray.MenuItem
	mConfig           *systray.MenuItem
	mLogs             *systray.MenuItem
	timeToFullCharge  *systray.MenuItem
	timeToEmptyCharge *systray.MenuItem
	mVersion          *systray.MenuItem
	updateMu          sync.Mutex
}

// New создает новый экземпляр Tray.
func New(appLog *logger.Logger, cfg *config.Config, cfgManager *config.Manager, bgManager *background.Manager) *Tray {
	return &Tray{
		log:        appLog,
		cfg:        cfg,
		cfgManager: cfgManager,
		bgManager:  bgManager,
	}
}

// Start запускает GUI-агент в системном трее.
func (t *Tray) Start() {
	systray.Run(t.onReady, t.onExit)
}

// onExit будет вызван при выходе из systray.
func (t *Tray) onExit() {
	// Здесь можно выполнить очистку, если это необходимо.
}

// onReady инициализирует иконку в трее, создает основные пункты меню, настраивает
// обработку кликов, а также запускает периодические обновления меню.
func (t *Tray) onReady() {

	// Устанавливаем иконку для системного меню
	systray.SetTitle("🔋👀") // Заголовок в виде эмодзи
	systray.SetTooltip("Управление macbat")

	// --- Создание элементов меню ---
	t.mVersion = systray.AddMenuItem("Версия ...", "Версия macbat")
	t.mVersion.Disable()
	systray.AddSeparator()
	// Режим работы
	t.mChargeMode = systray.AddMenuItem("Режим работы ...", "Текущий режим заряда")
	systray.AddSeparator()

	// --- Информационные пункты о времени зарядки/разрядки ---
	// Текущий заряд батареи
	t.mCurrent = systray.AddMenuItem("Загрузка...", "Текущий заряд батареи")
	t.timeToFullCharge = systray.AddMenuItem("Время до полной зарядки ...", "Расчётное время до 100% заряда")
	t.timeToEmptyCharge = systray.AddMenuItem("Время до полной разрядки ...", "Расчётное время до 0% заряда")
	t.timeToEmptyCharge.Hide()
	t.timeToFullCharge.Hide()

	// --- Пункты настройки порогов ---
	systray.AddSeparator()
	t.mMin = systray.AddMenuItem("Мин. порог ...", "Установить минимальный порог")
	t.mMax = systray.AddMenuItem("Макс. порог ...", "Установить максимальный порог")
	systray.AddSeparator()

	// --- Информационные пункты о состоянии батареи ---
	systray.AddSeparator()
	t.mCycles = systray.AddMenuItem("Циклов заряда ...", "Количество циклов перезарядки")
	t.mHealth = systray.AddMenuItem("Здоровье батареи ...", "Состояние аккумулятора")
	systray.AddSeparator()

	// --- Подменю интервалов и уведомлений ---
	t.mSettings = systray.AddMenuItem("Пороговые интервалы", "Настроить пороговые значения")
	t.mCheckCharging = t.mSettings.AddSubMenuItem("Интервал проверки при зарядке", "Установка интервала проверки, когда батарея заряжается")
	t.mCheckDischarging = t.mSettings.AddSubMenuItem("Интервал проверки при разрядке", "Установка интервала проверки, когда батарея разряжается")
	t.mMaxNotifications = t.mSettings.AddSubMenuItem("Число уведомлений", "Установка максимального количества повторов уведомлений о достижении порогов")

	// --- Подменю настроек и журнала ---
	t.mSettings = systray.AddMenuItem("Настройки и журнал", "Открыть")
	t.mConfig = t.mSettings.AddSubMenuItem("Открыть config.json", "Открыть файл конфигурации")
	t.mLogs = t.mSettings.AddSubMenuItem("Открыть macbat.log", "Открыть журнал ошибок и сообщений")

	// --- Кнопка "Выход" ---
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

	// Запускаем горутину для обновления меню каждые 5 секунд
	go func() {
		runtime.LockOSThread()                    // ➊ работаем всегда в одном ОС-потоке
		ticker := time.NewTicker(5 * time.Second) // ➋ каждые 5 секунд
		defer ticker.Stop()                       // ➌ останавливаем тикер при завершении горутины

		for range ticker.C {
			t.updateMenu() // ➋ обращаемся к IOKit в «правильном» потоке и обновляем меню
		}

	}()

	// Запускаем горутину для обработки кликов
	go t.handleMenuClicks(t.mSettings, t.mLogs, t.mConfig, mQuit)
}

// updateMenu обновляет меню приложения с информацией о текущем состоянии
// батареи, порогах и настройках.
func (t *Tray) updateMenu() {
	// Защищаем обновление меню от конкурентных вызовов
	t.updateMu.Lock()
	defer t.updateMu.Unlock()

	// Получаем информацию о батарее
	info, err := battery.GetBatteryInfo()
	if err != nil {
		t.mCurrent.SetTitle("Ошибка получения данных")
		return
	}

	// Получаем строку для отображения режима зарядки
	chargeModeStr := "Ноутбук питается от батареи"
	chargeModeIcon := "🪫"
	if info.IsCharging {
		chargeModeStr = "Ноутбук заряжаем от сети"
		chargeModeIcon = "🔌"
	}

	// Получаем пороги из конфигурации
	minThreshold := t.cfg.MinThreshold
	maxThreshold := t.cfg.MaxThreshold

	t.mVersion.SetTitle("Версия macbat " + version.Version)
	// Обновляем заголовок с иконкой батареи
	icon := getBatteryIcon(info.CurrentCapacity, info.IsCharging)
	t.mChargeMode.SetTitle(fmt.Sprintf("%-29s %-4s", chargeModeStr, chargeModeIcon))

	t.mCurrent.SetTitle(fmt.Sprintf("%-29s %4d%%  %-4s", "Текущий заряд", info.CurrentCapacity, icon))
	if info.IsCharging {
		t.timeToFullCharge.SetTitle(fmt.Sprintf("%-27s  %-5s", "До полного заряда", utils.FormatTimeToColonHMS(info.TimeToFull)))
		t.timeToEmptyCharge.Hide()
		t.timeToFullCharge.Show()
	} else {
		t.timeToEmptyCharge.SetTitle(fmt.Sprintf("%-26s  %-5s", "До полного разряда", utils.FormatTimeToColonHMS(info.TimeToEmpty)))
		t.timeToFullCharge.Hide()
		t.timeToEmptyCharge.Show()
	}

	// Записываем текущий заряд в лог
	t.log.Info(fmt.Sprintf("Текущий заряд: %d%%", info.CurrentCapacity))

	// Получаем индикаторы для порогов
	minIndicator := getMinThresholdIndicator(minThreshold)
	maxIndicator := getMaxThresholdIndicator(maxThreshold)

	// Обновляем пункты меню
	t.mMin.SetTitle(fmt.Sprintf("%-34s %4d%%  %s", "Мин. порог", minThreshold, minIndicator))
	t.mMax.SetTitle(fmt.Sprintf("%-33s %4d%%  %s", "Макс. порог", maxThreshold, maxIndicator))

	// Обновляем пункты меню
	healthIndicator := getHealthIndicator(info.HealthPercent)
	cyclesIndicator := getCyclesIndicator(info.CycleCount)
	t.mCycles.SetTitle(fmt.Sprintf("%-32s %4d  %s", "Циклов заряда", info.CycleCount, cyclesIndicator))
	t.mHealth.SetTitle(fmt.Sprintf("%-28s %4d%%  %s", "Здоровье батареи", info.HealthPercent, healthIndicator))

	// Обновляем пункты меню
	t.mCheckCharging.SetTitle(fmt.Sprintf("%-36s %4d с.", "Интервал проверки при зарядке", t.cfg.CheckIntervalWhenCharging))
	t.mCheckDischarging.SetTitle(fmt.Sprintf("%-35s %4d с.", "Интервал проверки при разрядке", t.cfg.CheckIntervalWhenDischarging))
	t.mMaxNotifications.SetTitle(fmt.Sprintf("%-45s %4d ув.", "Число уведомлений", t.cfg.MaxNotifications))
}

// getMinThresholdIndicator возвращает цветной индикатор для минимального порога.
func getMinThresholdIndicator(threshold int) string {
	switch {
	case threshold <= 10:
		return "🔴" // Оптимально0
	case threshold >= 11 || threshold <= 20:
		return "🟡" // Оптимально
	case threshold <= 28:
		return "🟢" // Оптимально
	default:
		return "🔴" // Неоптимально
	}
}

// getMaxThresholdIndicator возвращает цветной индикатор для максимального порога.
func getMaxThresholdIndicator(threshold int) string {
	switch {
	case threshold <= 70:
		return "🔴" // Неоптимально
	case threshold <= 81:
		return "🟢" // Оптимально
	case threshold <= 90:
		return "🟡" // Нормально
	default:
		return "🔴" // Неоптимально
	}
}

// getHealthIndicator возвращает цветной индикатор для здоровья батареи.
func getHealthIndicator(health int) string {
	switch {
	case health > 90:
		return "🟢" // Отлично
	case health > 70:
		return "🟡" // Нормально
	default:
		return "🔴" // Требует внимания
	}
}

// getCyclesIndicator возвращает цветной индикатор для циклов заряда.
func getCyclesIndicator(cycles int) string {
	switch {
	case cycles < 300:
		return "🟢" // Низкое
	case cycles < 700:
		return "🟡" // Среднее
	default:
		return "🔴" // Высокое
	}
}

// getBatteryIcon возвращает иконку батареи в зависимости от уровня заряда
func getBatteryIcon(percent int, isCharging bool) string {
	if isCharging {
		return "🔌"
	}
	switch {
	case percent <= 10:
		return "🔴"
	case percent <= 20:
		return "🟠"
	case percent <= 40:
		return "🟡"
	case percent <= 60:
		return "🔵" // Синий круг
	case percent <= 80:
		return "🟢"
	case percent <= 100:
		return "🟤"
	default:
		return "🟣"
	}
}

// handleMenuClicks обрабатывает нажатия на пункты меню.
//
// @param mSettings - пункт "Настройки"
// @param mLogs - пункт "Логи"
// @param mConfig - пункт "Конфигурация"
// @param mQuit - пункт "Выход"
//
// handleMenuClicks является goroutin'ом, который постоянно слушает каналы
// нажатий на пункты меню.
func (t *Tray) handleMenuClicks(mSettings, mLogs, mConfig, mQuit *systray.MenuItem) {

	for {
		select {
		// --- Выбрали пункт "Конфигурация" ---
		case <-t.mConfig.ClickedCh:
			if err := paths.OpenFileOrDir(paths.ConfigPath()); err != nil {
				dlgs.Error("Ошибка", "Не удалось открыть файл конфигурации.")
			}

		// --- Выбрали пункт "Логи" ---
		case <-t.mLogs.ClickedCh:
			if err := paths.OpenFileOrDir(paths.LogPath()); err != nil {
				dlgs.Error("Ошибка", "Не удалось открыть директорию логов.")
			}

		// --- Выбрали пункт "Здоровье батареи" ---
		case <-t.mHealth.ClickedCh:
			dlgs.Info("Здоровье батареи", "Здоровье батареи в современных ноутбуках определяется по состоянию износа аккумулятора. Если значение больше 90%, то это хороший результат, если меньше 50%, то пора задуматься над заменой аккумулятора.")

		// --- Выбрали пункт "Циклы заряда" ---
		case <-t.mCycles.ClickedCh:
			dlgs.Info("Циклы заряда", "Циклы заряда определяются по количеству перезарядок. Если значение меньше 500 циклов, то это хороший результат, если больше 1000, то пора задуматься над заменой аккумулятора.")

		// --- Выбрали пункт "Минимальный порог" ---
		case <-t.mMin.ClickedCh:
			t.handleThresholdChange("min")

		// --- Выбрали пункт "Максимальный порог" ---
		case <-t.mMax.ClickedCh:
			t.handleThresholdChange("max")

		// --- Выбрали пункт "Интервал проверки (зарядка)" ---
		case <-t.mCheckCharging.ClickedCh:
			t.handleIntegerConfigChange("check_interval_charging", "Интервал проверки (зарядка)", "Введите интервал в секундах:")

		// --- Выбрали пункт "Интервал проверки (разрядка)" ---
		case <-t.mCheckDischarging.ClickedCh:
			t.handleIntegerConfigChange("check_interval_discharging", "Интервал проверки (разрядка)", "Введите интервал в секундах:")

		// --- Выбрали пункт "Количество уведомлений" ---
		case <-t.mMaxNotifications.ClickedCh:
			t.handleIntegerConfigChange("max_notifications", "Количество уведомлений", "Введите максимальное количество уведомлений:")

		// Нажатие на "Выход"
		case <-mQuit.ClickedCh:
			if confirmed, err := dlgs.Question("Выход", "Вы уверены, что хотите закрыть приложение?", true); err != nil {
				dlgs.Error("Ошибка", "Не удалось отобразить диалоговое окно.")
			} else if confirmed {
				t.log.Info("Получен сигнал на выход. Завершение работы.")
				t.bgManager.Kill("--background")
				if _, err := monitor.CommandAgentService(t.log, "bootout"); err != nil {
					dlgs.Error("Ошибка", "Не удалось выгрузить агента: "+err.Error())
				}
				systray.Quit()
				return
			}
		}
	}
}

// handleIntegerConfigChange обрабатывает изменение целочисленных значений конфигурации.
//
// @param key - ключ конфигурации, который нужно изменить
// @param title - заголовок диалогового окна
// @param prompt - текст приглашения в диалоговом окне
//
// Диалоговое окно ввода будет отображено с текущим значением конфигурации.
// Если пользователь отменит ввод, то ничего не будет изменено.
// Если пользователь введет корректное целое число, то конфигурация будет обновлена.
// Если пользователь введет что-то неправильное, то будет отображена ошибка.
func (t *Tray) handleIntegerConfigChange(key, title, prompt string) {
	var currentVal int
	switch key {
	case "check_interval_charging":
		currentVal = t.cfg.CheckIntervalWhenCharging
	case "check_interval_discharging":
		currentVal = t.cfg.CheckIntervalWhenDischarging
	case "max_notifications":
		currentVal = t.cfg.MaxNotifications
	default:
		dlgs.Error(title, "Внутренняя ошибка: неизвестный ключ конфигурации.")
		return
	}

	input, confirmed, err := dlgs.Entry(title, prompt, strconv.Itoa(currentVal))
	if err != nil {
		dlgs.Error("Ошибка", "Не удалось отобразить диалоговое окно: "+err.Error())
		return
	}
	if !confirmed {
		t.log.Debug("Изменение значения отменено пользователем.")
		return
	}

	newValue, err := strconv.Atoi(input)
	if err != nil {
		dlgs.Error("Ошибка ввода", "Пожалуйста, введите корректное число.")
		return
	}

	switch key {
	case "check_interval_charging":
		t.cfg.CheckIntervalWhenCharging = newValue
	case "check_interval_discharging":
		t.cfg.CheckIntervalWhenDischarging = newValue
	case "max_notifications":
		t.cfg.MaxNotifications = newValue
	}

	if err := t.cfgManager.Save(t.cfg); err != nil {
		dlgs.Error("Ошибка сохранения", "Не удалось сохранить конфигурацию: "+err.Error())
		t.log.Error("Ошибка сохранения конфигурации: " + err.Error())
	} else {
		t.log.Info(fmt.Sprintf("Значение успешно обновлено на %d.", newValue))
	}
	t.updateMenu()
}

// handleThresholdChange обрабатывает логику изменения порогов.
//
// Диалог предлагает пользователю ввести новое значение порога, которое
// будет сохранено в конфигурации.
func (t *Tray) handleThresholdChange(mode string) {
	var title, prompt, currentValStr string
	var currentVal int

	if mode == "min" {
		title = "Минимальный порог"
		prompt = "Введите минимальный порог заряда (0-100):"
		currentVal = t.cfg.MinThreshold
	} else {
		title = "Максимальный порог"
		prompt = "Введите новое значение максимального порога (0-100).\n"
		currentVal = t.cfg.MaxThreshold
	}
	t.log.Info(fmt.Sprintf("Меняем %s...", strings.ToLower(mode)))
	currentValStr = strconv.Itoa(currentVal)

	newValStr, ok, err := dlgs.Entry(title, prompt, currentValStr)
	if err != nil {
		dlgs.Error("Ошибка", "Не удалось отобразить диалоговое окно.")
		return
	}
	if !ok {
		t.log.Debug("Пользователь нажал 'Отмена'")
		return
	}

	newVal, err := strconv.Atoi(newValStr)
	if err != nil {
		t.log.Debug("Ошибка ввода, введено не целое число.")
		dlgs.Error("Ошибка ввода", "Пожалуйста, введите целое число.")
		return
	}

	// Валидация введенного значения
	if mode == "min" {
		if newVal < 0 || newVal >= t.cfg.MaxThreshold {
			dlgs.Error("Ошибка значения", fmt.Sprintf("Значение должно быть между 0 и %d.", t.cfg.MaxThreshold-1))
			return
		}
		t.cfg.MinThreshold = newVal
	} else { // max
		if newVal <= t.cfg.MinThreshold || newVal > 100 {
			t.log.Debug(fmt.Sprintf("Ошибка значения, значение должно быть между %d и 100.", t.cfg.MinThreshold+1))
			dlgs.Error("Ошибка значения", fmt.Sprintf("Значение должно быть между %d и 100.", t.cfg.MinThreshold+1))
			return
		}
		t.cfg.MaxThreshold = newVal
	}

	t.log.Info(fmt.Sprintf("%s установлен в %d.", mode, newVal))

	// Сохраняем новую конфигурацию
	if err := t.cfgManager.Save(t.cfg); err != nil {
		t.log.Error("Ошибка сохранения конфигурации: " + err.Error())
		dlgs.Error("Ошибка сохранения", "Не удалось сохранить новую конфигурацию: "+err.Error())
	} else {
		t.log.Info("Успешное сохранение порога " + mode + "= " + strconv.Itoa(newVal) + ".")
		// Обновляем меню немедленно, чтобы показать изменения
		t.updateMenu()
	}
}
