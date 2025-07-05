// Package monitor содержит основную логику фонового процесса:
// мониторинг батареи и отслеживание изменений в файле конфигурации.

/**
 * @file monitor.go
 * @brief Модуль для мониторинга состояния батареи ноутбука.
 *
 * Этот модуль отслеживает уровень заряда и состояние подключения к сети.
 * Он отправляет уведомления, когда уровень заряда падает ниже минимального порога
 * при работе от батареи или поднимается выше максимального порога при зарядке.
 * Модуль является гибко настраиваемым и легко тестируемым.

 * @author Zeleza
 * @date 2025-06-21


 * @requestAI

	1. Напиши код модуля на Go стараясь использовать только библиотеки самого go,
	   постарайся не использовать системные команды вызывая их cmd.Exec или подобные системные вызовы.

	2. Модуль должен проверять состояние батареи ноутбука.

	3. Если текущий уровень заряда не изменился относительно переменной lastLevel, то сразу
	   выходим из проверки состояний триггеров.

	4. Если зарядка не подключена и текущий уровень заряда снизился установленного минимума
	   (задается в константе MinThreshold), то проверяем когда крайний раз отображалось
	   системное уведомление и если обо было больше, чем в константе NotificationInterval
	   и общее число показанных уведомлений было менее чем установлено в константе MaxNotifications,
	   то отображаем в этом случае системное уведомление для macos x.

	5. Если же, зарядка подключена и текущий уровень заряда повысился до установленного максимума
	   (задается в константе MaxThreshold), то проверяем когда крайний раз отображалось системное уведомление
	   и если обо было больше, чем в константе NotificationInterval и общее число показанных уведомлений
	   было менее чем установлено в константе MaxNotifications, то отображаем в этом случае системное уведомление.

	6. В случае, обнаружения смены режима заряда (с зарядки на разрядку или наоборот)
	   - все переменные сбрасываются.
	   Для проверки достижения состояний тригерров для показа системного сообщения
	   используй разные функции проверки состояний в случае разряда и заряда батареи.

	7. Для удобства тестирования предусмотри возможность в отдельной функции/модуля для автоматической симуляции
	   параметров батареи ноутбука. Сделай так, чтобы при симуляции режима, смена режима зарядки
	   не происходила до тех пор, пока не будет достигнуто одно из пороговых значений: MinThreshold или MaxThreshold.
	    при этом, код функции стимулятора должен быть таким, чтобы как можно чаще трестировались тригерные режимы
		- когда достигались крайние триггерные режимы maxThreshold и minThreshold.

	8. Сделай так, чтобы смена режима зарядки в симуляторе происходила, только после достижения текущего значения
		показанных уведомлений до значения в переменной MaxNotifications. И так, чтобы при симуляции текущий
		уровень зарядки изменялся бы вверх и вниз (в зависимости от типа текущего состояния зарядки) на 1%.

	9. Функцию показа системного уведомления - пока сделай в виде вывода сообщений
	   в файл лога /tmp/macbat.log. При этом, сделай так, чтобы логфайл создавался новый,
	   если размер файла превысит N записей

	10. Если можно улучшить код - сделай это и подробно прокомментируй весь код на русском,
	   также, в формате doxygen создай описание функций и самого модуля.

	11. Теперь, доработай код так, чтобы проверка заряда батареи запускалась в цикле по следующим правилам:
		- если текущий цикл зарядки, то цикл опроса равен значению переменной CheckIntervalWhenCharging,
		- а если происходит разрядка то цикл опроса равен значению переменной CheckIntervalWhenDischarging
*/

//================================================================================
// ПОДКЛЮЧЕНИЕ БИБЛИОТЕК
//================================================================================

package monitor

import (
	"fmt"
	"time"

	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/dialog"
	"macbat/internal/logger"
)

//================================================================================
// СТРУКТУРЫ ДАННЫХ
//================================================================================

// Monitor - это основная структура фонового процесса.
type Monitor struct {
	config               config.Config   // Конфигурация монитора.
	notifier             *logger.Logger  // Объект для отправки уведомлений.
	cfgManager           *config.Manager // Менеджер конфигурации.
	lastNotificationTime time.Time       // Временная метка последнего уведомления.
	notificationsShown   int             // Счетчик показанных уведомлений в текущем цикле.
	lastKnownCharging    bool            // Последнее известное состояние (заряжается/не заряжается).
	isInitialized        bool            // Флаг, показывающий, был ли монитор запущен хотя бы раз.
	lastLevel            int             // Последний известный уровень заряда для оптимизации.
	stopChan             chan struct{}
}

// batteryInfoProvider определяет абстрактный тип "поставщика" данных о батарее.
type batteryInfoProvider func() (*battery.BatteryInfo, error)

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
		notifier:   logger,
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
	m.notifier.Info("Запуск основного цикла монитора.")

	// Определяем, какой источник данных использовать: реальный или симулятор.
	var getInfo func() (*battery.BatteryInfo, error)
	if mode == "test" {
		// TODO: Реализовать логику симулятора
		m.notifier.Info("Режим симуляции пока не реализован. Используются реальные данные.")
		getInfo = battery.GetBatteryInfo
	} else {
		m.notifier.Info("Режим работы: РЕАЛЬНЫЕ ДАННЫЕ.")
		getInfo = battery.GetBatteryInfo
	}

	// Начальный интервал проверки.
	initialInterval := time.Duration(m.getCheckInterval()) * time.Second
	ticker := time.NewTicker(initialInterval)
	m.notifier.Info(fmt.Sprintf("Мониторинг запущен. Текущий интервал проверки: %d секунд", m.getCheckInterval()))

	// Сигнализируем, что монитор запущен.
	if started != nil {
		close(started)
	}

	for {
		select {
		case now := <-ticker.C:
			info, err := getInfo()
			if err != nil {
				m.notifier.Error(fmt.Sprintf("Ошибка получения информации о батарее: %v", err))
				continue
			}
			m.Check(now, *info)
			// После проверки обновляем интервал тикера, т.к. режим заряда мог измениться.
			ticker.Reset(time.Duration(m.getCheckInterval()) * time.Second)

		case <-m.stopChan:
			ticker.Stop()
			m.notifier.Info("Монитор остановлен.")
			return
		}
	}
}

// applyNewConfig безопасно применяет новую конфигурацию и перезапускает тикер.
//
// @param newCfg Новая конфигурация.
// @param ticker Тикер для перезапуска.
func (m *Monitor) applyNewConfig(newCfg *config.Config, ticker *time.Ticker) {
	m.notifier.Info("Получена новая конфигурация. Применение...")
	m.config = *newCfg // Обновляем конфигурацию.

	// Перезапускаем тикер с новым интервалом из новой конфигурации.
	newInterval := m.getCheckInterval()
	ticker.Reset(time.Duration(newInterval) * time.Second)
	m.notifier.Info("Новая конфигурация успешно применена.")

}

// getCheckInterval определяет текущий интервал проверки на основе состояния зарядки.
//
// @return Интервал проверки в зависимости от состояния зарядки.
func (m *Monitor) getCheckInterval() int {
	// Если зарядка включена, возвращаем интервал проверки при зарядке.
	if m.lastKnownCharging {
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

	// Если состояние батареи не изменилось, проверка пропускается.
	if m.isInitialized && info.CurrentCapacity == m.lastLevel && info.IsCharging == m.lastKnownCharging {
		m.notifier.Debug("Состояние батареи не изменилось. Проверка пропущена.")
		return
	}

	// Информируем о текущем состоянии батареи.
	m.notifier.Debug(fmt.Sprintf(
		"Проверка состояния: Зарядка=%v, Уровень=%d%%",
		info.IsCharging, info.CurrentCapacity,
	))

	// Запоминаем текущий уровень заряда.
	m.lastLevel = info.CurrentCapacity

	// Если это первая инициализация
	if !m.isInitialized {
		m.isInitialized = true                // Устанавливаем флаг инициализации.
		m.lastKnownCharging = info.IsCharging // Запоминаем текущее состояние зарядки.
	} else if m.lastKnownCharging != info.IsCharging {
		// Если режим зарядки изменился
		m.notifier.Check("Обнаружена смена режима заряда. Состояние сброшено.\n")
		m.resetState(info.IsCharging) // Сбрасываем состояние при смене режима заряда.
	}

	// Проверяем состояние заряда.
	if info.IsCharging {
		// Если зарядка включена, проверяем состояние заряда.
		m.checkChargingState(now, info)
	} else {
		// Если зарядка выключена, проверяем состояние разряда.
		m.checkDischargingState(now, info)
	}
}

// resetState сбрасывает внутреннее состояние мониторинга при смене режима заряда.
//
// @param newChargingState Новое состояние зарядки.
func (m *Monitor) resetState(newChargingState bool) {
	m.lastNotificationTime = time.Time{}
	m.notificationsShown = 0
	m.lastKnownCharging = newChargingState
	m.lastLevel = -1
}

// checkDischargingState проверяет, нужно ли отправлять уведомление при разрядке.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) checkDischargingState(now time.Time, info battery.BatteryInfo) {

	// Отладочное сообщение для проверки порогов.
	m.notifier.Debug(fmt.Sprintf(
		"Проверка нижнего порога: Текущий заряд=%d%%, Мин. порог=%d%%",
		info.CurrentCapacity, m.config.MinThreshold,
	))

	// Если уровень заряда выше порога, проверка пропускается.
	if info.CurrentCapacity > m.config.MinThreshold {
		return
	}

	// Если количество уведомлений не превышено и время с последнего уведомления прошло
	if m.notificationsShown < m.config.MaxNotifications && now.Sub(m.lastNotificationTime) >= time.Duration(m.config.NotificationInterval)*time.Second {
		remaining := m.config.MaxNotifications - m.notificationsShown - 1 // Оставшееся количество уведомлений
		// Формируем сообщение
		message := fmt.Sprintf(
			"Батарея разряжена до %d%%.\nПожалуйста, подключите зарядку.\nОсталось уведомлений: %d",
			info.CurrentCapacity,
			remaining,
		)
		// Отправляем уведомление
		m.notifier.Check(message)
		// Отображаем уведомление
		if err := dialog.ShowLowBatteryNotification(message, m.notifier); err != nil {
			m.notifier.Error(err.Error())
		}

		m.lastNotificationTime = now    // Обновляем время последнего уведомления
		m.notificationsShown++          // Увеличиваем счетчик уведомлений
		m.updateDischargeInterval(info) // Обновляем интервал проверки при разрядке в случае, если уровень заряда ниже порога.
	}
}

// checkChargingState проверяет, нужно ли отправлять уведомление при зарядке.
//
// @param now Текущее время.
// @param info Информация о батарее.
func (m *Monitor) checkChargingState(now time.Time, info battery.BatteryInfo) {

	// Отладочное сообщение для проверки порогов.
	m.notifier.Debug(fmt.Sprintf(
		"Проверка верхнего порога: Текущий заряд=%d%%, Макс. порог=%d%%",
		info.CurrentCapacity, m.config.MaxThreshold,
	))

	// Если уровень заряда ниже порога, проверка пропускается.
	if info.CurrentCapacity < m.config.MaxThreshold {
		return
	}

	// Если количество уведомлений не превышено и время с последнего уведомления прошло
	if m.notificationsShown < m.config.MaxNotifications && now.Sub(m.lastNotificationTime) >= time.Duration(m.config.NotificationInterval)*time.Second {
		// Определяем количество оставшихся уведомлений.
		remaining := m.config.MaxNotifications - m.notificationsShown - 1
		// Формируем сообщение.
		message := fmt.Sprintf(
			"Батарея заряжена до %d%%.\nМожете отключить зарядку.\nОсталось уведомлений: %d",
			info.CurrentCapacity,
			remaining,
		)
		m.notifier.Check(message) // Отправляем уведомление.
		if err := dialog.ShowHighBatteryNotification(message, m.notifier); err != nil {
			m.notifier.Error(err.Error())
		}

		m.lastNotificationTime = now // Обновляем время последнего уведомления.
		m.notificationsShown++       // Увеличиваем счетчик уведомлений.
		m.updateChargeInterval(info) // Обновляем интервал проверки при зарядке в случае, если достигнутый уровень заряда выше порога.
	}
}

// updateDischargeInterval обновляет интервал проверки при разрядке.
//
// @param info Информация о батарее.
func (m *Monitor) updateDischargeInterval(info battery.BatteryInfo) {
	gapCapacity := m.config.MinThreshold - info.CurrentCapacity                                          // Разница между минимальным порогом и текущим уровнем заряда.
	timeTick := m.config.CheckIntervalWhenDischarging / m.config.MinThreshold                            // Единица интервала проверки.
	m.config.CheckIntervalWhenDischarging = m.config.CheckIntervalWhenDischarging - timeTick*gapCapacity // Уменьшаем интервал проверки пропорционально разнице.
	m.cfgManager.Save(&m.config)                                                                         // Сохраняем конфигурацию в файле конфигурации.
}

// updateChargeInterval обновляет интервал проверки при зарядке.
//
// @param info Информация о батарее.
func (m *Monitor) updateChargeInterval(info battery.BatteryInfo) {
	gapCapacity := m.config.MaxThreshold - info.CurrentCapacity                                   // Разница между максимальным порогом и текущим уровнем заряда.
	timeBit := m.config.CheckIntervalWhenCharging / m.config.MaxThreshold                         // Единица интервала проверки.
	m.config.CheckIntervalWhenCharging = m.config.CheckIntervalWhenCharging - timeBit*gapCapacity // Уменьшаем интервал проверки пропорционально разнице.
	m.cfgManager.Save(&m.config)                                                                  // Сохраняем конфигурацию в файле конфигурации.
}

// Stop останавливает работу монитора.
//
// @return Ничего.
func (m *Monitor) Stop() {
	m.notifier.Info("Остановка монитора...")
	close(m.stopChan)
}
