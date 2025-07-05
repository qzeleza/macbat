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
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
)

// updateMenu обновляет состояние меню в трее
var updateMu sync.Mutex // защита от параллельного вызова

func updateMenu(mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode *systray.MenuItem, conf *config.Config) {
	updateMu.Lock()
	defer updateMu.Unlock()

	info, err := battery.GetBatteryInfo()
	if err != nil {
		mCurrent.SetTitle("Ошибка получения данных")
		return
	}

	// --- Определяем строки для отображения ---
	chargeModeStr := "Разрядка"
	if info.IsCharging {
		chargeModeStr = "Зарядка"
	}

	workModeStr := "Штатный"
	if modeRun == "test" {
		workModeStr = "Симуляция"
	}

	// --- Динамический расчет отступов для выравнивания ---
	labels := []string{
		"Текущий заряд:",
		"Мин. порог:",
		"Макс. порог:",
		"Циклов заряда:",
		"Здоровье батареи:",
		"Режим заряда:",
		"Режим работы:",
	}
	maxLength := 0
	for _, label := range labels {
		length := utf8.RuneCountInString(label)
		if length > maxLength {
			maxLength = length
		}
	}

	// Обновляем заголовок с иконкой батареи
	icon := getBatteryIcon(info.CurrentCapacity, info.IsCharging)
	mCurrent.SetTitle(fmt.Sprintf("%-*s %s %4d%%", maxLength, labels[0], icon, info.CurrentCapacity)) // Текущий заряд

	// Получаем пороги из конфигурации
	minThreshold := 21 // Значение по умолчанию
	maxThreshold := 81 // Значение по умолчанию
	if conf != nil {
		minThreshold = conf.MinThreshold
		maxThreshold = conf.MaxThreshold
	}

	chargeIcon := ""
	if info.IsCharging {
		chargeIcon = "⚡"
	}
	// Обновляем информацию в меню с использованием динамического отступа
	mChargeMode.SetTitle(fmt.Sprintf("%-21s %s %s", labels[5], chargeModeStr, chargeIcon)) // Режим заряда
	mWorkMode.SetTitle(fmt.Sprintf("%-20s %s", labels[6], workModeStr))                    // Режим работы

	mMin.SetTitle(fmt.Sprintf("%-21s       %4d%%", labels[1], minThreshold)) // Мин. порог
	mMax.SetTitle(fmt.Sprintf("%-21s       %4d%%", labels[2], maxThreshold)) // Макс. порог

	mCycles.SetTitle(fmt.Sprintf("%-22s    %4d", labels[3], info.CycleCount))   // Циклов заряда
	mHealth.SetTitle(fmt.Sprintf("%-20s %4d%%", labels[4], info.HealthPercent)) // Здоровье батареи

	log.Info("Данные меню успешно обновлены.")
}

// getBatteryIcon возвращает иконку батареи в зависимости от уровня заряда
func getBatteryIcon(percent int, isCharging bool) string {
	// Для зарядки используем один простой символ, чтобы проверить отображение.
	if isCharging {
		return "🔌"
	}

	// Для разных уровней разрядки используем стандартные цветные круги.
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

// onReady инициализирует иконку в трее
func onReady() {
	iconData := getAppIconFromFile()
	// Используем цветную иконку, а не шаблонную (template), чтобы macOS не перекрашивал её.
	systray.SetTitle("Страж")
	systray.SetIcon(iconData)
	systray.SetTooltip("Отслеживание достижения порогов заряда батареи")

	systray.AddSeparator()

	mChargeMode := systray.AddMenuItem("Загрузка...", "Разрядка и зарядка")
	// mChargeMode.Disable()

	mWorkMode := systray.AddMenuItem("Режим работы: --", "Штатный и симуляция (тестовый режим), запускается с флагом --test")
	// mWorkMode.Disable()

	systray.AddSeparator()

	mCurrent := systray.AddMenuItem("Текущий заряд: --%", "Текущий уровень заряда батареи")
	// mCurrent.Disable()
	systray.AddSeparator()

	mMin := systray.AddMenuItem("Мин. порог: --%", "Минимальный порог заряда батареи, при достижении которого будет запущен режим разрядки")
	// mMin.Disable()

	mMax := systray.AddMenuItem("Макс. порог: --%", "Максимальный порог заряда батареи, при достижении которого будет запущен режим зарядки")
	// mMax.Disable()
	systray.AddSeparator()

	mHealth := systray.AddMenuItem("Здоровье батареи: --%", "Здоровье батареи, показывает степень износа батареи")
	// mHealth.Disable()

	mCycles := systray.AddMenuItem("Циклов заряда: --", "Количество циклов заряда батареи")
	// mCycles.Disable()

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
		updateMenu(mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode, conf)
	}()

	// Запускаем тикер для обновления меню каждые 30 секунд
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateMenu(mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode, conf)
		}
	}()

	// --- Обработка всех кликов по меню в одной горутине ---
	go func() {
		for {
			select {

			// Нажатие на "Текущий заряд"
			case <-mCurrent.ClickedCh:
				dlgs.Warning("Внимание", "Текущий заряд батареи отображает процент оставшейся ёмкости относительно полной. Следите за этим показателем, чтобы не допускать глубокого разряда или перезаряда аккумулятора.\nРекомендуемые значения: от 20% до 80%.")

			// Нажатие на "Режим работы"
			case <-mWorkMode.ClickedCh:
				dlgs.Warning("Внимание", "Режим работы может быть штатным или тестовым (симуляция). В тестовом режиме можно проверить работу уведомлений и автоматического управления зарядом.")

			// Нажатие на "Режим заряда"
			case <-mChargeMode.ClickedCh:
				dlgs.Warning("Внимание", "Режим заряда показывает, заряжается ли сейчас аккумулятор или разряжается. При подключении к сети будет отображаться 'Зарядка', иначе — 'Разрядка'.")

			// Нажатие на "Здоровье батареи"
			case <-mHealth.ClickedCh:
				dlgs.Warning("Внимание", "Этот показатель отражает текущее здоровье батареи — чем ниже, тем выше износ аккумулятора. Снижение ниже 80% обычно означает заметную деградацию ёмкости. Для поддержания ресурса используйте аккуратные циклы заряда.")

			// Нажатие на "Циклов заряда"
			case <-mCycles.ClickedCh:
				dlgs.Warning("Внимание", "Количество циклов — это суммарное число полных разрядов/зарядов батареи. Большинство современных аккумуляторов рассчитаны примерно на 1000 циклов до существенного снижения ёмкости.")

			// Нажатие на "Мин. порог"
			case <-mMin.ClickedCh:
				handleThresholdChange(cfgManager, conf, log, mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode, "min")

			// Нажатие на "Макс. порог"
			case <-mMax.ClickedCh:
				handleThresholdChange(cfgManager, conf, log, mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode, "max")

			// Нажатие на "Выход"
			case <-mQuit.ClickedCh:
				killBackground()
				systray.Quit()
				time.Sleep(100 * time.Millisecond)
				os.Exit(0)
				return
			}
		}
	}()
}

// handleThresholdChange обрабатывает логику изменения порогов.
// @param cfgManager - менеджер конфигурации для сохранения.
// @param conf - текущая конфигурация.
// @param menuItems - все элементы меню для обновления.
// @param mode - какой порог меняем ("min" или "max").
func handleThresholdChange(cfgManager *config.Manager, conf *config.Config, log *logger.Logger, mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode *systray.MenuItem, mode string) {
	var title, prompt, currentValStr string
	var currentVal int

	log.Line()

	sunMessage := "При достижении этого порога будет показано системное уведомление."
	if mode == "min" {
		title = "Минимальный порог"
		prompt = "Введите новое значение минимального порога (0-100).\n" + sunMessage
		currentVal = conf.MinThreshold
	} else {
		title = "Максимальный порог"
		prompt = "Введите новое значение максимального порога (0-100).\n" + sunMessage
		currentVal = conf.MaxThreshold
	}
	log.Info(fmt.Sprintf("Меняем %s...", strings.ToLower(mode)))
	currentValStr = strconv.Itoa(currentVal)

	newValStr, ok, err := dlgs.Entry(title, prompt, currentValStr)
	if err != nil {
		dlgs.Error("Ошибка", "Не удалось отобразить диалоговое окно.")
		return
	}
	if !ok {
		log.Debug("Пользователь нажал 'Отмена'")
		// Пользователь нажал "Отмена"
		return
	}

	newVal, err := strconv.Atoi(newValStr)
	if err != nil {
		log.Debug("Ошибка ввода, введено не целое число.")
		dlgs.Error("Ошибка ввода", "Пожалуйста, введите целое число.")
		return
	}

	// Валидация введенного значения
	if mode == "min" {
		if newVal < 0 || newVal >= conf.MaxThreshold {
			log.Debug(fmt.Sprintf("Ошибка значения, значение должно быть между 0 и %d.", conf.MaxThreshold-1))
			dlgs.Error("Ошибка значения", fmt.Sprintf("Значение должно быть между 0 и %d.", conf.MaxThreshold-1))
			return
		}
		conf.MinThreshold = newVal
	} else { // max
		if newVal <= conf.MinThreshold || newVal > 100 {
			log.Debug(fmt.Sprintf("Ошибка значения, значение должно быть между %d и 100.", conf.MinThreshold+1))
			dlgs.Error("Ошибка значения", fmt.Sprintf("Значение должно быть между %d и 100.", conf.MinThreshold+1))
			return
		}
		conf.MaxThreshold = newVal
	}

	log.Info(fmt.Sprintf("%s установлен в %d.", mode, newVal))

	// Сохраняем новую конфигурацию
	if err := cfgManager.Save(conf); err != nil {
		log.Error("Ошибка сохранения конфигурации: " + err.Error())
		dlgs.Error("Ошибка сохранения", "Не удалось сохранить новую конфигурацию: "+err.Error())
	} else {
		// Обновляем меню немедленно, чтобы показать изменения
		updateMenu(mCurrent, mMin, mMax, mCycles, mHealth, mChargeMode, mWorkMode, conf)
	}
}

//go:embed sys-tray-icon.png
var iconData []byte

func getAppIconFromFile() []byte {
	return iconData
}
