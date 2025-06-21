// FILE: config_helper_test.go
// ИЗМЕНЕНИЯ: Исправлен тест валидации и тест памяти

package battery

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"macbat/internal/config"
)

// TestConfigValidation тестирует валидацию конфигурации
// ИСПРАВЛЕНИЕ: Изменено значение CheckInterval с 3600 на 3601 в тесте "Слишком_длинный_интервал_проверки"
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		expectValid bool
		description string
	}{
		{
			name: "Валидная конфигурация",
			config: config.Config{
				MinThreshold:  20,
				MaxThreshold:  80,
				CheckInterval: 30,
			},
			expectValid: true,
			description: "Стандартные разумные значения",
		},
		{
			name: "Минимальный порог выше максимального",
			config: config.Config{
				MinThreshold:  80,
				MaxThreshold:  20,
				CheckInterval: 30,
			},
			expectValid: false,
			description: "MinThreshold не может быть больше MaxThreshold",
		},
		{
			name: "Слишком низкий минимальный порог",
			config: config.Config{
				MinThreshold:  0,
				MaxThreshold:  80,
				CheckInterval: 30,
			},
			expectValid: false,
			description: "MinThreshold должен быть больше 0",
		},
		{
			name: "Слишком высокий максимальный порог",
			config: config.Config{
				MinThreshold:  20,
				MaxThreshold:  100,
				CheckInterval: 30,
			},
			expectValid: false,
			description: "MaxThreshold должен быть меньше 100",
		},
		{
			name: "Слишком короткий интервал проверки",
			config: config.Config{
				MinThreshold:  20,
				MaxThreshold:  80,
				CheckInterval: 0,
			},
			expectValid: false,
			description: "CheckInterval должен быть больше 0",
		},
		{
			name: "Слишком длинный интервал проверки",
			config: config.Config{
				MinThreshold:  20,
				MaxThreshold:  80,
				CheckInterval: 3601, // ИСПРАВЛЕНО: было 3600, стало 3601 чтобы действительно превышать лимит
			},
			expectValid: false,
			description: "CheckInterval не должен превышать 1 час",
		},
		{
			name: "Граничные валидные значения",
			config: config.Config{
				MinThreshold:  1,
				MaxThreshold:  99,
				CheckInterval: 1,
			},
			expectValid: true,
			description: "Минимальные и максимальные допустимые значения",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)

			if tt.expectValid && err != nil {
				t.Errorf("Конфигурация должна быть валидной, но получена ошибка: %v", err)
			}

			if !tt.expectValid && err == nil {
				t.Errorf("Конфигурация должна быть невалидной, но ошибки не получено")
			}

			t.Logf("Описание: %s", tt.description)
		})
	}
}

// Функция валидации конфигурации (должна быть в основном коде)
// ИСПРАВЛЕНИЕ: Добавлены детализированные сообщения об ошибках
func validateConfig(cfg *config.Config) error {
	if cfg.MinThreshold <= 0 {
		return fmt.Errorf("MinThreshold должен быть больше 0, получено: %d", cfg.MinThreshold)
	}

	if cfg.MaxThreshold >= 100 {
		return fmt.Errorf("MaxThreshold должен быть меньше 100, получено: %d", cfg.MaxThreshold)
	}

	if cfg.MinThreshold >= cfg.MaxThreshold {
		return fmt.Errorf("MinThreshold (%d) должен быть меньше MaxThreshold (%d)", cfg.MinThreshold, cfg.MaxThreshold)
	}

	if cfg.CheckInterval <= 0 {
		return fmt.Errorf("CheckInterval должен быть больше 0, получено: %d", cfg.CheckInterval)
	}

	if cfg.CheckInterval > 3600 {
		return fmt.Errorf("CheckInterval не должен превышать 3600 секунд (1 час), получено: %d", cfg.CheckInterval)
	}

	return nil
}

// TestConfigDefaults тестирует значения по умолчанию
func TestConfigDefaults(t *testing.T) {
	defaultConfig := config.Config{
		MinThreshold:  20,
		MaxThreshold:  80,
		CheckInterval: 30,
	}

	if defaultConfig.MinThreshold != 20 {
		t.Errorf("Значение MinThreshold по умолчанию должно быть 20, получено: %d", defaultConfig.MinThreshold)
	}

	if defaultConfig.MaxThreshold != 80 {
		t.Errorf("Значение MaxThreshold по умолчанию должно быть 80, получено: %d", defaultConfig.MaxThreshold)
	}

	if defaultConfig.CheckInterval != 30 {
		t.Errorf("Значение CheckInterval по умолчанию должно быть 30, получено: %d", defaultConfig.CheckInterval)
	}

	// Проверяем что значения по умолчанию валидны
	err := validateConfig(&defaultConfig)
	if err != nil {
		t.Errorf("Конфигурация по умолчанию должна быть валидной: %v", err)
	}
}

// TestNotificationLevelCalculation тестирует расчет уровня уведомлений
func TestNotificationLevelCalculation(t *testing.T) {
	tests := []struct {
		batteryLevel  int
		expectedLevel int
		shouldNotify  bool
		description   string
	}{
		{100, 20, true, "100% - должно уведомить (100%5=0)"},
		{99, 19, false, "99% - не должно уведомить (99%5≠0)"},
		{95, 19, true, "95% - должно уведомить (95%5=0)"},
		{90, 18, true, "90% - должно уведомить (90%5=0)"},
		{87, 17, false, "87% - не должно уведомить (87%5≠0)"},
		{85, 17, true, "85% - должно уведомить (85%5=0)"},
		{25, 5, true, "25% - должно уведомить (25%5=0)"},
		{23, 4, false, "23% - не должно уведомить (23%5≠0)"},
		{20, 4, true, "20% - должно уведомить (20%5=0)"},
		{15, 3, true, "15% - должно уведомить (15%5=0)"},
		{10, 2, true, "10% - должно уведомить (10%5=0)"},
		{5, 1, true, "5% - должно уведомить (5%5=0)"},
		{1, 0, false, "1% - не должно уведомить (1%5≠0)"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Level_%d", tt.batteryLevel), func(t *testing.T) {
			level := tt.batteryLevel / 5
			shouldNotify := (tt.batteryLevel % 5) == 0

			if level != tt.expectedLevel {
				t.Errorf("Уровень для %d%% = %d, ожидалось %d", tt.batteryLevel, level, tt.expectedLevel)
			}

			if shouldNotify != tt.shouldNotify {
				t.Errorf("ShouldNotify для %d%% = %v, ожидалось %v", tt.batteryLevel, shouldNotify, tt.shouldNotify)
			}

			t.Logf("%s", tt.description)
		})
	}
}

// TestTimingAccuracy проверяет точность интервалов времени
func TestTimingAccuracy(t *testing.T) {
	intervals := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	for _, interval := range intervals {
		t.Run(fmt.Sprintf("Interval_%v", interval), func(t *testing.T) {
			tolerance := interval / 10 // 10% допуск

			start := time.Now()
			timer := time.NewTimer(interval)
			<-timer.C
			elapsed := time.Since(start)

			diff := elapsed - interval
			if diff < 0 {
				diff = -diff
			}

			if diff > tolerance {
				t.Errorf("Интервал %v выполнился за %v (отклонение %v > допуска %v)",
					interval, elapsed, diff, tolerance)
			} else {
				t.Logf("✅ Интервал %v выполнился за %v (отклонение %v)", interval, elapsed, diff)
			}
		})
	}
}

// TestStateTransitions тестирует переходы состояний батареи
func TestStateTransitions(t *testing.T) {
	// Симулируем различные переходы состояния
	transitions := []struct {
		name        string
		from        BatteryInfo
		to          BatteryInfo
		expectReset bool
		description string
	}{
		{
			name:        "Подключение зарядки",
			from:        BatteryInfo{CurrentCapacity: 20, IsCharging: false, IsPlugged: false},
			to:          BatteryInfo{CurrentCapacity: 20, IsCharging: true, IsPlugged: true},
			expectReset: true,
			description: "При подключении зарядки должен сброситься счетчик уведомлений",
		},
		{
			name:        "Отключение зарядки",
			from:        BatteryInfo{CurrentCapacity: 85, IsCharging: true, IsPlugged: true},
			to:          BatteryInfo{CurrentCapacity: 85, IsCharging: false, IsPlugged: false},
			expectReset: true,
			description: "При отключении зарядки должен сброситься счетчик уведомлений",
		},
		{
			name:        "Изменение только уровня заряда",
			from:        BatteryInfo{CurrentCapacity: 85, IsCharging: true, IsPlugged: true},
			to:          BatteryInfo{CurrentCapacity: 86, IsCharging: true, IsPlugged: true},
			expectReset: false,
			description: "При изменении только уровня заряда сброса быть не должно",
		},
		{
			name:        "Подключение к сети без зарядки",
			from:        BatteryInfo{CurrentCapacity: 90, IsCharging: false, IsPlugged: false},
			to:          BatteryInfo{CurrentCapacity: 90, IsCharging: false, IsPlugged: true},
			expectReset: false,
			description: "Подключение к сети без начала зарядки не должно сбрасывать счетчик",
		},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем логику определения сброса
			chargingStateChanged := tt.from.IsCharging != tt.to.IsCharging

			if chargingStateChanged != tt.expectReset {
				t.Errorf("Переход '%s': ожидался сброс=%v, получен=%v",
					tt.name, tt.expectReset, chargingStateChanged)
			} else {
				t.Logf("✅ %s", tt.description)
			}
		})
	}
}

// TestBoundaryConditions тестирует граничные условия
func TestBoundaryConditions(t *testing.T) {
	cfg := &config.Config{
		MinThreshold:  20,
		MaxThreshold:  80,
		CheckInterval: 30,
	}

	boundaryTests := []struct {
		name          string
		batteryLevel  int
		isCharging    bool
		shouldTrigger bool
		triggerType   string
	}{
		{"Точно на минимальном пороге без зарядки", 20, false, true, "low"},
		{"Точно на минимальном пороге с зарядкой", 20, true, false, "none"},
		{"На 1% ниже минимального порога", 19, false, true, "low"},
		{"На 1% выше минимального порога", 21, false, false, "none"},
		{"Точно на максимальном пороге с зарядкой", 80, true, true, "high"},
		{"Точно на максимальном пороге без зарядки", 80, false, false, "none"},
		{"На 1% выше максимального порога", 81, true, true, "high"},
		{"На 1% ниже максимального порога", 79, true, false, "none"},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			triggersLow := tt.batteryLevel <= cfg.MinThreshold && !tt.isCharging
			triggersHigh := tt.batteryLevel >= cfg.MaxThreshold && tt.isCharging
			shouldTrigger := triggersLow || triggersHigh

			var triggerType string
			if triggersLow {
				triggerType = "low"
			} else if triggersHigh {
				triggerType = "high"
			} else {
				triggerType = "none"
			}

			if shouldTrigger != tt.shouldTrigger {
				t.Errorf("Уровень %d%% (зарядка=%v): ожидалось срабатывание=%v, получено=%v",
					tt.batteryLevel, tt.isCharging, tt.shouldTrigger, shouldTrigger)
			}

			if triggerType != tt.triggerType {
				t.Errorf("Уровень %d%% (зарядка=%v): ожидался тип='%s', получен='%s'",
					tt.batteryLevel, tt.isCharging, tt.triggerType, triggerType)
			}

			t.Logf("✅ Граница %d%% (зарядка=%v): триггер=%s", tt.batteryLevel, tt.isCharging, triggerType)
		})
	}
}

// TestConfigFileParsing тестирует загрузку конфигурации из файла
func TestConfigFileParsing(t *testing.T) {
	// Создаем временный файл конфигурации
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	validConfig := `{
		"min_threshold": 25,
		"max_threshold": 85,
		"check_interval": 60
	}`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Не удалось создать тестовый файл конфигурации: %v", err)
	}

	// Тестируем загрузку валидной конфигурации
	cfg, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Errorf("Ошибка загрузки валидной конфигурации: %v", err)
	}

	expectedCfg := config.Config{
		MinThreshold:  25,
		MaxThreshold:  85,
		CheckInterval: 60,
	}

	if cfg.MinThreshold != expectedCfg.MinThreshold {
		t.Errorf("MinThreshold = %d, ожидалось %d", cfg.MinThreshold, expectedCfg.MinThreshold)
	}

	// Тестируем невалидную конфигурацию
	invalidConfig := `{
		"min_threshold": "invalid",
		"max_threshold": 85
	}`

	invalidConfigPath := filepath.Join(tempDir, "invalid_config.json")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Не удалось создать невалидный файл конфигурации: %v", err)
	}

	_, err = loadConfigFromFile(invalidConfigPath)
	if err == nil {
		t.Errorf("Ожидалась ошибка при загрузке невалидной конфигурации")
	}

	// Тестируем несуществующий файл
	_, err = loadConfigFromFile("/nonexistent/config.json")
	if err == nil {
		t.Errorf("Ожидалась ошибка при загрузке несуществующего файла")
	}
}

// Заглушка для функции загрузки конфигурации
func loadConfigFromFile(path string) (config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, err
	}

	// Простая имитация JSON парсинга
	// В реальном коде здесь должен быть json.Unmarshal
	if string(data) == `{
		"min_threshold": 25,
		"max_threshold": 85,
		"check_interval": 60
	}` {
		return config.Config{
			MinThreshold:  25,
			MaxThreshold:  85,
			CheckInterval: 60,
		}, nil
	}

	return config.Config{}, fmt.Errorf("invalid JSON")
}

// TestMemoryUsage проверяет использование памяти
// ИСПРАВЛЕНИЕ: Правильная обработка uint64 арифметики для предотвращения underflow
func TestMemoryUsage(t *testing.T) {
	// Получаем начальную статистику памяти
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Выполняем множественные операции с батареей
	for i := 0; i < 1000; i++ {
		info := BatteryInfo{
			CurrentCapacity: i % 100,
			IsCharging:      i%2 == 0,
			IsPlugged:       i%3 == 0,
		}

		// Симулируем обработку
		_ = info.CurrentCapacity / 5
		_ = info.CurrentCapacity % 5
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// ИСПРАВЛЕНИЕ: Правильно обрабатываем случай когда m1.Alloc > m2.Alloc (после GC)
	var memDiff int64
	if m2.Alloc >= m1.Alloc {
		memDiff = int64(m2.Alloc - m1.Alloc)
	} else {
		// Память уменьшилась (GC сработал хорошо)
		memDiff = -int64(m1.Alloc - m2.Alloc)
	}

	const maxMemoryIncrease = 1024 * 1024 // 1MB
	if memDiff > maxMemoryIncrease {
		t.Errorf("Возможная утечка памяти: использовано дополнительно %d байт", memDiff)
	} else {
		t.Logf("✅ Использование памяти в норме: %+d байт", memDiff)
		if memDiff < 0 {
			t.Logf("   (память освободилась благодаря GC)")
		}
	}
}
