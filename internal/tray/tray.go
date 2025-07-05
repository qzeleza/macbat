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

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
)

// Tray управляет иконкой и меню в системном трее.
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

	chargeModeStr := "Разрядка"
	if info.IsCharging {
		chargeModeStr = "Зарядка"
	}

	// Получаем пороги из конфигурации
	minThreshold := t.conf.MinThreshold
	maxThreshold := t.conf.MaxThreshold

	chargeIcon := ""
	if info.IsCharging {
		chargeIcon = "⚡"
	}
	// Обновляем заголовок с иконкой батареи
	icon := getBatteryIcon(info.CurrentCapacity, info.IsCharging)
	t.mCurrent.SetTitle(fmt.Sprintf("%-30s %d%% %s", "Текущий заряд:", info.CurrentCapacity, icon))
	t.mChargeMode.SetTitle(fmt.Sprintf("%-30s %s%s", "Режим заряда:", chargeModeStr, chargeIcon))

	t.mMin.SetTitle(fmt.Sprintf("%-32s %3d%%", "Мин. порог:", minThreshold))
	t.mMax.SetTitle(fmt.Sprintf("%-32s %3d%%", "Макс. порог:", maxThreshold))

	t.mCycles.SetTitle(fmt.Sprintf("%-31s %4d", "Циклов заряда:", info.CycleCount))
	t.mHealth.SetTitle(fmt.Sprintf("%-27s %3d%%", "Здоровье батареи:", info.HealthPercent))
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

// onReady инициализирует иконку в трее и запускает главный цикл обработки событий.
func (t *Tray) onReady() {
	systray.SetIcon(getAppIconFromFile())
	systray.SetTitle("👀")
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

	mConfig := systray.AddMenuItem("Открыть config.json", "Открыть файл конфигурации")
	mLogs := systray.AddMenuItem("Открыть macbat.log", "Открыть журнал ошибок и сообщений")

	// --- Кнопка "Выход" ---
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

	// Создаем канал для получения обновлений конфигурации.
	configUpdateChan := make(chan *config.Config)
	// Запускаем наблюдателя за файлом конфигурации в отдельной горутине.
	go config.Watch(paths.ConfigPath(), configUpdateChan, t.log)

	// Обновляем динамические данные (состояние батареи) каждые 5 секунд.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Первоначальное обновление меню.
	t.updateMenu()

	// Запускаем главный цикл обработки событий в основном потоке.
	for {
		select {
		// Событие 1: Получили обновленную конфигурацию из файла.
		case newCfg, ok := <-configUpdateChan:
			if !ok {
				t.log.Debug("Канал обновлений конфигурации был закрыт.")
				return
			}
			t.log.Info("Получена новая конфигурация из файла. Обновление меню...")
			t.updateMu.Lock()
			t.conf = newCfg
			t.updateMu.Unlock()
			t.updateMenu()

		// Событие 2: Периодическое обновление по таймеру для динамических данных.
		case <-ticker.C:
			t.updateMenu()

		case <-t.mCycles.ClickedCh:
			dlgs.Info("Циклы заряда", "В современных ноутбуках циклы заряда батареи могут достигать 1000 и более.")

		case <-t.mHealth.ClickedCh:
			dlgs.Info("Здоровье батареи", "Здоровье батареи указывает на ее состояние и может влиять на срок службы.")

		// --- Обработка нажатий на подменю "Настройки" ---
		case <-mConfig.ClickedCh:
			if err := paths.OpenFileOrDir(paths.ConfigPath()); err != nil {
				dlgs.Error("Ошибка", "Не удалось открыть файл конфигурации.")
			}

		case <-mLogs.ClickedCh:
			if err := paths.OpenFileOrDir(paths.LogPath()); err != nil {
				dlgs.Error("Ошибка", "Не удалось открыть директорию логов.")
			}

		// --- Обработка нажатий на пороги ---
		case <-t.mMin.ClickedCh:
			t.handleThresholdChange("min")

		case <-t.mMax.ClickedCh:
			t.handleThresholdChange("max")

		// Нажатие на "Выход"
		case <-mQuit.ClickedCh:
			if confirmed, err := dlgs.Question("Выход", "Вы уверены, что хотите закрыть приложение?", true); err != nil {
				dlgs.Error("Ошибка", "Не удалось отобразить диалоговое окно.")
			} else if confirmed {
				t.log.Info("Получен сигнал на выход. Завершение работы.")
				t.bgManager.Kill("--background")
				if _, err := monitor.UnloadAgent(t.log); err != nil {
					dlgs.Error("Ошибка", "Не удалось удалить агента: "+err.Error())
				}
				systray.Quit()
				return
			}
		}
	}
}

// handleThresholdChange обрабатывает логику изменения порогов.
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
