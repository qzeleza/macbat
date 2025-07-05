// Пакет tray содержит реализацию иконки в системном трее
package tray

import (
	_ "embed"
	"fmt"
	"macbat/internal/background"
	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
)

// Tray управляет иконкой и меню в системном трее.
// Содержит всю логику, связанную с GUI-агентом.
// @property log - логгер для записи событий.
// @property updateMu - мьютекс для безопасного обновления меню из разных горутин.
// @property mChargeMode - элемент меню, отображающий режим заряда.
// @property mCurrent - элемент меню, отображающий текущий заряд.
// @property mMin - элемент меню для минимального порога.
// @property mMax - элемент меню для максимального порога.
// @property mCycles - элемент меню для количества циклов заряда.
// @property mHealth - элемент меню для здоровья батареи.
// @property cfgManager - менеджер конфигурации.
// @property conf - текущая конфигурация.
type Tray struct {
	log         *logger.Logger
	bgManager   *background.Manager
	updateMu    sync.Mutex
	mChargeMode *systray.MenuItem
	mCurrent    *systray.MenuItem
	mMin        *systray.MenuItem
	mMax        *systray.MenuItem
	mCycles     *systray.MenuItem
	mHealth     *systray.MenuItem
	cfgManager  *config.Manager
	conf        *config.Config
}

// New создает новый экземпляр Tray.
// @param appLog - логгер для записи событий.
// @param cfg - текущая конфигурация.
// @param cfgManager - менеджер конфигурации.
// @param bgManager - менеджер фоновых процессов.
// @return *Tray - новый экземпляр Tray.
func New(appLog *logger.Logger, cfg *config.Config, cfgManager *config.Manager, bgManager *background.Manager) *Tray {
	return &Tray{
		log:        appLog,
		conf:       cfg,
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
	t.log.Info("Выход из приложения systray.")
}

func (t *Tray) updateMenu() {
	t.updateMu.Lock()
	defer t.updateMu.Unlock()

	info, err := battery.GetBatteryInfo()
	if err != nil {
		t.mCurrent.SetTitle("Ошибка получения данных")
		return
	}

	// --- Определяем строки для отображения ---
	chargeModeStr := "Разрядка"
	if info.IsCharging {
		chargeModeStr = "Зарядка"
	}

	// --- Динамический расчет отступов для выравнивания ---
	labels := []string{
		"Текущий заряд:",
		"Мин. порог:",
		"Макс. порог:",
		"Циклов заряда:",
		"Здоровье батареи:",
		"Режим заряда:",
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
	t.mCurrent.SetTitle(fmt.Sprintf("%-*s %s %4d%%", maxLength, labels[0], icon, info.CurrentCapacity)) // Текущий заряд

	// Получаем пороги из конфигурации
	minThreshold := t.conf.MinThreshold
	maxThreshold := t.conf.MaxThreshold

	chargeIcon := ""
	if info.IsCharging {
		chargeIcon = "⚡"
	}
	// Обновляем информацию в меню с использованием динамического отступа
	t.mChargeMode.SetTitle(fmt.Sprintf("%-21s %s %s", labels[5], chargeModeStr, chargeIcon)) // Режим заряда

	t.mMin.SetTitle(fmt.Sprintf("%-21s       %4d%%", labels[1], minThreshold)) // Мин. порог
	t.mMax.SetTitle(fmt.Sprintf("%-21s       %4d%%", labels[2], maxThreshold)) // Макс. порог

	t.mCycles.SetTitle(fmt.Sprintf("%-22s    %4d", labels[3], info.CycleCount))   // Циклов заряда
	t.mHealth.SetTitle(fmt.Sprintf("%-20s %4d%%", labels[4], info.HealthPercent)) // Здоровье батареи

	t.log.Info("Данные меню успешно обновлены.")
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
func (t *Tray) onReady() {
	iconData := getAppIconFromFile()
	// Используем цветную иконку, а не шаблонную (template), чтобы macOS не перекрашивал её.
	systray.SetTitle("👀")
	systray.SetIcon(iconData)
	systray.SetTooltip("Отслеживание достижения порогов заряда батареи")

	systray.AddSeparator()

	t.mChargeMode = systray.AddMenuItem("Режим заряда: ...", "Показывает текущий режим заряда")

	systray.AddSeparator()

	t.mCurrent = systray.AddMenuItem("Текущий заряд: --%", "Текущий уровень заряда батареи")

	systray.AddSeparator()

	t.mMin = systray.AddMenuItem("Мин. порог: --%", "Минимальный порог заряда батареи, при достижении которого будет запущен режим разрядки")

	t.mMax = systray.AddMenuItem("Макс. порог: --%", "Максимальный порог заряда батареи, при достижении которого будет запущен режим зарядки")

	systray.AddSeparator()

	t.mHealth = systray.AddMenuItem("Здоровье батареи: --%", "Здоровье батареи, показывает степень износа батареи")

	t.mCycles = systray.AddMenuItem("Циклов заряда: --", "Количество циклов заряда батареи")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход", "Завершить работу приложения")

	// Создаем менеджер конфигурации и загружаем её
	var err error
	t.cfgManager, err = config.New(t.log, paths.ConfigPath())
	if err != nil {
		t.log.Error("Не удалось создать менеджер конфигурации: " + err.Error())
		dlgs.Error("Критическая ошибка", "Не удалось создать менеджер конфигурации.")
		systray.Quit()
		return
	}

	t.conf, err = t.cfgManager.Load()
	if err != nil {
		t.log.Error("Не удалось загрузить конфигурацию: " + err.Error())
		dlgs.Error("Критическая ошибка", "Не удалось загрузить конфигурацию.")
		systray.Quit()
		return
	}

	// Запускаем периодическое обновление меню
	go func() {
		// Первое обновление с небольшой задержкой для инициализации GUI
		time.Sleep(100 * time.Millisecond)
		t.updateMenu()

		// Последующие обновления по тикеру
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			t.updateMenu()
		}
	}()

	// --- Обработка всех кликов по меню в одной горутине ---
	go func() {
		for {
			select {

			// Нажатие на "Текущий заряд"
			case <-t.mCurrent.ClickedCh:
				dlgs.Warning("Внимание", "Текущий заряд батареи отображает процент оставшейся ёмкости относительно полной. Следите за этим показателем, чтобы не допускать глубокого разряда или перезаряда аккумулятора.\nРекомендуемые значения: от 20% до 80%.")

			// Нажатие на "Режим заряда"
			case <-t.mChargeMode.ClickedCh:
				dlgs.Warning("Внимание", "Режим заряда показывает, заряжается ли сейчас аккумулятор или разряжается. При подключении к сети будет отображаться 'Зарядка', иначе — 'Разрядка'.")

			// Нажатие на "Здоровье батареи"
			case <-t.mHealth.ClickedCh:
				dlgs.Warning("Внимание", "Этот показатель отражает текущее здоровье батареи — чем ниже, тем выше износ аккумулятора. Снижение ниже 80% обычно означает заметную деградацию ёмкости. Для поддержания ресурса используйте аккуратные циклы заряда.")

			// Нажатие на "Циклов заряда"
			case <-t.mCycles.ClickedCh:
				dlgs.Warning("Внимание", "Количество циклов — это суммарное число полных разрядов/зарядов батареи. Большинство современных аккумуляторов рассчитаны примерно на 1000 циклов до существенного снижения ёмкости.")

			// Нажатие на "Мин. порог"
			case <-t.mMin.ClickedCh:
				t.handleThresholdChange("min")

			// Нажатие на "Макс. порог"
			case <-t.mMax.ClickedCh:
				t.handleThresholdChange("max")

			// Нажатие на "Выход"
			case <-mQuit.ClickedCh:
				t.bgManager.Kill("--background")
				systray.Quit()
				return
			}
		}
	}()
}

// handleThresholdChange обрабатывает логику изменения порогов.
// @param mode - какой порог меняем ("min" или "max").
func (t *Tray) handleThresholdChange(mode string) {
	var title, prompt, currentValStr string
	var currentVal int

	t.log.Line()

	sunMessage := "При достижении этого порога будет показано системное уведомление."
	if mode == "min" {
		title = "Минимальный порог"
		prompt = "Введите новое значение минимального порога (0-100).\n" + sunMessage
		currentVal = t.conf.MinThreshold
	} else {
		title = "Максимальный порог"
		prompt = "Введите новое значение максимального порога (0-100).\n" + sunMessage
		currentVal = t.conf.MaxThreshold
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
		// Пользователь нажал "Отмена"
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
		if newVal < 0 || newVal >= t.conf.MaxThreshold {
			t.log.Debug(fmt.Sprintf("Ошибка значения, значение должно быть между 0 и %d.", t.conf.MaxThreshold-1))
			dlgs.Error("Ошибка значения", fmt.Sprintf("Значение должно быть между 0 и %d.", t.conf.MaxThreshold-1))
			return
		}
		t.conf.MinThreshold = newVal
	} else { // max
		if newVal <= t.conf.MinThreshold || newVal > 100 {
			t.log.Debug(fmt.Sprintf("Ошибка значения, значение должно быть между %d и 100.", t.conf.MinThreshold+1))
			dlgs.Error("Ошибка значения", fmt.Sprintf("Значение должно быть между %d и 100.", t.conf.MinThreshold+1))
			return
		}
		t.conf.MaxThreshold = newVal
	}

	t.log.Info(fmt.Sprintf("%s установлен в %d.", mode, newVal))

	// Сохраняем новую конфигурацию
	if err := t.cfgManager.Save(t.conf); err != nil {
		t.log.Error("Ошибка сохранения конфигурации: " + err.Error())
		dlgs.Error("Ошибка сохранения", "Не удалось сохранить новую конфигурацию: "+err.Error())
	} else {
		// Обновляем меню немедленно, чтобы показать изменения
		t.updateMenu()
	}
}

//go:embed sys-tray-icon.png
var iconData []byte

func getAppIconFromFile() []byte {
	return iconData
}
