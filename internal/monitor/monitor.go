// Package monitor содержит основную логику фонового процесса:
// мониторинг батареи и отслеживание изменений в файле конфигурации.

/**
 * @file monitor.go
 * @brief Модуль для мониторинга состояния батареи ноутбука с уведомлениями о достижении пороговых значений.
 *
 * Этот модуль отслеживает уровень заряда и состояние подключения к сети.
 * Он отправляет уведомления только при достижении установленных пороговых значений:
 * - При зарядке: уведомление при достижении max_threshold (максимальный порог)
 * - При разрядке: уведомление при достижении min_threshold (минимальный порог)
 *
 * Модуль является гибко настраиваемым и легко тестируемым.
 *
 * @author Zeleza
 * @date 2025-08-07
 * @version 2.2.0
 *
 * @details
 * Основные принципы работы:
 * 1. Модуль использует только стандартные библиотеки Go, избегая системных вызовов.
 * 2. Проверяет состояние батареи ноутбука в непрерывном цикле.
 * 3. Если текущий уровень заряда не изменился относительно lastLevel, проверка пропускается.
 * 4. При разрядке: отправляет уведомление только при первом пересечении min_threshold.
 * 5. При зарядке: отправляет уведомление только при первом пересечении max_threshold.
 * 6. При смене режима заряда (зарядка ↔ разрядка) состояние сбрасывается.
 * 7. Использует разные функции для проверки состояний при заряде и разряде батареи.
 * 8. Поддерживает режим симуляции для тестирования с изменением заряда на 1% за итерацию.
 * 9. Для отображения системных уведомлений использует модуль dialog.
 * 10. Интервал проверки зависит от режима: CheckIntervalWhenCharging при зарядке,
 *     CheckIntervalWhenDischarging при разрядке.
 */

//================================================================================
// ПОДКЛЮЧЕНИЕ БИБЛИОТЕК
//================================================================================

package monitor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/qzeleza/macbat/internal/battery"
	"github.com/qzeleza/macbat/internal/config"
	"github.com/qzeleza/macbat/internal/dialog"
	"github.com/qzeleza/macbat/internal/logger"
)

//================================================================================
// СТРУКТУРЫ ДАННЫХ
//================================================================================

// Monitor - это основная структура фонового процесса.
type Monitor struct {
	config            config.Config   // Конфигурация монитора.
	log               *logger.Logger  // Объект для отправки уведомлений.
	cfgManager        *config.Manager // Менеджер конфигурации.
	lastKnownCharging bool            // Последнее известное состояние (заряжается/не заряжается).
	isInitialized     bool            // Флаг, показывающий, был ли монитор запущен хотя бы раз.
	lastLevel         int             // Последний известный уровень заряда для оптимизации.
	lastDirection     int             // Направление изменения уровня: -1 (падение), 0 (нет изменений), 1 (рост).
	stopChan          chan struct{}
	lastBrightness    int // Последняя известная яркость экрана
}

const (
	// defaultMonitorIntervalSeconds используется, когда конфигурация задаёт некорректные интервалы (<= 0).
	defaultMonitorIntervalSeconds = 30
)

//================================================================================
// ОСНОВНАЯ ЛОГИКА МОНИТОРИНГА
//================================================================================

// NewMonitor создает новый экземпляр монитора.
//
// @param cfg Конфигурация монитора.
// @param cfgManager Менеджер конфигурации.
// @param logger Логгер для вывода сообщений.
// @return Указатель на полностью готовый к работе экземпляр Monitor.
func NewMonitor(cfg *config.Config, cfgManager *config.Manager, logger *logger.Logger) *Monitor {
	return &Monitor{
		config:     *cfg,
		log:        logger,
		cfgManager: cfgManager,
		lastLevel:  -1,
		stopChan:   make(chan struct{}),
	}
}

// Start запускает основной цикл работы монитора с поддержкой обновления конфигурации.
// Этот метод является блокирующим и должен выполняться в главной горутине фонового процесса.
//
// @param mode Режим работы (например, "simulate").
// @param started Канал для сигнала о том, что монитор успешно запущен.
// @return Ничего.
func (m *Monitor) Start(mode string, started chan<- struct{}) {
	m.log.Info("Запуск основного цикла монитора.")

	// Определяем, какой источник данных использовать: реальный или симулятор.
	var getInfo func() (*battery.BatteryInfo, error)
	// if mode == "test" {
	// 	// TODO: Реализовать логику симулятора
	// 	m.log.Info("Режим симуляции пока не реализован. Используются реальные данные.")
	// 	getInfo = battery.GetBatteryInfo
	// } else {
	m.log.Info("Режим работы: РЕАЛЬНЫЕ ДАННЫЕ.")
	getInfo = battery.GetBatteryInfo
	// }

	// Немедленно получаем данные, чтобы не ждать первого тика.
	initialInfo, err := getInfo()
	if err != nil {
		m.log.Error(fmt.Sprintf("Не удалось получить первичную информацию о батарее: %v", err))
	}

	if initialInfo != nil {
		m.Check(time.Now(), *initialInfo)
	}

	initialInterval := m.safeInterval(m.intervalForInfo(initialInfo))
	ticker := time.NewTicker(initialInterval)
	m.log.Info(fmt.Sprintf("Мониторинг запущен. Текущий интервал проверки: %s", initialInterval))

	// Сигнализируем, что монитор запущен.
	if started != nil {
		close(started)
	}

	for { // Запускаем основной безконечный цикл
		select {
		// В случае получения сигнала от таймера
		case now := <-ticker.C:
			// Получаем информацию о батарее
			info, err := getInfo()
			if err != nil {
				m.log.Error(fmt.Sprintf("Ошибка получения информации о батарее: %v", err))
				continue
			}
			// Выполняем проверку состояния батареи и соблюдения порогов
			m.Check(now, *info)
			// После проверки обновляем интервал тикера, т.к. режим заряда мог измениться.
			nextInterval := m.safeInterval(m.intervalForInfo(info))
			ticker.Reset(nextInterval)
			m.log.Line()
			m.log.Info(fmt.Sprintf("Текущий интервал проверки: %s", nextInterval))
			m.log.Info(fmt.Sprintf("Текущий уровень заряда: %d%%", info.CurrentCapacity))
			m.log.Info(fmt.Sprintf("Текущее состояние зарядки: %v", info.IsCharging))
			m.log.Info(fmt.Sprintf("Текущее состояние подключения к сети: %v", info.IsPlugged))
			m.log.Info(fmt.Sprintf("Текущее время до полной зарядки: %d минут", info.TimeToFull))
			m.log.Info(fmt.Sprintf("Текущее время до полной разрядки: %d минут", info.TimeToEmpty))
			m.log.Line()
		case <-m.stopChan: // В случае получения сигнала остановки
			ticker.Stop()
			m.log.Info("Монитор остановлен.")
			return
		}
	}
}

// intervalForInfo подбирает интервал опроса на основании состояния батареи.
func (m *Monitor) intervalForInfo(info *battery.BatteryInfo) time.Duration {
	if info == nil {
		return time.Duration(m.config.CheckIntervalWhenDischarging) * time.Second
	}
	return time.Duration(m.getCheckInterval(*info)) * time.Second
}

// safeInterval заменяет нулевые и отрицательные интервалы на значение по умолчанию.
func (m *Monitor) safeInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		m.log.Debug(fmt.Sprintf("Интервал %s некорректен, используется значение по умолчанию %dс", interval, defaultMonitorIntervalSeconds))
		return time.Duration(defaultMonitorIntervalSeconds) * time.Second
	}
	return interval
}

// getCheckInterval определяет текущий интервал проверки на основе состояния зарядки.
//
// @return Интервал проверки в зависимости от состояния зарядки.
func (m *Monitor) getCheckInterval(info battery.BatteryInfo) int {
	// Если зарядка включена, возвращаем интервал проверки при зарядке.
	if info.IsCharging {
		return m.config.CheckIntervalWhenCharging
	}
	// Иначе возвращаем интервал проверки при разрядке.
	return m.config.CheckIntervalWhenDischarging
}

// Check выполняет разовую проверку состояния батареи.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) Check(now time.Time, info battery.BatteryInfo) {

	// Логируем, если состояние батареи не изменилось, но продолжаем проверку пороговых значений
	if m.isInitialized && info.CurrentCapacity == m.lastLevel && info.IsCharging == m.lastKnownCharging {
		m.log.Debug("Состояние батареи не изменилось, но продолжаем проверку пороговых значений.")
		// НЕ ВОЗВРАЩАЕМСЯ! Продолжаем проверку пороговых значений
	}

	// Информируем о текущем состоянии батареи.
	m.log.Debug(fmt.Sprintf(
		"Проверка состояния: Зарядка=%v, Уровень=%d%%",
		info.IsCharging, info.CurrentCapacity,
	))

	// КРИТИЧНО: Проверяем пороговые значения ДО обновления lastLevel!
	// Это позволяет корректно обрабатывать случай m.lastLevel == -1
	if info.IsCharging {
		// Если зарядка включена, проверяем состояние заряда.
		m.checkChargingState(now, info)
	} else {
		// Если зарядка выключена, проверяем состояние разряда.
		m.checkDischargingState(now, info)
	}

	// Логика управления яркостью экрана (только если включена в настройках)
	if m.config.BrightnessControlEnabled {
		// Сохраняем яркость при подключении зарядки
		if !m.lastKnownCharging && info.IsCharging {
			brightness, err := getCurrentBrightness()
			if err == nil {
				m.lastBrightness = brightness
				m.log.Info(fmt.Sprintf("Яркость сохранена при подключении зарядки: %d%%", brightness))
			} else {
				m.log.Error(fmt.Sprintf("Ошибка получения яркости при подключении зарядки: %v", err))
			}
		}

		// Восстанавливаем яркость при отключении зарядки
		if m.lastKnownCharging && !info.IsCharging && m.lastBrightness > 0 {
			err := setBrightness(m.lastBrightness)
			if err == nil {
				m.log.Info(fmt.Sprintf("Яркость восстановлена при отключении зарядки: %d%%", m.lastBrightness))
			} else {
				m.log.Error(fmt.Sprintf("Ошибка установки яркости при отключении зарядки: %v", err))
			}
		}
	}

	// ТЕПЕРЬ запоминаем текущий уровень заряда ПОСЛЕ проверки порогов
	m.lastLevel = info.CurrentCapacity

	// Если это первая инициализация
	if !m.isInitialized {
		m.isInitialized = true                // Устанавливаем флаг инициализации.
		m.lastKnownCharging = info.IsCharging // Запоминаем текущее состояние зарядки.
	} else if m.lastKnownCharging != info.IsCharging {
		// Если режим зарядки изменился
		m.log.Check("Обнаружена смена режима заряда. Состояние сброшено.\n")
		m.resetState(info.IsCharging) // Сбрасываем состояние при смене режима заряда.
	}
	m.log.Info(fmt.Sprintf("Текущий интервал проверки: %d секунд", m.getCheckInterval(info)))
}

// resetState сбрасывает внутреннее состояние мониторинга при смене режима заряда.
//
// @param newChargingState Новое состояние зарядки.
func (m *Monitor) resetState(newChargingState bool) {
	m.lastKnownCharging = newChargingState
	m.lastLevel = -1
	if !newChargingState {
		m.lastBrightness = 0 // Обнуляем яркость после восстановления при отключении зарядки
	}
}

// checkDischargingState проверяет, нужно ли отправлять уведомление при разрядке.
// Отправляет уведомление только при первом достижении минимального порога.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) checkDischargingState(now time.Time, info battery.BatteryInfo) {

	// Отладочное сообщение для проверки состояния разрядки
	m.log.Debug(fmt.Sprintf(
		"Проверка разрядки: Текущий заряд=%d%%, LastLevel=%d, IsCharging=%t, MinThreshold=%d%%",
		info.CurrentCapacity, m.lastLevel, info.IsCharging, m.config.MinThreshold,
	))

	// Если батарея заряжается, проверка разрядки не нужна
	if info.IsCharging {
		m.log.Debug("Батарея заряжается, проверка разрядки пропущена")
		return
	}

	level := info.CurrentCapacity

	// Определяем направление изменения уровня заряда
	var currentDirection int
	if m.lastLevel != -1 {
		if level > m.lastLevel {
			currentDirection = 1 // Рост уровня
		} else if level < m.lastLevel {
			currentDirection = -1 // Падение уровня
		} else {
			currentDirection = 0 // Нет изменений
		}
	}

	// Уведомление отправляется только при первом пересечении порога с учетом направления изменения
	if m.lastLevel != -1 {
		crossedThreshold := m.lastLevel > m.config.MinThreshold && level <= m.config.MinThreshold
		if crossedThreshold && currentDirection == -1 {
			// Критический уровень достигнут впервые при разрядке
			message := fmt.Sprintf(
				"⚠️ КРИТИЧЕСКИЙ УРОВЕНЬ ЗАРЯДА: %d%%\n\nПожалуйста, срочно подключите зарядное устройство!",
				level,
			)
			m.log.Info(fmt.Sprintf("🔋 ОТПРАВКА УВЕДОМЛЕНИЯ О РАЗРЯДКЕ: %d%% (порог: %d%%)", level, m.config.MinThreshold))
			m.log.Check(message)
			if err := dialog.ShowLowBatteryNotification(message, m.log); err != nil {
				m.log.Error(fmt.Sprintf("Ошибка отправки уведомления о разрядке: %v", err))
			} else {
				m.log.Info(fmt.Sprintf("✅ Уведомление о разрядке (%d%%) успешно отправлено", level))
			}
		} else {
			m.log.Debug(fmt.Sprintf("Порог не пересечен или направление не падение: lastLevel=%d, level=%d, threshold=%d, direction=%d", m.lastLevel, level, m.config.MinThreshold, currentDirection))
		}
		m.lastDirection = currentDirection
		return
	}

	// Сценарий первого запуска: если заряд уже на пороге или ниже
	if m.lastLevel == -1 && level <= m.config.MinThreshold {
		// Первый запуск и заряд уже критический
		message := fmt.Sprintf(
			"⚠️ КРИТИЧЕСКИЙ УРОВЕНЬ ЗАРЯДА: %d%%\n\nПожалуйста, срочно подключите зарядное устройство!",
			level,
		)

		m.log.Info(fmt.Sprintf("🔋 ОТПРАВКА УВЕДОМЛЕНИЯ О КРИТИЧЕСКОМ ЗАРЯДЕ (первый запуск): %d%%", level))
		m.log.Check(message)

		// Отображаем уведомление
		if err := dialog.ShowLowBatteryNotification(message, m.log); err != nil {
			m.log.Error(fmt.Sprintf("Ошибка отправки уведомления о критическом заряде: %v", err))
		} else {
			m.log.Info("✅ Уведомление о критическом заряде успешно отправлено")
		}
	} else {
		m.log.Debug(fmt.Sprintf("Условия для уведомления о разрядке не выполнены: level=%d, last=%d", level, m.lastLevel))
	}
	// Обновляем направление для следующей итерации
	m.lastDirection = currentDirection
}

// checkChargingState проверяет, нужно ли отправлять уведомление при зарядке.
// Отправляет уведомление только при первом достижении максимального порога.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) checkChargingState(now time.Time, info battery.BatteryInfo) {
	level := info.CurrentCapacity
	// Отладочное сообщение для проверки состояния зарядки
	m.log.Debug(fmt.Sprintf(
		"Проверка зарядки: Текущий заряд=%d%%, LastLevel=%d, IsCharging=%t, MaxThreshold=%d%%",
		level, m.lastLevel, info.IsCharging, m.config.MaxThreshold,
	))

	// Если батарея не заряжается, проверка зарядки не нужна
	if !info.IsCharging {
		m.log.Debug("Батарея не заряжается, проверка зарядки пропущена")
		return
	}

	// Определяем направление изменения уровня заряда
	var currentDirection int
	if m.lastLevel != -1 {
		if level > m.lastLevel {
			currentDirection = 1 // Рост уровня
		} else if level < m.lastLevel {
			currentDirection = -1 // Падение уровня
		} else {
			currentDirection = 0 // Нет изменений
		}
	}

	// Проверяем, пересекли ли мы порог max_threshold при зарядке с учетом направления изменения
	// Уведомление отправляется только при первом пересечении порога
	if m.lastLevel != -1 {
		crossedThreshold := m.lastLevel < m.config.MaxThreshold && level >= m.config.MaxThreshold
		if crossedThreshold && currentDirection == 1 {
			// Максимальный уровень достигнут впервые при зарядке
			message := fmt.Sprintf(
				"⚡ МАКСИМАЛЬНЫЙ УРОВЕНЬ ЗАРЯДА: %d%%\n\nРекомендуется отключить зарядное устройство для продления срока службы батареи.",
				level,
			)
			m.log.Info(fmt.Sprintf("🔌 ОТПРАВКА УВЕДОМЛЕНИЯ О ЗАРЯДКЕ: %d%% (порог: %d%%)", level, m.config.MaxThreshold))
			m.log.Check(message)
			if err := dialog.ShowHighBatteryNotification(message, m.log); err != nil {
				m.log.Error(fmt.Sprintf("Ошибка отправки уведомления о зарядке: %v", err))
			} else {
				m.log.Info(fmt.Sprintf("✅ Уведомление о зарядке (%d%%) успешно отправлено", level))
			}
		} else {
			m.log.Debug(fmt.Sprintf("Порог не пересечен или направление не рост: lastLevel=%d, level=%d, threshold=%d, direction=%d", m.lastLevel, level, m.config.MaxThreshold, currentDirection))
		}
		m.lastDirection = currentDirection
		return
	}

	// Сценарий первого запуска: если заряд уже на пороге или выше
	if m.lastLevel == -1 && level >= m.config.MaxThreshold {
		// Первый запуск и заряд уже максимальный
		message := fmt.Sprintf(
			"⚡ МАКСИМАЛЬНЫЙ УРОВЕНЬ ЗАРЯДА: %d%%\n\nРекомендуется отключить зарядное устройство для продления срока службы батареи.",
			level,
		)

		m.log.Info(fmt.Sprintf("🔌 ОТПРАВКА УВЕДОМЛЕНИЯ О МАКСИМАЛЬНОМ ЗАРЯДЕ (первый запуск): %d%%", level))
		m.log.Check(message)

		// Отображаем уведомление
		if err := dialog.ShowHighBatteryNotification(message, m.log); err != nil {
			m.log.Error(fmt.Sprintf("Ошибка отправки уведомления о максимальном заряде: %v", err))
		} else {
			m.log.Info("✅ Уведомление о максимальном заряде успешно отправлено")
		}
	} else {
		m.log.Debug(fmt.Sprintf("Условия для уведомления о зарядке не выполнены: level=%d, last=%d", level, m.lastLevel))
	}
	// Обновляем направление для следующей итерации
	m.lastDirection = currentDirection
}

// Stop останавливает работу монитора.
//
// @return Ничего.
func (m *Monitor) Stop() {
	m.log.Info("Остановка монитора...")
	close(m.stopChan)
}

// getCurrentBrightness получает текущую яркость экрана (0-100)
//
// @return Текущая яркость экрана (0-100).
func getCurrentBrightness() (int, error) {
	cmd := exec.Command("osascript", "-e", "tell application \"System Events\" to get brightness of (brightness of (display 1))")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	brightness, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, err
	}
	return brightness, nil
}

// setBrightness устанавливает яркость экрана (0-100)
//
// @param brightness Яркость экрана (0-100).
//
// @return Ничего.
func setBrightness(brightness int) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("яркость должна быть в диапазоне 0-100")
	}
	cmd := exec.Command("osascript", "-e", fmt.Sprintf("tell application \"System Events\" to set brightness of (brightness of (display 1)) to %d", brightness))
	return cmd.Run()
}
