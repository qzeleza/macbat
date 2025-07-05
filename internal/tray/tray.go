// Пакет tray содержит реализацию иконки в системном трее
package tray

import (
	_ "embed"
	"fmt"
	"macbat/internal/background"
	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/monitor"
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
	// Здесь можно выполнить очистку, если это необходимо.
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
	systray.SetTemplateIcon(getAppIconFromFile(), getAppIconFromFile())
	systray.SetTitle("Macbat")
	systray.SetTooltip("Управление macbat")

	// --- Создание элементов меню ---
	t.mCurrent = systray.AddMenuItem("Загрузка...", "Текущий заряд батареи")
	t.mChargeMode = systray.AddMenuItem("Режим заряда: ...", "Текущий режим заряда")
	systray.AddSeparator()
	t.mMin = systray.AddMenuItem("Мин. порог: ...", "Установить минимальный порог")
	t.mMax = systray.AddMenuItem("Макс. порог: ...", "Установить максимальный порог")
	systray.AddSeparator()
	t.mCycles = systray.AddMenuItem("Циклов заряда: ...", "Количество циклов перезарядки")
	t.mHealth = systray.AddMenuItem("Здоровье батареи: ...", "Состояние аккумулятора")
	systray.AddSeparator()

	// --- Меню "Фоновый процесс" ---
	mToggleBackground := systray.AddMenuItem("Фоновый процесс", "Управление фоновым процессом")
	mStartBg := mToggleBackground.AddSubMenuItem("Запустить", "Запустить фоновый процесс")
	mStopBg := mToggleBackground.AddSubMenuItem("Остановить", "Остановить фоновый процесс")
	mRestartBg := mToggleBackground.AddSubMenuItem("Перезапустить", "Перезапустить фоновый процесс")

	// --- Меню "Настройки" ---
	mSettings := systray.AddMenuItem("Настройки", "Открыть настройки")
	mConfig := mSettings.AddSubMenuItem("Открыть config.json", "Открыть файл конфигурации")
	mLogs := mSettings.AddSubMenuItem("Открыть логи", "Открыть директорию с логами")

	// --- Кнопка "Выход" ---
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

	// Создаем канал для получения обновлений конфигурации.
	configUpdateChan := make(chan *config.Config)
	// Запускаем наблюдателя за файлом конфигурации в отдельной горутине.
	go config.Watch(paths.ConfigPath(), configUpdateChan, t.log)

	// Первоначальное обновление меню.
	t.updateMenu()

	// Горутина для периодического обновления и обработки событий.
	go func() {
		// Обновляем динамические данные (состояние батареи) каждые 5 секунд.
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			// Событие 1: Получили обновленную конфигурацию из файла.
			case newCfg, ok := <-configUpdateChan:
				if !ok {
					t.log.Debug("Канал обновлений конфигурации был закрыт. Выход из цикла событий.")
					return
				}
				t.log.Info("Получена новая конфигурация из файла. Обновление меню...")
				// Блокируем мьютекс для безопасного обновления конфигурации.
				t.updateMu.Lock()
				t.conf = newCfg
				t.updateMu.Unlock()
				// Немедленно обновляем меню, чтобы отразить изменения.
				t.updateMenu()

			// Событие 2: Периодическое обновление по таймеру для динамических данных.
			case <-ticker.C:
				t.updateMenu()

			// --- Обработка нажатий на подменю "Фоновый процесс" ---
			case <-mStartBg.ClickedCh:
				t.log.Info("Запуск фонового процесса...")
				monitor.LoadAgent(t.log)

			case <-mStopBg.ClickedCh:
				t.log.Info("Остановка фонового процесса...")
				monitor.UnloadAgent(t.log)

			case <-mRestartBg.ClickedCh:
				t.log.Info("Перезапуск фонового процесса...")
				monitor.UnloadAgent(t.log)
				// Небольшая пауза перед запуском.
				time.Sleep(1 * time.Second)
				monitor.LoadAgent(t.log)

			// --- Обработка нажатий на подменю "Настройки" ---
			case <-mConfig.ClickedCh:
				if err := paths.OpenFileOrDir(paths.ConfigPath()); err != nil {
					dlgs.Error("Ошибка", "Не удалось открыть файл конфигурации.")
				}

			case <-mLogs.ClickedCh:
				if err := paths.OpenFileOrDir(paths.LogDir()); err != nil {
					dlgs.Error("Ошибка", "Не удалось открыть директорию логов.")
				}

			// Нажатие на "Мин. порог"
			case <-t.mMin.ClickedCh:
				t.handleThresholdChange("min")

			// Нажатие на "Макс. порог"
			case <-t.mMax.ClickedCh:
				t.handleThresholdChange("max")

			// Нажатие на "Выход"
			case <-mQuit.ClickedCh:
				if confirmed, err := dlgs.Question("Выход", "Вы уверены, что хотите закрыть приложение?", true); err != nil {
					dlgs.Error("Ошибка", "Не удалось отобразить диалоговое окно.")
				} else if !confirmed {
					// Если пользователь отказался выходить, просто продолжаем цикл обработки событий,
					// чтобы меню оставалось рабочим.
					continue
				}

				t.bgManager.Kill("--background")
				t.log.Info("Завершение работы приложения...")
				monitor.UnloadAgent(t.log)
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
