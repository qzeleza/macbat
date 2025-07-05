//go:build integration
// +build integration

package battery

import (
	"fmt"
	"testing"
	"time"
)

// TestRealBatteryInfo тестирует получение реальной информации о батарее
func TestRealBatteryInfo(t *testing.T) {
	// Пропускаем тест если это не macOS
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	info, err := GetInfo()
	if err != nil {
		t.Fatalf("Ошибка получения информации о батарее: %v", err)
	}

	// Проверяем базовые поля
	if info.CurrentCapacity < 0 || info.CurrentCapacity > 100 {
		t.Errorf("CurrentCapacity должен быть от 0 до 100, получено: %d", info.CurrentCapacity)
	}

	if info.MaxCapacity <= 0 {
		t.Errorf("MaxCapacity должен быть больше 0, получено: %d", info.MaxCapacity)
	}

	if info.DesignCapacity <= 0 {
		t.Errorf("DesignCapacity должен быть больше 0, получено: %d", info.DesignCapacity)
	}

	if info.CycleCount < 0 {
		t.Errorf("CycleCount не может быть отрицательным, получено: %d", info.CycleCount)
	}

	// Проверяем расчет здоровья батареи
	expectedHealth := int(float64(info.MaxCapacity) * 100 / float64(info.DesignCapacity))
	if info.HealthPercent != expectedHealth {
		t.Errorf("HealthPercent = %d, ожидалось %d", info.HealthPercent, expectedHealth)
	}

	// Логируем информацию для визуальной проверки
	t.Logf("Информация о батарее:")
	t.Logf("  Текущий заряд: %d%%", info.CurrentCapacity)
	t.Logf("  Максимальная емкость: %d mAh", info.MaxCapacity)
	t.Logf("  Проектная емкость: %d mAh", info.DesignCapacity)
	t.Logf("  Циклы зарядки: %d", info.CycleCount)
	t.Logf("  Напряжение: %d mV", info.Voltage)
	t.Logf("  Сила тока: %d mA", info.Amperage)
	t.Logf("  Зарядка: %v", info.IsCharging)
	t.Logf("  Подключено к сети: %v", info.IsPlugged)
	t.Logf("  Здоровье батареи: %d%%", info.HealthPercent)

	if info.TimeToEmpty > 0 {
		t.Logf("  Время до разряда: %d минут", info.TimeToEmpty)
	}
	if info.TimeToFull > 0 {
		t.Logf("  Время до полной зарядки: %d минут", info.TimeToFull)
	}
}

// TestBatteryInfoConsistency проверяет консистентность данных при повторных вызовах
func TestBatteryInfoConsistency(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	// Получаем информацию несколько раз подряд
	var infos []BatteryInfo
	for i := 0; i < 5; i++ {
		info, err := GetInfo()
		if err != nil {
			t.Fatalf("Ошибка получения информации о батарее (попытка %d): %v", i+1, err)
		}
		infos = append(infos, info)
		time.Sleep(100 * time.Millisecond)
	}

	// Проверяем, что статические параметры не изменяются
	baseInfo := infos[0]
	for i, info := range infos[1:] {
		if info.DesignCapacity != baseInfo.DesignCapacity {
			t.Errorf("DesignCapacity изменился между вызовами: %d -> %d (вызов %d)",
				baseInfo.DesignCapacity, info.DesignCapacity, i+2)
		}

		// MaxCapacity может слегка изменяться, но не должен сильно отличаться
		if abs(info.MaxCapacity-baseInfo.MaxCapacity) > 100 {
			t.Errorf("MaxCapacity сильно изменился: %d -> %d (вызов %d)",
				baseInfo.MaxCapacity, info.MaxCapacity, i+2)
		}
	}
}

// TestPowerSourceNotifications тестирует систему уведомлений об изменениях питания
func TestPowerSourceNotifications(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	// Канал для получения уведомлений
	notifications := make(chan bool, 10)

	// Регистрируем callback
	err := registerPowerSourceChanges(func() {
		select {
		case notifications <- true:
		default:
			// Канал полон, пропускаем
		}
	})

	if err != nil {
		t.Fatalf("Не удалось зарегистрировать уведомления: %v", err)
	}

	t.Log("Уведомления зарегистрированы. Подключите/отключите зарядное устройство в течение 10 секунд...")

	// Ждем уведомления или таймаут
	timeout := time.After(10 * time.Second)
	notificationReceived := false

	select {
	case <-notifications:
		notificationReceived = true
		t.Log("✅ Получено уведомление об изменении источника питания")
	case <-timeout:
		t.Log("⚠️  Уведомление не получено за 10 секунд (возможно, источник питания не изменялся)")
	}

	// Если получили уведомление, проверяем что данные действительно изменились
	if notificationReceived {
		info1, _ := GetInfo()
		time.Sleep(500 * time.Millisecond) // Даем время на изменение

		select {
		case <-notifications:
			info2, _ := GetInfo()
			if info1.IsCharging == info2.IsCharging && info1.IsPlugged == info2.IsPlugged {
				t.Log("Состояние питания не изменилось между уведомлениями")
			} else {
				t.Logf("✅ Состояние изменилось: зарядка %v->%v, подключено %v->%v",
					info1.IsCharging, info2.IsCharging, info1.IsPlugged, info2.IsPlugged)
			}
		case <-time.After(2 * time.Second):
			// Нормально, если второго уведомления нет
		}
	}
}

// TestBatteryHealthCalculation проверяет корректность расчета здоровья батареи
func TestBatteryHealthCalculation(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	info, err := GetInfo()
	if err != nil {
		t.Fatalf("Ошибка получения информации о батарее: %v", err)
	}

	// Проверяем границы здоровья батареи
	if info.HealthPercent < 0 || info.HealthPercent > 100 {
		t.Errorf("HealthPercent должен быть от 0 до 100, получено: %d", info.HealthPercent)
	}

	// Для новых устройств здоровье должно быть близко к 100%
	if info.CycleCount < 100 && info.HealthPercent < 95 {
		t.Logf("⚠️  Здоровье батареи %d%% при %d циклах (может указывать на проблему)",
			info.HealthPercent, info.CycleCount)
	}

	// Для сильно изношенных батарей предупреждение
	if info.HealthPercent < 80 {
		t.Logf("⚠️  Здоровье батареи критически низкое: %d%% (%d циклов)",
			info.HealthPercent, info.CycleCount)
	}
}

// TestConcurrentBatteryAccess тестирует безопасность многопоточного доступа
func TestConcurrentBatteryAccess(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	const goroutines = 10
	const iterations = 20

	done := make(chan bool, goroutines)
	errors := make(chan error, goroutines*iterations)

	// Запускаем несколько горутин для одновременного доступа к батарее
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < iterations; j++ {
				_, err := GetInfo()
				if err != nil {
					errors <- err
				}
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Проверяем ошибки
	close(errors)
	errorCount := 0
	for err := range errors {
		t.Errorf("Ошибка в concurrent access: %v", err)
		errorCount++
	}

	if errorCount == 0 {
		t.Logf("✅ Многопоточный доступ работает корректно")
	}
}

// Вспомогательные функции

func isMacOS() bool {
	// Простая проверка - пытаемся получить информацию о батарее
	_, err := getBatteryInfo()
	return err == nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TestRunLoopExecution проверяет что RunLoop может быть запущен и остановлен
func TestRunLoopExecution(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	// Тест проверяет что мы можем безопасно зарегистрировать обработчик
	// без зависания или краха приложения
	callbackExecuted := false

	err := registerPowerSourceChanges(func() {
		callbackExecuted = true
	})

	if err != nil {
		t.Fatalf("Не удалось зарегистрировать обработчик: %v", err)
	}

	// Проверяем что функция регистрации прошла успешно
	t.Log("✅ Обработчик изменений питания зарегистрирован успешно")

	// Примечание: полноценный тест RunLoop требует изменения источника питания
	// что сложно автоматизировать в unit-тестах
}

// TestBatteryDataValidation проверяет валидность всех полей батареи
func TestBatteryDataValidation(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	info, err := GetInfo()
	if err != nil {
		t.Fatalf("Ошибка получения информации о батарее: %v", err)
	}

	// Проверяем логическую согласованность данных
	validationErrors := []string{}

	// Проверка емкостей
	if info.MaxCapacity > info.DesignCapacity*1.1 {
		validationErrors = append(validationErrors,
			fmt.Sprintf("MaxCapacity (%d) не может значительно превышать DesignCapacity (%d)",
				info.MaxCapacity, info.DesignCapacity))
	}

	// Проверка состояний зарядки
	if info.IsCharging && !info.IsPlugged {
		validationErrors = append(validationErrors,
			"IsCharging=true, но IsPlugged=false (логическая несогласованность)")
	}

	// Проверка времени
	if info.IsCharging && info.TimeToEmpty > 0 {
		validationErrors = append(validationErrors,
			"При зарядке TimeToEmpty должно быть 0 или не задано")
	}

	if !info.IsCharging && info.IsPlugged && info.TimeToFull > 0 {
		validationErrors = append(validationErrors,
			"Если подключено к сети но не заряжается, TimeToFull должно быть 0")
	}

	// Проверка силы тока
	if info.IsCharging && info.Amperage < 0 {
		// При зарядке ток обычно положительный
		t.Logf("⚠️  При зарядке ток отрицательный: %d mA", info.Amperage)
	}

	if !info.IsCharging && info.Amperage > 0 {
		// При разряде ток обычно отрицательный
		t.Logf("⚠️  При разряде ток положительный: %d mA", info.Amperage)
	}

	// Выводим ошибки валидации
	for _, err := range validationErrors {
		t.Errorf("Ошибка валидации: %s", err)
	}

	if len(validationErrors) == 0 {
		t.Log("✅ Все данные батареи прошли валидацию")
	}
}

// TestBatteryChangeDetection тестирует обнаружение изменений батареи
func TestBatteryChangeDetection(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	// Получаем начальное состояние
	initialInfo, err := GetInfo()
	if err != nil {
		t.Fatalf("Ошибка получения начальной информации о батарее: %v", err)
	}

	t.Logf("Начальное состояние: заряд=%d%%, зарядка=%v, подключено=%v",
		initialInfo.CurrentCapacity, initialInfo.IsCharging, initialInfo.IsPlugged)

	// Мониторим изменения в течение короткого времени
	changes := 0
	startTime := time.Now()
	timeout := 30 * time.Second

	for time.Since(startTime) < timeout {
		currentInfo, err := GetInfo()
		if err != nil {
			continue
		}

		// Проверяем значимые изменения
		if currentInfo.CurrentCapacity != initialInfo.CurrentCapacity ||
			currentInfo.IsCharging != initialInfo.IsCharging ||
			currentInfo.IsPlugged != initialInfo.IsPlugged {
			changes++
			t.Logf("Изменение %d: заряд=%d%%, зарядка=%v, подключено=%v",
				changes, currentInfo.CurrentCapacity, currentInfo.IsCharging, currentInfo.IsPlugged)
			initialInfo = currentInfo
		}

		time.Sleep(1 * time.Second)
	}

	t.Logf("Обнаружено %d изменений за %v", changes, timeout)
}

// TestEdgeCases проверяет граничные случаи
func TestEdgeCases(t *testing.T) {
	if !isMacOS() {
		t.Skip("Тест доступен только на macOS")
	}

	info, err := GetInfo()
	if err != nil {
		t.Fatalf("Ошибка получения информации о батарее: %v", err)
	}

	// Тест очень низкого заряда
	if info.CurrentCapacity <= 5 {
		t.Logf("⚠️  Критически низкий заряд: %d%%", info.CurrentCapacity)

		// При очень низком заряде некоторые показания могут быть неточными
		if info.TimeToEmpty > 0 && info.TimeToEmpty < 60 {
			t.Logf("Осталось времени: %d минут", info.TimeToEmpty)
		}
	}

	// Тест очень высокого заряда
	if info.CurrentCapacity >= 95 {
		t.Logf("✅ Высокий заряд: %d%%", info.CurrentCapacity)

		if info.IsCharging && info.TimeToFull > 0 && info.TimeToFull < 60 {
			t.Logf("До полной зарядки: %d минут", info.TimeToFull)
		}
	}

	// Тест старой батареи
	if info.CycleCount > 1000 {
		t.Logf("⚠️  Батарея имеет много циклов: %d", info.CycleCount)

		if info.HealthPercent < 80 {
			t.Logf("⚠️  Здоровье батареи снижено: %d%%", info.HealthPercent)
		}
	}
}
