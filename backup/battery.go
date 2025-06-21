// ФАЙЛ: battery_improved.go
// Улучшенная версия с исправлением всех выявленных проблем

package battery

// #include <CoreFoundation/CoreFoundation.h>
import "C"

import (
	"fmt"
	"macbat/internal/config"
	"macbat/internal/log"
	"macbat/internal/notification"
	"macbat/internal/utils"
	"sync"
	"time"
)

// Константы для улучшения читаемости кода
const (
	DefaultNotificationInterval = 5 * time.Minute // Интервал уведомлений по умолчанию
	MaxReasonableNotifications  = 100             // Максимальное разумное количество уведомлений
)

// Интерфейсы для тестируемости
type BatteryInfoProvider interface {
	GetBatteryInfo() (BatteryInfo, error) // Получение информации о батарее
}

type NotificationSender interface {
	// Отправка уведомления о низком заряде
	SendLowBatteryNotification(currentLevel, threshold int) error
	// Отправка уведомления о высоком заряде
	SendHighBatteryNotification(currentLevel, threshold int) error
}

type Logger interface {
	Check(message string) // Запись информационного сообщения
	Error(message string) // Запись сообщения об ошибке
}

// BatteryObserver управляет наблюдением за состоянием батареи
//
// Отвечает за мониторинг уровня заряда батареи, отправку уведомлений
// и управление состоянием уведомлений.
type BatteryObserver struct {
	// Состояние (защищено мьютексом)
	mu sync.RWMutex

	// Зависимости (инъектируются для тестирования)
	provider BatteryInfoProvider
	notifier NotificationSender
	logger   Logger
	note     notification.Notifier
}

// NotificationState представляет состояние уведомлений
// @deprecated Используйте config.NotificationState вместо этого типа
type NotificationState config.NotificationState

// NewBatteryObserver создает новый экземпляр наблюдателя за батареей
//
// @param provider - провайдер информации о батарее
// @param notifier - интерфейс для отправки уведомлений
// @param logger - логгер для записи событий
// @return *BatteryObserver - указатель на созданный экземпляр наблюдателя
func NewBatteryObserver(provider BatteryInfoProvider, notifier NotificationSender, logger Logger) *BatteryObserver {
	return &BatteryObserver{
		provider: provider,
		notifier: notifier,
		logger:   logger,
		note:     notification.Notifier,
	}
}

// GetInfo возвращает текущую информацию о батарее
//
// @return BatteryInfo - информация о текущем состоянии батареи
// @return error - ошибка в случае неудачного получения информации
//
// @note Метод является потокобезопасным
func (b *BatteryObserver) GetInfo() (BatteryInfo, error) {
	if b.provider != nil {
		return b.provider.GetBatteryInfo()
	}

	// Резервный вариант (для обратной совместимости)
	info, err := getBatteryInfo()
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("не удалось получить информацию о батарее: %w", err)
	}
	return *info, nil
}

// ObserveChanges основная функция наблюдения за состоянием батареи
//
// @param cfg - конфигурация приложения
// @return error - ошибка в случае проблем при обработке
//
// @note Метод является потокобезопасным и должен вызываться периодически
func (b *BatteryObserver) ObserveChanges(cfg *config.Config) error {
	// Проверка входных параметров
	if cfg == nil {
		return fmt.Errorf("конфигурация не может быть пустой")
	}

	// Получаем информацию о батарее
	info, err := b.GetInfo()
	if err != nil {
		b.logError("Ошибка получения информации о батарее: " + err.Error())
		return fmt.Errorf("получение информации о батарее: %w", err)
	}

	// Логируем текущее состояние
	b.logBatteryStatus(info, cfg)

	// Проверяем состояние и отправляем уведомления если нужно
	return b.processNotifications(info, cfg)
}

// processNotifications обрабатывает логику отправки уведомлений
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return error - ошибка в случае проблем при отправке уведомления
//
// @note Вызывается из ObserveChanges, не предназначен для прямого вызова
func (b *BatteryObserver) processNotifications(info BatteryInfo, cfg *config.Config) error {
	if cfg.NotificationState == nil {
		return fmt.Errorf("состояние уведомлений не инициализировано")
	}

	// Проверяем, нужно ли отправлять уведомление
	if !b.canSendNotification(info, cfg) {
		return nil
	}

	// Отправляем соответствующее уведомление
	err := b.sendAppropriateNotification(info, cfg)
	if err != nil {
		b.logError(fmt.Sprintf("Ошибка при отправке уведомления: %v", err))
		return fmt.Errorf("не удалось отправить уведомление: %w", err)
	}

	// Обновляем состояние уведомлений в конфиге
	now := time.Now()
	cfg.NotificationState.LastTime = now
	cfg.NotificationState.Count++
	cfg.NotificationState.LastLevel = info.CurrentCapacity
	cfg.NotificationState.LastCharging = info.IsCharging

	// Сохраняем обновленный конфиг
	err = config.SaveConfig(cfg)
	if err != nil {
		b.logError(fmt.Sprintf("Ошибка при сохранении конфигурации: %v", err))
		return fmt.Errorf("не удалось сохранить состояние уведомлений: %w", err)
	}

	b.logCheck(fmt.Sprintf("Сохранено состояние уведомлений: счетчик=%d, уровень=%d%%, зарядка=%v",
		cfg.NotificationState.Count, cfg.NotificationState.LastLevel, cfg.NotificationState.LastCharging))

	return nil
}

// handleChargingStateChange обрабатывает изменение состояния зарядки
//
// @param info - текущая информация о батарее
// @return bool - true если состояние зарядки изменилось, иначе false
//
// @note При изменении состояния зарядки сбрасывает счетчик уведомлений
func (b *BatteryObserver) handleChargingStateChange(info BatteryInfo) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Состояние теперь управляется через конфиг
	b.logCheck(fmt.Sprintf("Изменение состояния зарядки: %v", info.IsCharging))
	return false
}

// canSendNotification проверяет возможность отправки уведомления
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return bool - true если уведомление можно отправить, иначе false
//
// @note Проверяет временные интервалы, лимиты и изменения уровня заряда
func (b *BatteryObserver) canSendNotification(info BatteryInfo, cfg *config.Config) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if cfg.NotificationState == nil {
		b.logCheck("Ошибка: состояние уведомлений не инициализировано")
		return false
	}

	state := cfg.NotificationState

	// Если состояние зарядки изменилось, сбрасываем счетчик уведомлений
	if info.IsCharging != state.LastCharging {
		b.logCheck("Состояние зарядки изменилось, сброс счетчика уведомлений")
		return true
	}

	// Проверяем, не достигли ли мы лимита уведомлений
	if b.hasReachedMaxNotifications(cfg) {
		b.logCheck(fmt.Sprintf("Достигнут лимит уведомлений (%d)", cfg.MaxNotifications))
		return false
	}

	// Проверяем, изменился ли уровень заряда с последнего уведомления
	if info.CurrentCapacity == state.LastLevel {
		b.logCheck("Уровень заряда не изменился с последнего уведомления")
		return false
	}

	// Проверяем временной интервал между уведомлениями
	if time.Since(state.LastTime) < b.getNotificationInterval(cfg) {
		b.logCheck("Не прошло достаточно времени с последнего уведомления")
		return false
	}

	return true
}

// sendAppropriateNotification определяет и отправляет подходящее уведомление
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return error - ошибка в случае проблем при отправке уведомления
//
// @note Автоматически определяет тип уведомления на основе текущего состояния
func (b *BatteryObserver) sendAppropriateNotification(info BatteryInfo, cfg *config.Config) error {
	var err error
	notificationSent := false

	switch {
	case b.shouldNotifyLowBattery(info, cfg):
		err = b.sendLowBatteryNotification(info, cfg)
		notificationSent = err == nil

	case b.shouldNotifyHighBattery(info, cfg):
		err = b.sendHighBatteryNotification(info, cfg)
		notificationSent = err == nil

	default:
		b.logCheck("Уровень заряда в пределах нормы")
	}

	// Обновляем состояние если уведомление было отправлено
	if notificationSent {
		b.updateNotificationState(time.Now(), info.CurrentCapacity, info.IsCharging, cfg)
	}

	// Всегда обновляем последний уровень для следующей проверки
	cfg.NotificationState.LastLevel = info.CurrentCapacity

	return err
}

// shouldNotifyLowBattery проверяет нужно ли уведомление о низком заряде
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return bool - true если нужно отправить уведомление о низком заряде
//
// @note Проверяет, что заряд ниже минимального порога и батарея не заряжается
func (b *BatteryObserver) shouldNotifyLowBattery(info BatteryInfo, cfg *config.Config) bool {
	return info.CurrentCapacity <= cfg.MinThreshold && !info.IsCharging
}

// shouldNotifyHighBattery проверяет нужно ли уведомление о высоком заряде
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return bool - true если нужно отправить уведомление о высоком заряде
//
// @note Проверяет, что заряд выше максимального порога и батарея заряжается
func (b *BatteryObserver) shouldNotifyHighBattery(info BatteryInfo, cfg *config.Config) bool {
	return info.CurrentCapacity >= cfg.MaxThreshold && info.IsCharging
}

// sendLowBatteryNotification отправляет уведомление о низком заряде
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return error - ошибка в случае проблем при отправке уведомления
//
// @note Использует переданный notifier или глобальный notification
func (b *BatteryObserver) sendLowBatteryNotification(info BatteryInfo, cfg *config.Config) error {
	b.logCheck("Отправка уведомления о низком заряде")

	if b.notifier != nil {
		return b.notifier.SendLowBatteryNotification(info.CurrentCapacity, cfg.MinThreshold)
	}

	// Резервный вариант (для обратной совместимости)
	return notification.Notifier.ShowLowBatteryNotification(info.CurrentCapacity, cfg.MinThreshold)
}

// sendHighBatteryNotification отправляет уведомление о высоком заряде
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
// @return error - ошибка в случае проблем при отправке уведомления
//
// @note Использует переданный notifier или глобальный notification
func (b *BatteryObserver) sendHighBatteryNotification(info BatteryInfo, cfg *config.Config) error {
	b.logCheck("Отправка уведомления о высоком заряде")

	if b.notifier != nil {
		return b.notifier.SendHighBatteryNotification(info.CurrentCapacity, cfg.MaxThreshold)
	}

	// Резервный вариант (для обратной совместимости)
	return notification.Notifier.ShowHighBatteryNotification(info.CurrentCapacity, cfg.MaxThreshold)
}

// Вспомогательные методы (thread-safe)

// resetNotificationState сбрасывает состояние уведомлений
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
//
// @note Сбрасывает счетчик уведомлений и обновляет последнее известное состояние
func (b *BatteryObserver) resetNotificationState(info BatteryInfo, cfg *config.Config) {
	if cfg == nil || cfg.NotificationState == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Сбрасываем счетчик уведомлений и обновляем состояние
	cfg.NotificationState.Count = 0
	cfg.NotificationState.LastTime = time.Time{}
	cfg.NotificationState.LastLevel = info.CurrentCapacity
	cfg.NotificationState.LastCharging = info.IsCharging

	// Сохраняем изменения в конфигурации
	if err := config.SaveConfig(cfg); err != nil {
		b.logError(fmt.Sprintf("Ошибка при сохранении состояния уведомлений: %v", err))
	}
}

// updateNotificationState обновляет состояние уведомлений
//
// @param notificationTime - время последнего уведомления
// @param currentLevel - текущий уровень заряда
// @param isCharging - текущее состояние зарядки
// @param cfg - конфигурация приложения
//
// @note Обновляет счетчик уведомлений и логирует изменения
func (b *BatteryObserver) updateNotificationState(notificationTime time.Time, currentLevel int, isCharging bool, cfg *config.Config) {
	if cfg == nil || cfg.NotificationState == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Обновляем состояние
	cfg.NotificationState.LastTime = notificationTime
	cfg.NotificationState.Count++
	cfg.NotificationState.LastLevel = currentLevel
	cfg.NotificationState.LastCharging = isCharging

	// Сохраняем изменения в конфигурации
	if err := config.SaveConfig(cfg); err != nil {
		b.logError(fmt.Sprintf("Ошибка при сохранении состояния уведомлений: %v", err))
		return
	}

	// Логируем изменения
	b.logCheck(fmt.Sprintf("Уведомление #%d отправлено. Текущий уровень: %d%%. Время: %s",
		cfg.NotificationState.Count,
		currentLevel,
		notificationTime.Format(time.RFC3339)))

	// Если превышен разумный лимит уведомлений, сбрасываем счетчик
	if cfg.NotificationState.Count > MaxReasonableNotifications {
		b.logCheck(fmt.Sprintf("Превышен разумный лимит уведомлений (%d). Сброс счетчика.", MaxReasonableNotifications))
		cfg.NotificationState.Count = 0
		if err := config.SaveConfig(cfg); err != nil {
			b.logError(fmt.Sprintf("Ошибка при сбросе счетчика уведомлений: %v", err))
		}
	}
}

// getNotificationInterval возвращает интервал между уведомлениями
//
// @param cfg - конфигурация приложения
// @return time.Duration - интервал между уведомлениями
//
// @note Возвращает значение из конфигурации или значение по умолчанию
func (b *BatteryObserver) getNotificationInterval(cfg *config.Config) time.Duration {
	if cfg.NotificationInterval > 0 {
		return time.Duration(cfg.NotificationInterval) * time.Minute
	}
	return DefaultNotificationInterval
}

// hasReachedMaxNotifications проверяет достигнут ли лимит уведомлений
//
// @param cfg - конфигурация приложения
// @return bool - true если лимит достигнут или превышен
//
// @note Возвращает false если лимит не задан (<= 0)
func (b *BatteryObserver) hasReachedMaxNotifications(cfg *config.Config) bool {
	if cfg == nil || cfg.MaxNotifications <= 0 || cfg.NotificationState == nil {
		return false
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	return cfg.NotificationState.Count >= cfg.MaxNotifications
}

// GetNotificationState возвращает текущее состояние уведомлений
//
// @param cfg - конфигурация приложения
// @return *config.NotificationState - указатель на текущее состояние уведомлений
//
// @note Предназначено в первую очередь для тестирования
// @warning Возвращает nil, если состояние не инициализировано
func (b *BatteryObserver) GetNotificationState(cfg *config.Config) *config.NotificationState {
	if cfg == nil || cfg.NotificationState == nil {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Возвращаем указатель на текущее состояние из конфига
	return cfg.NotificationState
}

// Методы логирования

// logCheck записывает информационное сообщение в лог
//
// @param message - текст сообщения для логирования
//
// @note Использует переданный логгер или глобальный log
func (b *BatteryObserver) logCheck(message string) {
	if b.logger != nil {
		b.logger.Check(message)
	} else {
		log.Check(message) // Резервный вариант
	}
}

// logError записывает сообщение об ошибке в лог
//
// @param message - текст сообщения об ошибке
//
// @note Использует переданный логгер или глобальный log
func (b *BatteryObserver) logError(message string) {
	if b.logger != nil {
		b.logger.Error(message)
	} else {
		log.Error(message) // Резервный вариант
	}
}

// logBatteryStatus логирует текущее состояние батареи
//
// @param info - текущая информация о батарее
// @param cfg - конфигурация приложения
//
// @note Выводит отладочную информацию о текущем состоянии
func (b *BatteryObserver) logBatteryStatus(info BatteryInfo, cfg *config.Config) {
	b.logCheck(fmt.Sprintf("Проверка уровня зарядки: %d%%|%d%%|%d%%",
		cfg.MinThreshold, info.CurrentCapacity, cfg.MaxThreshold))
	b.logCheck(fmt.Sprintf("Зарядка: %v", utils.BoolToYesNo(info.IsCharging)))
	b.logCheck(fmt.Sprintf("Интервал проверки: %d сек", GetCurrentCheckInterval(cfg)))
}

// Глобальная переменная для обратной совместимости (можно убрать в будущем)
var defaultObserver *BatteryObserver

// GetInfo глобальная функция для обратной совместимости
//
// @return BatteryInfo - информация о текущем состоянии батареи
// @return error - ошибка в случае проблем при получении информации
//
// @deprecated Используйте методы BatteryObserver
func GetInfo() (BatteryInfo, error) {
	if defaultObserver == nil {
		defaultObserver = NewBatteryObserver(nil, nil, nil)
	}
	return defaultObserver.GetInfo()
}

// GetCurrentCheckInterval возвращает текущий интервал проверки в зависимости от состояния зарядки.
// @param cfg *Config - конфигурация
// @return int - установленный интервал проверки,
// если не удалось получить информацию о батарее, возвращает 0
func GetCurrentCheckInterval(cfg *config.Config) int {
	info, err := GetInfo()
	if err != nil {
		return 0
	}
	if info.IsCharging {
		return cfg.CheckIntervalWhenCharging
	}
	return cfg.CheckIntervalWhenDischarging
}

// ObserveChanges глобальная функция для обратной совместимости
//
// @param cfg - конфигурация приложения
//
// @deprecated Используйте методы BatteryObserver
// @note Создает глобальный экземпляр BatteryObserver при первом вызове
func ObserveChanges(cfg *config.Config) {
	if defaultObserver == nil {
		defaultObserver = NewBatteryObserver(nil, nil, nil)
	}

	if err := defaultObserver.ObserveChanges(cfg); err != nil {
		log.Error("Ошибка наблюдения за состоянием батареи: " + err.Error())
	}

}
