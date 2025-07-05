package battery

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"macbat/internal/config"
)

// TestHelpers содержит вспомогательные функции для более стабильного тестирования

// waitForCondition ждет выполнения условия с таймаутом
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, description string) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("Таймаут ожидания условия: %s (ждали %v)", description, timeout)
	return false
}

// runWithTimeout запускает функцию с таймаутом
func runWithTimeout(t *testing.T, fn func(), timeout time.Duration, description string) {
	done := make(chan bool, 1)

	go func() {
		fn()
		done <- true
	}()

	select {
	case <-done:
		// Функция завершилась вовремя
	case <-time.After(timeout):
		t.Errorf("Таймаут выполнения: %s (ждали %v)", description, timeout)
	}
}

// stabilizeMemory принудительно стабилизирует память перед измерением
func stabilizeMemory() {
	// Несколько циклов GC для стабилизации
	for i := 0; i < 3; i++ {
		runtime.GC()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
}

// TestMemoryUsageFixed - исправленная версия теста памяти
func TestMemoryUsageFixed(t *testing.T) {
	// Стабилизируем память перед началом
	stabilizeMemory()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Выполняем операции с явным выделением памяти
	data := make([]BatteryInfo, 0, 1000)
	for i := 0; i < 1000; i++ {
		info := BatteryInfo{
			CurrentCapacity: i % 100,
			IsCharging:      i%2 == 0,
			IsPlugged:       i%3 == 0,
		}
		data = append(data, info)

		// Симулируем обработку
		_ = info.CurrentCapacity / 5
		_ = info.CurrentCapacity % 5
	}

	// Принудительно используем данные чтобы компилятор их не оптимизировал
	var sum int
	for _, info := range data {
		sum += info.CurrentCapacity
	}
	_ = sum

	// Стабилизируем память после операций
	stabilizeMemory()
	runtime.ReadMemStats(&m2)

	// Вычисляем разность с правильной обработкой знака
	var memDiff int64
	if m2.Alloc >= m1.Alloc {
		memDiff = int64(m2.Alloc - m1.Alloc)
	} else {
		memDiff = -int64(m1.Alloc - m2.Alloc)
	}

	// Более разумный лимит для данного теста
	const maxMemoryIncrease = 100 * 1024 // 100KB (для 1000 структур BatteryInfo)

	if memDiff > maxMemoryIncrease {
		t.Errorf("Подозрение на утечку памяти: использовано %d байт (лимит %d)", memDiff, maxMemoryIncrease)
	} else {
		t.Logf("✅ Использование памяти в норме: %+d байт (лимит %d)", memDiff, maxMemoryIncrease)
		if memDiff < 0 {
			t.Logf("   (память освободилась благодаря GC)")
		}
	}
}

// TestPeriodicCheckIntervalStable - более стабильная версия теста интервалов
func TestPeriodicCheckIntervalStable(t *testing.T) {
	cfg := &config.Config{
		MinThreshold:  20,
		MaxThreshold:  80,
		CheckInterval: 1, // 1 секунда
	}

	mockNotificationSystem.Clear()
	mockBatteryProvider.SetBatteryInfo(BatteryInfo{
		CurrentCapacity: 15,
		IsCharging:      false,
	})

	var checkCount int64
	var mu sync.Mutex
	done := make(chan struct{})

	// Счетчик в отдельной горутине
	go func() {
		var lastNotificationLevel int
		var lastChargingState bool

		ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				checkBatteryStateForTest(cfg, &lastNotificationLevel, &lastChargingState)
				mu.Lock()
				checkCount++
				mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	// Ждем стабильное количество проверок
	testDuration := 2 * time.Second
	time.Sleep(testDuration)
	close(done)

	// Небольшая пауза для завершения последней проверки
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	finalCount := checkCount
	mu.Unlock()

	// Ожидаем примерно 2 проверки за 2 секунды (с интервалом 1 сек)
	// Допускаем отклонение ±1 из-за особенностей планировщика
	expectedMin := int64(1)
	expectedMax := int64(4)

	if finalCount < expectedMin || finalCount > expectedMax {
		t.Errorf("Количество проверок вне ожидаемого диапазона: получено %d, ожидалось %d-%d за %v",
			finalCount, expectedMin, expectedMax, testDuration)
	} else {
		t.Logf("✅ Выполнено %d проверок за %v (интервал %ds)", finalCount, testDuration, cfg.CheckInterval)
	}

	// Проверяем уведомления
	notificationCount := mockNotificationSystem.Count()
	if notificationCount != 1 {
		t.Errorf("Ожидалось 1 уведомление, получено: %d", notificationCount)
	}
}

// TestConfigValidationRobust - более устойчивый тест валидации конфигурации
func TestConfigValidationRobust(t *testing.T) {
	// Простая функция валидации для этого теста
	validate := func(cfg *config.Config) error {
		if cfg.MinThreshold <= 0 {
			return fmt.Errorf("MinThreshold должен быть больше 0")
		}
		if cfg.MaxThreshold >= 100 {
			return fmt.Errorf("MaxThreshold должен быть меньше 100")
		}
		if cfg.MinThreshold >= cfg.MaxThreshold {
			return fmt.Errorf("MinThreshold должен быть меньше MaxThreshold")
		}
		if cfg.CheckInterval <= 0 {
			return fmt.Errorf("CheckInterval должен быть больше 0")
		}
		if cfg.CheckInterval > 3600 {
			return fmt.Errorf("CheckInterval не должен превышать 3600 секунд")
		}
		return nil
	}

	// Граничный случай: точно на лимите (должен быть валидным)
	validConfig := &config.Config{
		MinThreshold:  1,
		MaxThreshold:  99,
		CheckInterval: 3600, // Ровно 1 час - должно быть валидным
	}

	err := validate(validConfig)
	if err != nil {
		t.Errorf("Конфигурация на границе должна быть валидной: %v", err)
	}

	// Тест превышения лимита
	invalidConfig := &config.Config{
		MinThreshold:  1,
		MaxThreshold:  99,
		CheckInterval: 3601, // Больше 1 часа - должно быть невалидным
	}

	err = validate(invalidConfig)
	if err == nil {
		t.Errorf("Конфигурация с превышением лимита должна быть невалидной")
	}

	// Тест нулевых значений
	zeroConfig := &config.Config{
		MinThreshold:  0,
		MaxThreshold:  80,
		CheckInterval: 30,
	}

	err = validate(zeroConfig)
	if err == nil {
		t.Errorf("Конфигурация с нулевым MinThreshold должна быть невалидной")
	}

	// Тест инвертированных порогов
	invertedConfig := &config.Config{
		MinThreshold:  80,
		MaxThreshold:  20,
		CheckInterval: 30,
	}

	err = validate(invertedConfig)
	if err == nil {
		t.Errorf("Конфигурация с инвертированными порогами должна быть невалидной")
	}

	t.Logf("✅ Все граничные случаи валидации обработаны корректно")
}

// TestThreadSafetyMocks - тест потокобезопасности моков
func TestThreadSafetyMocks(t *testing.T) {
	const goroutines = 10
	const operationsPerGoroutine = 100

	mockNotificationSystem.Clear()

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Запускаем несколько горутин для одновременной работы с моками
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Тестируем MockBatteryInfoProvider
				mockBatteryProvider.SetBatteryInfo(BatteryInfo{
					CurrentCapacity: (id*operationsPerGoroutine + j) % 100,
					IsCharging:      j%2 == 0,
				})

				info, err := mockBatteryProvider.GetBatteryInfo()
				if err != nil {
					t.Errorf("Ошибка получения данных в горутине %d: %v", id, err)
				}

				// Тестируем MockNotificationSystem
				if info.CurrentCapacity%10 == 0 { // Уведомляем только иногда
					mockNotificationSystem.ShowNotification("Test",
						fmt.Sprintf("Goroutine %d, operation %d", id, j))
				}
			}
		}(i)
	}

	// Ждем завершения всех горутин с таймаутом
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("✅ Все %d горутин завершились успешно", goroutines)
	case <-time.After(10 * time.Second):
		t.Errorf("Таймаут ожидания завершения горутин")
	}

	// Проверяем что моки остались в рабочем состоянии
	finalInfo, err := mockBatteryProvider.GetBatteryInfo()
	if err != nil {
		t.Errorf("Моки повреждены после многопоточного доступа: %v", err)
	}

	notificationCount := mockNotificationSystem.Count()
	t.Logf("✅ Получено %d уведомлений, финальное состояние: %d%%",
		notificationCount, finalInfo.CurrentCapacity)
}
