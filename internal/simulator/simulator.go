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
 * @brief Имитирует поведение батареи для стресс-теста логики уведомлений.
 */
type BatterySimulator struct {
	notifier         *logger.Logger
	info             battery.BatteryInfo
	minThreshold     int
	maxThreshold     int
	maxNotifications int // Сколько уведомлений нужно дождаться
	state            simulatorState
}

// ИЗМЕНЕНИЕ: Конструктор теперь принимает maxNotifications и инициализирует состояние.
func NewBatterySimulator(logger *logger.Logger,
	startCurrentCapacity int,
	startCharging bool,
	minThreshold, maxThreshold int,
	maxNotifications int) *BatterySimulator {

	s := &BatterySimulator{
		notifier: logger,
		info: battery.BatteryInfo{
			CurrentCapacity: startCurrentCapacity,
			IsCharging:      startCharging,
		},
		minThreshold:     minThreshold,
		maxThreshold:     maxThreshold,
		maxNotifications: maxNotifications,
	}
	if startCharging {
		s.state = StateRampingUp
	} else {
		s.state = StateRampingDown
	}
	return s
}

func (s *BatterySimulator) GetNextState(monitorNotificationsShown int) (*battery.BatteryInfo, error) {

	switch s.state {
	case StateRampingDown:
		s.notifier.Test(fmt.Sprintf("Фаза плавной разрядки... %d", s.info.CurrentCapacity))
		s.info.CurrentCapacity -= 1 // ИЗМЕНЕНИЕ: Уменьшаем заряд на 1%
		if s.info.CurrentCapacity <= s.minThreshold {
			s.notifier.Test("Достигнут минимальный порог.")
			s.state = StateTriggeringMin
		}

	case StateTriggeringMin:
		s.notifier.Test(fmt.Sprintf("Ожидание (Разрядка). Показано монитором: %d. ", monitorNotificationsShown))
		if monitorNotificationsShown >= s.maxNotifications {
			s.notifier.Test("Монитор показал все уведомления! Переключаю на зарядку.")
			s.info.IsCharging = true
			s.state = StateRampingUp
			s.info.CurrentCapacity = s.maxThreshold - 1
			return &s.info, nil
		}
		// ИЗМЕНЕНИЕ: Создаем пульсацию заряда в триггерной зоне
		if s.info.CurrentCapacity <= s.minThreshold-1 {
			s.info.CurrentCapacity = s.minThreshold
		} else {
			s.info.CurrentCapacity = s.minThreshold - 1
		}
		s.notifier.Test(fmt.Sprintf("Пульсация заряда -> %d%%\n", s.info.CurrentCapacity))

	case StateRampingUp:
		s.notifier.Test(fmt.Sprintf("Фаза плавной зарядки... %d", s.info.CurrentCapacity))
		s.info.CurrentCapacity += 1 // ИЗМЕНЕНИЕ: Увеличиваем заряд на 1%
		if s.info.CurrentCapacity >= s.maxThreshold {
			s.notifier.Test("Достигнут максимальный порог.")
			s.state = StateTriggeringMax
		}

	case StateTriggeringMax:
		s.notifier.Test(fmt.Sprintf("Ожидание (Зарядка). Показано монитором: %d. ", monitorNotificationsShown))
		if monitorNotificationsShown >= s.maxNotifications {
			s.notifier.Test("Монитор показал все уведомления! Переключаю на разрядку.")
			s.info.IsCharging = false
			s.state = StateRampingDown
			s.info.CurrentCapacity = s.minThreshold + 1
			return &s.info, nil
		}
		// ИЗМЕНЕНИЕ: Создаем пульсацию заряда в триггерной зоне
		if s.info.CurrentCapacity >= s.maxThreshold+1 {
			s.info.CurrentCapacity = s.maxThreshold
		} else {
			s.info.CurrentCapacity = s.maxThreshold + 1
		}
		s.notifier.Test(fmt.Sprintf("Пульсация заряда -> %d%%\n", s.info.CurrentCapacity))
	}

	// Удерживаем заряд в пределах 0-100
	if s.info.CurrentCapacity > 100 {
		s.info.CurrentCapacity = 100
	} else if s.info.CurrentCapacity < 0 {
		s.info.CurrentCapacity = 0
	}

	return &s.info, nil
}
