// Package monitor содержит основную логику фонового процесса:
// мониторинг батареи и отслеживание изменений в файле конфигурации.

/**
 * @file monitor.go
 * @brief Модуль для мониторинга состояния батареи ноутбука с уведомлениями на каждый процент изменения заряда.
 *
 * Этот модуль отслеживает уровень заряда и состояние подключения к сети.
 * Он отправляет уведомления при каждом изменении уровня заряда на 1%:
 * - При зарядке: уведомление на каждый +1% заряда до отключения зарядки
 * - При разрядке: уведомление на каждый -1% заряда до подключения зарядки
 *
 * Модуль является гибко настраиваемым и легко тестируемым.
 *
 * @author Zeleza
 * @date 2025-08-07
 * @version 2.1.18
 *
 * @details
 * Основные принципы работы:
 * 1. Модуль использует только стандартные библиотеки Go, избегая системных вызовов.
 * 2. Проверяет состояние батареи ноутбука в непрерывном цикле.
 * 3. Если текущий уровень заряда не изменился относительно lastLevel, проверка пропускается.
 * 4. При разрядке: отправляет уведомление на каждое снижение заряда на 1% до подключения зарядки.
 * 5. При зарядке: отправляет уведомление на каждое увеличение заряда на 1% до отключения зарядки.
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
	stopChan          chan struct{}
}

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

	// Получаем начальный интервал проверки на основе состояния зарядки
	// Если зарядка включена, то начальный интервал равен значению переменной CheckIntervalWhenCharging,
	// а если зарядка выключена, то начальный интервал равен значению переменной CheckIntervalWhenDischarging.
	initialInterval := time.Duration(m.getCheckInterval(battery.BatteryInfo{})) * time.Second
	ticker := time.NewTicker(initialInterval) // Создаем тикер с начальным интервалом проверки
	m.log.Info(fmt.Sprintf("Мониторинг запущен. Текущий интервал проверки: %d секунд", m.getCheckInterval(battery.BatteryInfo{})))

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
			ticker.Reset(time.Duration(m.getCheckInterval(*info)) * time.Second)
			m.log.Line()
			m.log.Info(fmt.Sprintf("Текущий интервал проверки: %d секунд", m.getCheckInterval(*info)))
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
}

// checkDischargingState проверяет, нужно ли отправлять уведомление при разрядке.
// Отправляет уведомление на каждое снижение заряда на 1% до подключения зарядки.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) checkDischargingState(now time.Time, info battery.BatteryInfo) {

	// Отладочное сообщение для проверки состояния разрядки
	m.log.Debug(fmt.Sprintf(
		"Проверка разрядки: Текущий заряд=%d%%, LastLevel=%d, IsCharging=%t",
		info.CurrentCapacity, m.lastLevel, info.IsCharging,
	))

	// Если батарея заряжается, проверка разрядки не нужна
	if info.IsCharging {
		m.log.Debug("Батарея заряжается, проверка разрядки пропущена")
		return
	}

	level := info.CurrentCapacity
	// Проверяем, снизился ли заряд на 1% или более при разрядке (зарядка НЕ подключена)
	if m.lastLevel != -1 && level < m.lastLevel {

		// Отправляем уведомление на каждый процент снижения заряда до достижения минимального порога
		// for level := m.lastLevel - 1; level >= info.CurrentCapacity; level-- {
		// Отправляем уведомление только если заряд выше минимального порога
		// или если достигли минимального порога (критическое уведомление)
		if level <= m.config.MinThreshold {
			// Формируем сообщение в зависимости от уровня заряда
			message := fmt.Sprintf(
				"⚠️ КРИТИЧЕСКИЙ УРОВЕНЬ ЗАРЯДА: %d%%\n\nПожалуйста, срочно подключите зарядное устройство!",
				level,
			)

			m.log.Info(fmt.Sprintf("🔋 ОТПРАВКА УВЕДОМЛЕНИЯ О РАЗРЯДКЕ: %d%%", level))
			m.log.Check(message)

			// Отображаем уведомление
			if err := dialog.ShowLowBatteryNotification(message, m.log); err != nil {
				m.log.Error(fmt.Sprintf("Ошибка отправки уведомления о разрядке: %v", err))
			} else {
				m.log.Info(fmt.Sprintf("✅ Уведомление о разрядке (%d%%) успешно отправлено", level))
			}
		} else {
			// Если заряд ниже минимального порога, прекращаем отправку уведомлений
			m.log.Debug(fmt.Sprintf("Заряд %d%% ниже минимального порога %d%%, уведомления прекращены", level, m.config.MinThreshold))
		}
	} else if m.lastLevel == -1 && level <= m.config.MinThreshold {
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
}

// checkChargingState проверяет, нужно ли отправлять уведомление при зарядке.
// Отправляет уведомление на каждое увеличение заряда на 1% до отключения зарядки.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) checkChargingState(now time.Time, info battery.BatteryInfo) {
	level := info.CurrentCapacity
	// Отладочное сообщение для проверки состояния зарядки
	m.log.Debug(fmt.Sprintf(
		"Проверка зарядки: Текущий заряд=%d%%, LastLevel=%d, IsCharging=%t",
		level, m.lastLevel, info.IsCharging,
	))

	// Если батарея не заряжается, проверка зарядки не нужна
	if !info.IsCharging {
		m.log.Debug("Батарея не заряжается, проверка зарядки пропущена")
		return
	}

	// Проверяем, увеличился ли заряд на 1% или более при зарядке (зарядка подключена)
	if m.lastLevel != -1 && level > m.lastLevel {

		// Отправляем уведомление на каждый процент увеличения заряда до достижения максимального порога
		// for level := m.lastLevel + 1; level <= info.CurrentCapacity; level++ {
		// Отправляем уведомление только если заряд ниже максимального порога
		// или если достигли максимального порога (критическое уведомление)
		if level >= m.config.MaxThreshold {
			// Формируем сообщение в зависимости от уровня заряда
			message := fmt.Sprintf(
				"⚡ МАКСИМАЛЬНЫЙ УРОВЕНЬ ЗАРЯДА: %d%%\n\nРекомендуется отключить зарядное устройство для продления срока службы батареи.",
				level,
			)

			m.log.Info(fmt.Sprintf("🔌 ОТПРАВКА УВЕДОМЛЕНИЯ О ЗАРЯДКЕ: %d%%", level))
			m.log.Check(message)

			// Отображаем уведомление
			if err := dialog.ShowHighBatteryNotification(message, m.log); err != nil {
				m.log.Error(fmt.Sprintf("Ошибка отправки уведомления о зарядке: %v", err))
			} else {
				m.log.Info(fmt.Sprintf("✅ Уведомление о зарядке (%d%%) успешно отправлено", level))
			}
		} else {
			m.log.Debug(fmt.Sprintf("Заряд %d%% ниже максимального порога %d%%, уведомления прекращены", level, m.config.MaxThreshold))
			// break
		}
	} else if m.lastLevel == -1 && level >= m.config.MaxThreshold {
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
}

// Stop останавливает работу монитора.
//
// @return Ничего.
func (m *Monitor) Stop() {
	m.log.Info("Остановка монитора...")
	close(m.stopChan)
}
