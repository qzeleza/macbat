package simulator

import (
	"fmt"
	"macbat/internal/battery"
	"macbat/internal/logger"
)

//================================================================================
// СИМУЛЯТОР БАТАРЕИ (для демонстрации)
//================================================================================

// ИЗМЕНЕНИЕ: Добавляем перечисление для состояний симулятора.
type simulatorState int

const (
	StateRampingDown   simulatorState = iota // Состояние: быстро разряжаемся до порога
	StateTriggeringMin                       // Состояние: держим заряд ниже порога для вызова уведомлений
	StateRampingUp                           // Состояние: быстро заряжаемся до порога
	StateTriggeringMax                       // Состояние: держим заряд выше порога
)

/**
 * @struct BatterySimulator
 * @brief Имитирует поведение батареи для стресс-теста логики уведомлений
 *
 * @field notifier Указатель на логгер для записи сообщений
 * @field info Структура с информацией о текущем состоянии батареи
 * @field minThreshold Минимальный порог заряда для уведомлений
 * @field maxThreshold Максимальный порог заряда для уведомлений
 * @field maxNotifications Максимальное количество уведомлений, которые должны быть показаны
 * @field state Текущее состояние симулятора
 */
type BatterySimulator struct {
	notifier         *logger.Logger
	info             battery.BatteryInfo
	minThreshold     int
	maxThreshold     int
	maxNotifications int // Сколько уведомлений нужно дождаться
	state            simulatorState
}

// NewBatterySimulator создает новый экземпляр BatterySimulator.
//
// logger - логгер, который будет использоваться для вывода сообщений
// startCurrentCapacity - начальный заряд батареи
// startCharging - начальное состояние зарядки (true - заряжается, false - разряжается)
// minThreshold, maxThreshold - пороги, при достижении которых генерируется уведомление
// maxNotifications - максимальное количество уведомлений, которое будет сгенерировано
func NewBatterySimulator(logger *logger.Logger,
	startCurrentCapacity int,
	startCharging bool,
	minThreshold, maxThreshold int,
	maxNotifications int) *BatterySimulator {

	// Инициализация состояния симулятора
	s := &BatterySimulator{
		notifier: logger, // Указатель на логгер для записи сообщений
		info: battery.BatteryInfo{ // Структура с информацией о текущем состоянии батареи
			CurrentCapacity: startCurrentCapacity, // Начальный заряд батареи
			IsCharging:      startCharging,        // Начальное состояние зарядки (true - заряжается, false - разряжается)
		},
		minThreshold:     minThreshold,     // Минимальный порог заряда для уведомлений
		maxThreshold:     maxThreshold,     // Максимальный порог заряда для уведомлений
		maxNotifications: maxNotifications, // Максимальное количество уведомлений, которое будет сгенерировано
	}
	// Если зарядка включена, то переключаем состояние на плавную зарядку
	if startCharging {
		s.state = StateRampingUp // Переключаем состояние на плавную зарядку
	} else {
		s.state = StateRampingDown // Переключаем состояние на плавную разрядку
	}
	return s
}

/**
 * @brief Получает следующее состояние батареи в симуляторе.
 *
 * Эта функция моделирует изменение заряда батареи в зависимости от текущего
 * состояния симулятора и количества показанных уведомлений монитором. Она
 * управляет фазами разрядки и зарядки, изменяя заряд в пределах триггерных
 * зон и переключая состояния симулятора при достижении порогов.
 *
 * @param monitornotificationsRemaining Количество уведомлений, показанных монитором.
 * @return Указатель на структуру BatteryInfo с текущим состоянием батареи.
 * @return error Всегда возвращает nil, так как ошибки не обрабатываются.
 */
func (s *BatterySimulator) GetNextState(monitornotificationsRemaining int) (*battery.BatteryInfo, error) {

	const step = 2 // Шаг изменения заряда

	switch s.state {

	// Фаза плавной разрядки
	case StateRampingDown:
		s.notifier.Test(fmt.Sprintf("Фаза плавной разрядки... %d", s.info.CurrentCapacity))
		s.info.CurrentCapacity -= step // Уменьшаем заряд на step процентов
		// Если заряд меньше или равен минимальному порогу, то переключаем состояние на StateTriggeringMin
		if s.info.CurrentCapacity <= s.minThreshold {
			s.notifier.Test("Достигнут минимальный порог.")
			s.state = StateTriggeringMin
		}

	// Фаза достижения минимального порога разряда (дождемся пока не будет показано maxNotifications уведомлений)
	case StateTriggeringMin:
		s.notifier.Test(fmt.Sprintf("Ожидание (Разрядка). Показано монитором: %d. ", monitornotificationsRemaining))
		// Если количество показанных уведомлений больше или равно maxNotifications, то переключаем состояние на StateRampingUp
		if monitornotificationsRemaining >= s.maxNotifications {
			s.notifier.Test("Монитор показал все уведомления! Переключаю на зарядку.")
			s.info.IsCharging = true                       // Устанавливаем зарядку в true
			s.state = StateRampingUp                       // Переключаем состояние на StateRampingUp
			s.info.CurrentCapacity = s.maxThreshold - step // Устанавливаем заряд на maxThreshold - step
			return &s.info, nil                            // Возвращаем текущее состояние батареи
		}
		// Создаем пульсацию заряда в триггерной зоне, если текущий заряд меньше или равен minThreshold-step
		if s.info.CurrentCapacity <= s.minThreshold-step {
			s.info.CurrentCapacity = s.minThreshold // Устанавливаем заряд на minThreshold
		} else {
			s.info.CurrentCapacity = s.minThreshold - step // Устанавливаем заряд на minThreshold - step
		}
		s.notifier.Test(fmt.Sprintf("Пульсация заряда -> %d%%\n", s.info.CurrentCapacity))

	// Фаза плавной зарядки
	case StateRampingUp:
		s.notifier.Test(fmt.Sprintf("Фаза плавной зарядки... %d", s.info.CurrentCapacity))
		// Увеличиваем заряд на step процентов
		s.info.CurrentCapacity += step
		// Если заряд больше или равен максимальному порогу, то переключаем состояние на StateTriggeringMax
		if s.info.CurrentCapacity >= s.maxThreshold {
			s.notifier.Test("Достигнут максимальный порог.")
			s.state = StateTriggeringMax // Переключаем состояние на StateTriggeringMax
		}

	// Фаза до достижения максимального порога зарядки
	// (дождемся пока не будет показано maxNotifications уведомлений)
	case StateTriggeringMax:
		s.notifier.Test(fmt.Sprintf("Ожидание (Зарядка). Показано монитором: %d. ", monitornotificationsRemaining))
		// Если количество показанных уведомлений больше или равно maxNotifications, то переключаем состояние на StateRampingDown
		if monitornotificationsRemaining >= s.maxNotifications {
			s.notifier.Test("Монитор показал все уведомления! Переключаю на разрядку.")
			s.info.IsCharging = false                      // Устанавливаем зарядку в false
			s.state = StateRampingDown                     // Переключаем состояние на StateRampingDown
			s.info.CurrentCapacity = s.minThreshold + step // Устанавливаем заряд на minThreshold + step
			return &s.info, nil                            // Возвращаем текущее состояние батареи
		}
		// Создаем пульсацию заряда в триггерной зоне,
		// если текущий заряд больше или равен maxThreshold+step
		if s.info.CurrentCapacity >= s.maxThreshold+step {
			s.info.CurrentCapacity = s.maxThreshold // Устанавливаем заряд на maxThreshold
		} else {
			s.info.CurrentCapacity = s.maxThreshold + step // Устанавливаем заряд на maxThreshold + step
		}
		s.notifier.Test(fmt.Sprintf("Пульсация заряда -> %d%%\n", s.info.CurrentCapacity))
	}

	// Удерживаем заряд в пределах 0-100
	if s.info.CurrentCapacity > 100 {
		s.info.CurrentCapacity = 100 // Устанавливаем заряд на 100
	} else if s.info.CurrentCapacity < 0 {
		s.info.CurrentCapacity = 0 // Устанавливаем заряд на 0
	}

	return &s.info, nil
}
