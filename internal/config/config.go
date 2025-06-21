package config

import (
	"encoding/json"
	"fmt"
	"macbat/internal/log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	// configInstance хранит закешированный экземпляр конфигурации
	configInstance *Config
	configOnce     sync.Once
	configErr      error
)

// NotificationState содержит состояние уведомлений
// @note Используется для сохранения состояния между перезапусками приложения
type NotificationState struct {
	LastTime     time.Time `json:"last_time"`     // Время последнего уведомления
	Count        int       `json:"count"`         // Количество отправленных уведомлений
	LastLevel    int       `json:"last_level"`    // Последний уровень заряда, при котором было отправлено уведомление
	LastCharging bool      `json:"last_charging"` // Последнее состояние зарядки
}

// Config содержит настройки приложения.
type Config struct {
	MinThreshold                 int                `json:"min_threshold"`                // Минимальный порог заряда (%)
	MaxThreshold                 int                `json:"max_threshold"`                // Максимальный порог заряда (%)
	CheckIntervalWhenCharging    int                `json:"check_interval_charging"`      // Интервал проверки при зарядке (секунды)
	CheckIntervalWhenDischarging int                `json:"check_interval_discharging"`   // Интервал проверки без зарядки (секунды)
	NotificationInterval         int                `json:"notification_interval"`        // Интервал уведомлений (минуты)
	MaxNotifications             int                `json:"max_notifications"`            // Максимум уведомлений подряд
	Debug                        bool               `json:"debug"`                        // Отображать DEBUG логи
	NotificationState            *NotificationState `json:"notification_state,omitempty"` // Состояние уведомлений
}

// DefaultConfig возвращает конфигурацию по умолчанию.
// @return Config - конфигурация по умолчанию
func DefaultConfig() Config {
	return Config{
		MinThreshold:                 21,
		MaxThreshold:                 81,
		CheckIntervalWhenCharging:    60,   // 1 минута при зарядке
		CheckIntervalWhenDischarging: 2280, // 38 минут при работе от батареи
		NotificationInterval:         5,
		MaxNotifications:             3,
		Debug:                        false,
		NotificationState: &NotificationState{
			LastTime:     time.Time{},
			Count:        0,
			LastLevel:    0,
			LastCharging: false,
		},
	}
}

// mergeWithDefaults объединяет загруженную конфигурацию с настройками по умолчанию.
// @param loaded *Config - загруженная конфигурация
// @return *Config - конфигурация с замененными нулевыми значениями
func mergeWithDefaults(loaded *Config) *Config {
	defaultCfg := DefaultConfig()

	if loaded.MinThreshold == 0 {
		log.Info(fmt.Sprintf("Установлен минимальный порог по умолчанию: %d%%", defaultCfg.MinThreshold))
		loaded.MinThreshold = defaultCfg.MinThreshold
	}
	if loaded.MaxThreshold == 0 {
		log.Info(fmt.Sprintf("Установлен максимальный порог по умолчанию: %d%%", defaultCfg.MaxThreshold))
		loaded.MaxThreshold = defaultCfg.MaxThreshold
	}
	if loaded.CheckIntervalWhenCharging == 0 {
		log.Info(fmt.Sprintf("Установлен интервал проверки при зарядке по умолчанию: %d сек", defaultCfg.CheckIntervalWhenCharging))
		loaded.CheckIntervalWhenCharging = defaultCfg.CheckIntervalWhenCharging
	}
	if loaded.CheckIntervalWhenDischarging == 0 {
		log.Info(fmt.Sprintf("Установлен интервал проверки при отключении зарядки по умолчанию: %d сек", defaultCfg.CheckIntervalWhenDischarging))
		loaded.CheckIntervalWhenDischarging = defaultCfg.CheckIntervalWhenDischarging
	}
	if loaded.NotificationInterval == 0 {
		log.Info(fmt.Sprintf("Установлен интервал уведомлений по умолчанию: %d мин", defaultCfg.NotificationInterval))
		loaded.NotificationInterval = defaultCfg.NotificationInterval
	}
	if loaded.MaxNotifications == 0 {
		log.Info(fmt.Sprintf("Установлено максимальное количество уведомлений по умолчанию: %d", defaultCfg.MaxNotifications))
		loaded.MaxNotifications = defaultCfg.MaxNotifications
	}
	if !loaded.Debug {
		log.Info(fmt.Sprintf("Установлено отображение DEBUG логов по умолчанию: %t", defaultCfg.Debug))
		loaded.Debug = defaultCfg.Debug
	}

	log.Debug("Конфигурация успешно обновлена.")
	return loaded
}

// InitConfig инициализирует конфигурацию при запуске приложения
// @return error - ошибка, если не удалось инициализировать конфигурацию
func InitConfig() error {
	_, err := LoadConfig()
	return err
}

// LoadConfig загружает конфигурацию из файла или создает новую, если файл не существует.
// Конфигурация загружается только один раз и кешируется.
// @return *Config - указатель на конфигурацию
// @return error - ошибка, если не удалось загрузить/создать конфигурацию
func LoadConfig() (*Config, error) {
	// Используем sync.Once для потокобезопасной инициализации
	configOnce.Do(func() {
		configInstance, configErr = loadConfigFromFile()
		if configErr != nil {
			configErr = fmt.Errorf("не удалось загрузить конфигурацию: %w", configErr)
		}
	})

	return configInstance, configErr
}

// loadConfigFromFile загружает конфигурацию из файла или создает новую
// @return *Config - указатель на конфигурацию
// @return error - ошибка, если не удалось загрузить/создать конфигурацию
func loadConfigFromFile() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("не удалось определить путь к конфигурации: %w", err)
	}

	// Если файл конфигурации не существует, создаем новый с настройками по умолчанию
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := saveConfigToFile(configPath, &config); err != nil {
			return nil, fmt.Errorf("не удалось сохранить конфигурацию по умолчанию: %w", err)
		}
		return &config, nil
	}

	// Читаем существующий файл конфигурации
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл конфигурации: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("ошибка при разборе файла конфигурации: %w", err)
	}

	// Убедимся, что состояние уведомлений инициализировано
	if config.NotificationState == nil {
		config.NotificationState = &NotificationState{}
	}

	// Объединяем с настройками по умолчанию
	mergedConfig := mergeWithDefaults(&config)

	// Сохраняем обновленную конфигурацию, если были применены значения по умолчанию
	updatedData, err := json.MarshalIndent(mergedConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации обновленной конфигурации: %w", err)
	}

	// Читаем текущее содержимое файла для сравнения
	currentData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения текущей конфигурации: %w", err)
	}

	// Сохраняем, если конфигурация изменилась
	if string(updatedData) != string(currentData) {
		log.Debug("Обнаружены изменения в конфигурации, сохраняем...")
		if err := saveConfigToFile(configPath, mergedConfig); err != nil {
			return nil, fmt.Errorf("не удалось сохранить обновленную конфигурацию: %w", err)
		}
		log.Debug("Конфигурация успешно обновлена")
	}

	return mergedConfig, nil
}

// getConfigPath возвращает путь к файлу конфигурации
// @return string - путь к файлу конфигурации
// @return error - ошибка, если не удалось определить путь
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("не удалось определить домашнюю директорию: %w", err)
	}

	appConfigDir := filepath.Join(homeDir, ".config", "macbat")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать директорию конфигурации: %w", err)
	}

	return filepath.Join(appConfigDir, "config.json"), nil
}

// saveConfigToFile сохраняет конфигурацию в файл
// @param configPath string - путь к файлу конфигурации
// @param cfg *Config - указатель на конфигурацию
// @return error - ошибка, если не удалось сохранить
func saveConfigToFile(configPath string, cfg *Config) error {
	tempFile := configPath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл конфигурации: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cfg); err != nil {
		os.Remove(tempFile) // Удаляем временный файл в случае ошибки
		return fmt.Errorf("ошибка при кодировании конфигурации: %w", err)
	}

	// Атомарная замена файла
	if err := os.Rename(tempFile, configPath); err != nil {
		os.Remove(tempFile) // Удаляем временный файл в случае ошибки
		return fmt.Errorf("не удалось сохранить конфигурацию: %w", err)
	}

	return nil
}

// SaveConfig сохраняет текущую конфигурацию в файл
// @param cfg *Config - указатель на конфигурацию для сохранения
// @return error - ошибка, если не удалось сохранить
func SaveConfig(cfg *Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("не удалось определить путь к конфигурации: %w", err)
	}
	return saveConfigToFile(configPath, cfg)
}

// UpdateConfig обновляет конфигурацию и сохраняет файл.
// @param key string - имя параметра
// @param value string - новое значение
// @return error - ошибка, если не удалось обновить или сохранить конфигурацию
func UpdateConfig(key string, value string, currentCapacity int, isCharging bool) error {
	// Загружаем текущую конфигурацию
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Обновляем значение в зависимости от ключа
	switch key {
	case "min":
		min, err := strconv.Atoi(value)
		if err != nil || min < 1 || min > 100 {
			return fmt.Errorf("некорректное значение для минимального порога: %s. Укажите число от 1 до 100", value)
		}
		config.MinThreshold = min

	case "max":
		max, err := strconv.Atoi(value)
		if err != nil || max < 1 || max > 100 {
			return fmt.Errorf("некорректное значение для максимального порога: %s. Укажите число от 1 до 100", value)
		}
		config.MaxThreshold = max

	case "check-interval-charging":
		interval, err := strconv.Atoi(value)
		if err != nil || interval < 1 {
			return fmt.Errorf("некорректное значение для интервала проверки при зарядке: %s. Укажите положительное число", value)
		}
		config.CheckIntervalWhenCharging = interval

	case "check-interval-discharging":
		interval, err := strconv.Atoi(value)
		if err != nil || interval < 1 {
			return fmt.Errorf("некорректное значение для интервала проверки при разрядке: %s. Укажите положительное число", value)
		}
		config.CheckIntervalWhenDischarging = interval

	case "interval":
		interval, err := strconv.Atoi(value)
		if err != nil || interval < 1 {
			return fmt.Errorf("некорректное значение для интервала уведомлений: %s. Укажите положительное число", value)
		}
		config.NotificationInterval = interval

	case "max-notifications":
		max, err := strconv.Atoi(value)
		if err != nil || max < 1 {
			return fmt.Errorf("некорректное значение для максимального количества уведомлений: %s. Укажите положительное число", value)
		}
		config.MaxNotifications = max

	default:
		return fmt.Errorf("неизвестный параметр конфигурации: %s", key)
	}

	// Сбрасываем счетчик уведомлений и обновляем состояние
	config.NotificationState.Count = 0
	config.NotificationState.LastTime = time.Time{}
	config.NotificationState.LastLevel = currentCapacity
	config.NotificationState.LastCharging = isCharging

	// Сохраняем обновленную конфигурацию
	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("не удалось сохранить конфигурацию: %w", err)
	}

	return nil
}

// UpdateConfigWithFunc обновляет конфигурацию с помощью функции-обновляльщика
// @param updater func(*Config) - функция для обновления конфигурации
// @return error - ошибка, если не удалось обновить или сохранить конфигурацию
func UpdateConfigWithFunc(updater func(*Config)) error {
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("не удалось загрузить конфигурацию: %w", err)
	}

	// Вызываем функцию обновления
	updater(config)

	// Сохраняем обновленную конфигурацию
	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("не удалось сохранить конфигурацию: %w", err)
	}

	return nil
}

// CheckWriteAccess проверяет права записи в директорию.
// @param path string - путь к директории
// @return error - ошибка, если нет прав записи
func CheckWriteAccess(path string) error {
	testFile := filepath.Join(path, ".test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("нет прав на запись в директорию %s: %w", path, err)
	}
	// Удаляем тестовый файл
	_ = os.Remove(testFile)
	return nil
}

// UpdateFullConfig обновляет конфигурацию через CLI контекст
// @param c *cli.Context - контекст CLI
// @param key string - ключ параметра
// @param value string - новое значение
// @param setInterval string - интервал для настройки
// @return error - ошибка, если не удалось обновить конфигурацию
func UpdateFullConfig(c *cli.Context, key, value, setInterval string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	switch key {
	case "check":
		// Извлекаем второй аргумент, следующий за 'check'
		param := strings.ToLower(value)

		// Извлекаем третий аргумент, следующий за 'up' или 'down'
		interval, err := strconv.Atoi(setInterval)
		if err != nil {
			return fmt.Errorf("значение должно быть числом")
		}
		if interval <= 0 {
			return fmt.Errorf("интервал проверки должен быть положительным числом")
		}

		if param == "up" {
			cfg.CheckIntervalWhenCharging = interval
		} else if param == "down" {
			cfg.CheckIntervalWhenDischarging = interval
		} else {
			return fmt.Errorf("значение должно быть 'up' или 'down'")
		}

	case "min":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("значение должно быть числом")
		}
		if val < 0 || val > 100 {
			return fmt.Errorf("минимальный порог должен быть от 0 до 100")
		}
		if val >= cfg.MaxThreshold {
			return fmt.Errorf("минимальный порог должен быть меньше максимального")
		}
		cfg.MinThreshold = val

	case "max":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("значение должно быть числом")
		}
		if val <= 0 || val > 100 {
			return fmt.Errorf("максимальный порог должен быть от 1 до 100")
		}
		if val <= cfg.MinThreshold {
			return fmt.Errorf("максимальный порог должен быть больше минимального")
		}
		cfg.MaxThreshold = val

	case "notification-interval":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("значение должно быть числом")
		}
		if val <= 0 {
			return fmt.Errorf("интервал уведомлений должен быть положительным числом")
		}
		cfg.NotificationInterval = val

	case "max-notifications":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("значение должно быть числом")
		}
		if val <= 0 {
			return fmt.Errorf("максимальное количество уведомлений должно быть положительным числом")
		}
		cfg.MaxNotifications = val

	default:
		return fmt.Errorf("неизвестный параметр конфигурации: %s", key)
	}

	// Сохраняем обновленную конфигурацию
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("не удалось сохранить конфигурацию: %w", err)
	}

	return nil
}

// reloadService перезагружает сервис
// @return error - ошибка, если не удалось перезагрузить сервис
// @return nil - если сервис успешно перезагружен
func reloadService() error {
	// Останавливаем сервис
	cmd := exec.Command("launchctl", "unload", "/Library/LaunchDaemons/com.macbat.plist")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("не удалось остановить сервис: %w", err)
	}

	// Запускаем сервис
	cmd = exec.Command("launchctl", "load", "/Library/LaunchDaemons/com.macbat.plist")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("не удалось запустить сервис: %w", err)
	}

	return nil
}

// updatePlistInterval обновляет значение StartInterval в plist файле
// @param plistContent string - содержимое plist файла
// @param interval int - новый интервал в секундах
// @return string - обновленное содержимое plist файла
func updatePlistInterval(plistContent string, interval int) string {
	// Регулярное выражение для поиска и замены значения StartInterval
	re := regexp.MustCompile(`<key>StartInterval</key>\s*<integer>\d+</integer>`)
	return re.ReplaceAllString(plistContent, fmt.Sprintf("<key>StartInterval</key>\n        <integer>%d</integer>", interval))
}
