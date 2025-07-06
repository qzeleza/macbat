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

package monitor1

import (
	"fmt"
	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/simulator"
	"time"
)

//================================================================================
// СТРУКТУРЫ ДАННЫХ
//================================================================================

/**
 * @struct Monitor
 * @brief Основной объект, который управляет состоянием и логикой мониторинга.
 */
type Monitor struct {
	config                 config.Config  // Конфигурация монитора.
	notifier               *logger.Logger // Объект для отправки уведомлений.
	lastNotificationTime   time.Time      // Временная метка последнего уведомления.
	notificationsRemaining int            // Счетчик показанных уведомлений в текущем цикле.
	lastKnownCharging      bool           // Последнее известное состояние (заряжается/не заряжается).
	isInitialized          bool           // Флаг, показывающий, был ли монитор запущен хотя бы раз.
	lastLevel              int            // Последний известный уровень заряда для оптимизации.
}

//================================================================================
// ОСНОВНАЯ ЛОГИКА МОНИТОРИНГА
//================================================================================

// ИЗМЕНЕНИЕ: Определяем абстрактный тип "поставщика" данных о батарее.
type batteryInfoProvider func() (*battery.BatteryInfo, error)

/**
 * @brief Создает и инициализирует новый экземпляр монитора батареи.
 * @param config Конфигурация с порогами и интервалами.
 * @param logger Реализация интерфейса Logger для отправки уведомлений.
 * @return Указатель на полностью готовый к работе экземпляр Monitor.
 */
func NewMonitor(config *config.Config, logger *logger.Logger) *Monitor {
	return &Monitor{
		config:   *config,
		notifier: logger,
		// Инициализируем lastLevel значением, которое точно не совпадет
		// с реальным уровнем заряда при первой проверке.
		lastLevel: -1,
	}
}

/**
 * @brief ИЗМЕНЕНИЕ: Запускает бесконечный цикл мониторинга батареи.
 *
 * Этот метод является главной точкой входа для запуска службы. Он будет
 * работать до тех пор, пока программа не будет принудительно завершена.
 * Внутри цикла он получает состояние батареи, выполняет проверку и затем
 * ожидает в течение интервала, который зависит от текущего режима заряда.
 */
func (m *Monitor) Start() {
	var provider batteryInfoProvider

	m.notifier.Info("Запуск монитора батареи...")

	if m.config.UseSimulator {
		m.notifier.Test("Режим работы: СИМУЛЯТОР.")
		simulator := simulator.NewBatterySimulator(
			m.notifier, 23, false,
			m.config.MinThreshold,
			m.config.MaxThreshold,
			m.config.MaxNotifications,
		)
		provider = func() (*battery.BatteryInfo, error) {
			return simulator.GetNextState(m.notificationsRemaining)
		}
	} else {
		m.notifier.Info("Режим работы: РЕАЛЬНЫЕ ДАННЫЕ.")
		provider = battery.GetBatteryInfo
	}

	m.notifier.Info(fmt.Sprintf(
		"Мониторинг запущен. Интервалы: зарядка - %v, разрядка - %v.",
		m.config.CheckIntervalWhenCharging,
		m.config.CheckIntervalWhenDischarging,
	))

	for {
		currentInfo, err := provider()
		if err != nil {
			m.notifier.Error(fmt.Sprintf("Ошибка получения данных о батарее: %v. Следующая попытка через 30 секунд.", err))
			time.Sleep(30 * time.Second)
			continue
		}

		m.Check(time.Now(), *currentInfo)

		var sleepDuration time.Duration
		if currentInfo.IsCharging {
			sleepDuration = m.config.CheckIntervalWhenCharging
		} else {
			sleepDuration = m.config.CheckIntervalWhenDischarging
		}
		m.notifier.Debug(fmt.Sprintf("Следующая проверка через %v.", sleepDuration))
		time.Sleep(sleepDuration)
	}
}

/**
 * @brief Основной метод, выполняющий проверку состояния батареи.
 *
 * Этот метод является точкой входа для всей логики. Он анализирует
 * предоставленную информацию о батарее и текущее время, решая, нужно ли
 * отправлять уведомление или сбрасывать состояние.
 * @param now Текущее время. Передается явно для упрощения тестирования.
 * @param info Актуальная информация о состоянии батареи.
 */
func (m *Monitor) Check(now time.Time, info battery.BatteryInfo) {
	// Оптимизация: если уровень заряда не изменился, дальнейшая проверка не нужна.
	if m.isInitialized && info.CurrentCapacity == m.lastLevel {
		m.notifier.Debug("Уровень заряда не изменился. Проверка пропущена.")
		return
	}
	// Обновляем последнее известное значение уровня.
	m.lastLevel = info.CurrentCapacity

	// При первом запуске инициализируем начальное состояние.
	if !m.isInitialized {
		m.isInitialized = true
		m.lastKnownCharging = info.IsCharging
	} else if m.lastKnownCharging != info.IsCharging {
		// Если режим заряда изменился, сбрасываем счетчики.
		m.notifier.Check("Обнаружена смена режима заряда. Состояние сброшено.\n")
		m.resetState(info.IsCharging)
	}

	// Вызов соответствующей логики в зависимости от режима.
	if info.IsCharging {
		m.checkChargingState(now, info)
	} else {
		m.checkDischargingState(now, info)
	}
}

/**
 * @brief Сбрасывает внутреннее состояние монитора.
 *
 * Вызывается при смене режима заряда (с зарядки на разрядку или наоборот).
 * Обнуляет счетчики уведомлений и временные метки.
 * @param newChargingState Новое состояние зарядки (`true` или `false`).
 */
func (m *Monitor) resetState(newChargingState bool) {
	m.lastNotificationTime = time.Time{} // Сброс на "нулевое" время.
	m.notificationsRemaining = 0
	m.lastKnownCharging = newChargingState
	m.lastLevel = -1.0 // Сбрасываем уровень, чтобы следующая проверка точно сработала.
}

/**
 * @brief Проверяет, нужно ли отправлять уведомление при разрядке батареи.
 * @param now Текущее время.
 * @param info Актуальная информация о батарее.
 * @private
 */
func (m *Monitor) checkDischargingState(now time.Time, info battery.BatteryInfo) {
	// Условие для срабатывания: уровень заряда ниже порога.
	if info.CurrentCapacity > m.config.MinThreshold {
		return
	}

	// Дополнительные условия: не превышен лимит уведомлений и прошел ли интервал.
	if m.notificationsRemaining < m.config.MaxNotifications && now.Sub(m.lastNotificationTime) >= m.config.NotificationInterval {
		message := fmt.Sprintf(
			"Батарея разряжена до %d%%.\nПожалуйста, подключите зарядку.\nОсталось %d увед.",
			info.CurrentCapacity,
			m.config.MaxNotifications-m.notificationsRemaining,
		)
		// Отправляем уведомление в консоль
		m.notifier.Check(message)
		// Отправляем системное уведомление в macOS
		if err := m.notifier.ShowLowBatteryNotification(message); err != nil {
			m.notifier.Error(err.Error())
		}
		m.lastNotificationTime = now
		m.notificationsRemaining++
	}
}

/**
 * @brief Проверяет, нужно ли отправлять уведомление при зарядке батареи.
 * @param now Текущее время.
 * @param info Актуальная информация о батарее.
 * @private
 */
func (m *Monitor) checkChargingState(now time.Time, info battery.BatteryInfo) {
	// Условие для срабатывания: уровень заряда выше порога.
	if info.CurrentCapacity < m.config.MaxThreshold {
		return
	}

	// Дополнительные условия: не превышен лимит уведомлений и прошел ли интервал.
	if m.notificationsRemaining < m.config.MaxNotifications && now.Sub(m.lastNotificationTime) >= m.config.NotificationInterval {
		message := fmt.Sprintf(
			"Батарея заряжена до %d%%.\nПожалуйста, отключите зарядку.\nОсталось %d увед.",
			info.CurrentCapacity,
			m.config.MaxNotifications-m.notificationsRemaining,
		)
		// Отправляем уведомление в консоль
		m.notifier.Check(message)
		// Отправляем системное уведомление в macOS
		if err := m.notifier.ShowHighBatteryNotification(message); err != nil {
			m.notifier.Error(err.Error())
		}
		m.lastNotificationTime = now
		m.notificationsRemaining++
	}
}
