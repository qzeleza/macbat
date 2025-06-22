// Package config управляет конфигурацией приложения.
// Он предоставляет структуру Config и Manager для загрузки, сохранения
// и управления настройками из файла JSON. Модуль является самодостаточным
// и не зависит от глобального состояния.
package config

import (
	"encoding/json"
	"fmt"
	"macbat/internal/logger" // Предполагается, что у вас есть такой логгер
	"macbat/internal/paths"
	"os"
	"path/filepath"
	"time"
)

// NotificationState содержит состояние уведомлений.
// @note Используется для сохранения состояния между перезапусками приложения.
// @remark Эту структуру логичнее было бы перенести в пакет, управляющий состоянием (state),
// но пока оставляем здесь для совместимости.
type NotificationState struct {
	LastTime     time.Time `json:"last_time"`
	Count        int       `json:"count"`
	LastLevel    int       `json:"last_level"`
	LastCharging bool      `json:"last_charging"`
}

// Config содержит все настраиваемые параметры приложения.
type Config struct {
	MinThreshold                 int    `json:"min_threshold"`
	MaxThreshold                 int    `json:"max_threshold"`
	CheckIntervalWhenCharging    int    `json:"check_interval_charging"`
	CheckIntervalWhenDischarging int    `json:"check_interval_discharging"`
	NotificationInterval         int    `json:"notification_interval"`
	MaxNotifications             int    `json:"max_notifications"`
	DebugEnabled                 bool   `json:"debug_enabled"`
	LogFilePath                  string `json:"log_file_path"`
	LogRotationLines             int    `json:"log_rotation_lines"`
	UseSimulator                 bool   `json:"use_simulator"`
	LogEnabled                   bool   `json:"log_enabled"`
}

// Manager инкапсулирует всю логику управления конфигурацией.
// Это основная структура модуля, заменяющая глобальные переменные и синглтоны.
type Manager struct {
	configPath string
	log        *logger.Logger
}

// New создает новый экземпляр менеджера конфигурации.
// @param log *logger.Logger - экземпляр логгера.
// @param customPath ...string - необязательный пользовательский путь к файлу конфигурации.
// Если путь не указан, будет использован путь по умолчанию (~/.config/macbat/config.json).
// @return *Manager - указатель на новый экземпляр менеджера.
// @return error - ошибка, если не удалось определить путь или создать директорию.
func New(log *logger.Logger, customPath ...string) (*Manager, error) {
	var configPath string

	if len(customPath) > 0 && customPath[0] != "" {
		configPath = customPath[0]
	} else {
		configPath = paths.ConfigPath()
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию для конфигурации %s: %w", configDir, err)
	}

	return &Manager{
		configPath: configPath,
		log:        log,
	}, nil
}

// Default возвращает указатель на структуру Config с настройками по умолчанию.
func Default() *Config {
	return &Config{
		MinThreshold:                 21,
		MaxThreshold:                 81,
		NotificationInterval:         1800, // ИЗМЕНЕНИЕ: 30 минут = 1800 секунд
		MaxNotifications:             3,
		LogFilePath:                  paths.LogPath(),
		LogRotationLines:             1000,
		CheckIntervalWhenCharging:    30,   // ИЗМЕНЕНИЕ: 30 секунд
		CheckIntervalWhenDischarging: 1800, // ИЗМЕНЕНИЕ: 30 минут = 1800 секунд
		UseSimulator:                 false,
		LogEnabled:                   true,
		DebugEnabled:                 false,
	}
}

// ConfigPath возвращает путь к файлу конфигурации, которым управляет менеджер.
func (m *Manager) ConfigPath() string {
	return m.configPath
}

// Load загружает конфигурацию из файла.
// Если файл не существует, он будет создан с настройками по умолчанию.
// Если в файле отсутствуют какие-либо поля (равны "нулевому" значению),
// они будут заполнены значениями по умолчанию.
// @return *Config - указатель на загруженную и проверенную конфигурацию.
// @return error - ошибка, если не удалось прочитать или разобрать файл.
func (m *Manager) Load() (*Config, error) {
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		m.log.Info("Файл конфигурации не найден. Создание нового с настройками по умолчанию.")
		defaultCfg := Default()
		if err := m.Save(defaultCfg); err != nil {
			return nil, fmt.Errorf("не удалось сохранить конфигурацию по умолчанию: %w", err)
		}
		return defaultCfg, nil
	}

	// ИЗМЕНЕНИЕ: Читаем файл один раз в память для последующего двойного разбора.
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %w", err)
	}

	// Шаг 1: Разбираем JSON в общую карту, чтобы определить присутствующие ключи.
	presenceMap := make(map[string]interface{})
	if err := json.Unmarshal(data, &presenceMap); err != nil {
		return nil, fmt.Errorf("ошибка при первичном разборе файла конфигурации (в карту): %w", err)
	}

	// Шаг 2: Разбираем тот же JSON в строго типизированную структуру.
	var loadedCfg Config
	if err := json.Unmarshal(data, &loadedCfg); err != nil {
		return nil, fmt.Errorf("ошибка при вторичном разборе файла конфигурации (в структуру): %w", err)
	}

	// Шаг 3: Объединяем конфигурацию, используя карту присутствия ключей.
	finalCfg, wasModified := m.mergeWithDefaults(&loadedCfg, presenceMap)
	if wasModified {
		m.log.Info("Конфигурация была дополнена значениями по умолчанию. Сохраняем изменения...")
		if err := m.Save(finalCfg); err != nil {
			m.log.Debug(fmt.Sprintf("Не удалось автоматически сохранить дополненную конфигурацию: %v", err))
		}
	}

	return finalCfg, nil
}

// Save атомарно сохраняет предоставленную конфигурацию в файл.
// Использует временный файл и переименование для безопасности записи.
// @param cfg *Config - указатель на конфигурацию для сохранения.
// @return error - ошибка, если не удалось записать или переименовать файл.
func (m *Manager) Save(cfg *Config) error {
	tempFile := m.configPath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл конфигурации: %w", err)
	}
	// `defer os.Remove(tempFile)` удалит временный файл в любом случае:
	// и при успешном переименовании, и при ошибке.
	defer os.Remove(tempFile)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Для читаемого формата JSON

	if err := encoder.Encode(cfg); err != nil {
		file.Close() // Закрываем перед удалением
		return fmt.Errorf("ошибка при кодировании конфигурации: %w", err)
	}
	// Важно закрыть файл перед переименованием, особенно в Windows.
	file.Close()

	// Атомарная замена файла.
	if err := os.Rename(tempFile, m.configPath); err != nil {
		return fmt.Errorf("не удалось сохранить конфигурацию: %w", err)
	}

	m.log.Debug("Конфигурация успешно сохранена.")
	return nil
}

// mergeWithDefaults проверяет нулевые значения в загруженной конфигурации и заменяет их
// значениями по умолчанию. Возвращает итоговую конфигурацию и флаг, были ли внесены изменения.
func (m *Manager) mergeWithDefaults(loaded *Config, presenceMap map[string]interface{}) (finalCfg *Config, wasModified bool) {
	defaultCfg := Default()
	changesMade := false

	// Функция-помощник для проверки наличия ключа в JSON
	keyExists := func(key string) bool {
		_, ok := presenceMap[key]
		return ok
	}

	// Применяем значения по умолчанию, только если ключ ОТСУТСТВУЕТ в файле.
	if !keyExists("min_threshold") {
		m.log.Debug(fmt.Sprintf("Поле 'min_threshold' отсутствует. Установлено значение по умолчанию: %d", defaultCfg.MinThreshold))
		loaded.MinThreshold = defaultCfg.MinThreshold
		changesMade = true
	}
	if !keyExists("max_threshold") {
		m.log.Debug(fmt.Sprintf("Поле 'max_threshold' отсутствует. Установлено значение по умолчанию: %d", defaultCfg.MaxThreshold))
		loaded.MaxThreshold = defaultCfg.MaxThreshold
		changesMade = true
	}
	if !keyExists("check_interval_charging") {
		m.log.Debug(fmt.Sprintf("Поле 'check_interval_charging' отсутствует. Установлено значение по умолчанию: %v", defaultCfg.CheckIntervalWhenCharging))
		loaded.CheckIntervalWhenCharging = defaultCfg.CheckIntervalWhenCharging
		changesMade = true
	}
	if !keyExists("check_interval_discharging") {
		m.log.Debug(fmt.Sprintf("Поле 'check_interval_discharging' отсутствует. Установлено значение по умолчанию: %v", defaultCfg.CheckIntervalWhenDischarging))
		loaded.CheckIntervalWhenDischarging = defaultCfg.CheckIntervalWhenDischarging
		changesMade = true
	}
	if !keyExists("notification_interval") {
		m.log.Debug(fmt.Sprintf("Поле 'notification_interval' отсутствует. Установлено значение по умолчанию: %v", defaultCfg.NotificationInterval))
		loaded.NotificationInterval = defaultCfg.NotificationInterval
		changesMade = true
	}
	if !keyExists("max_notifications") {
		m.log.Debug(fmt.Sprintf("Поле 'max_notifications' отсутствует. Установлено значение по умолчанию: %d", defaultCfg.MaxNotifications))
		loaded.MaxNotifications = defaultCfg.MaxNotifications
		changesMade = true
	}
	if !keyExists("debug_enabled") {
		m.log.Debug(fmt.Sprintf("Поле 'debug_enabled' отсутствует. Установлено значение по умолчанию: %t", defaultCfg.DebugEnabled))
		loaded.DebugEnabled = defaultCfg.DebugEnabled
		changesMade = true
	}
	if !keyExists("log_file_path") {
		m.log.Debug(fmt.Sprintf("Поле 'log_file_path' отсутствует. Установлено значение по умолчанию: %s", defaultCfg.LogFilePath))
		loaded.LogFilePath = defaultCfg.LogFilePath
		changesMade = true
	}
	if !keyExists("log_rotation_lines") {
		m.log.Debug(fmt.Sprintf("Поле 'log_rotation_lines' отсутствует. Установлено значение по умолчанию: %d", defaultCfg.LogRotationLines))
		loaded.LogRotationLines = defaultCfg.LogRotationLines
		changesMade = true
	}
	if !keyExists("use_simulator") {
		m.log.Debug(fmt.Sprintf("Поле 'use_simulator' отсутствует. Установлено значение по умолчанию: %t", defaultCfg.UseSimulator))
		loaded.UseSimulator = defaultCfg.UseSimulator
		changesMade = true
	}
	if !keyExists("log_enabled") {
		m.log.Debug(fmt.Sprintf("Поле 'log_enabled' отсутствует. Установлено значение по умолчанию: %t", defaultCfg.LogEnabled))
		loaded.LogEnabled = defaultCfg.LogEnabled
		changesMade = true
	}

	return loaded, changesMade
}
