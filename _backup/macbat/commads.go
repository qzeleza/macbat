package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/log"
	"macbat/internal/paths"
	"macbat/internal/service"
	"macbat/internal/utils"
)

// CheckCommand обрабатывает команду проверки состояния батареи.
//
// Выполняет следующие действия:
// 1. Получает информацию о батарее
// 2. Проверяет текущий уровень заряда
// 3. Отправляет уведомление при необходимости
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае неудачи
//
// Пример использования:
//
//	if err := CheckCommand(c); err != nil {
//	    log.Fatalf("Ошибка проверки состояния батареи: %v", err)
//	}
//
// Примечание: Требует прав администратора для выполнения
func CheckCommand(c *cli.Context) error {
	log.Check("Запуск команды проверки состояния батареи")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Ошибка загрузки конфигурации: " + err.Error())
		return err
	}

	// Проверяем состояние батареи и триггеры срабатывания уведомлений
	battery.ObserveChanges(cfg)

	successMsg := "Проверка состояния батареи завершена успешно!"
	log.Check(successMsg)
	return nil
}

// InstallCommand обрабатывает команду установки приложения как агента launchd.
//
// Выполняет следующие действия:
// 1. Устанавливает исполняемый файл в системную директорию
// 2. Создает конфигурационный файл с настройками по умолчанию
// 3. Регистрирует агент в launchd
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае неудачной установки
//
// Пример использования:
//
//	macbat install
//
// Примечание: Требует прав администратора для выполнения
func InstallCommand(c *cli.Context) error {

	log.Info("Запуск команды установки")

	if err := service.Install(); err != nil {
		errMsg := fmt.Sprintf("Ошибка установки: %v", err)
		log.Error(errMsg)
		box.AddLine(errMsg, "", "")
		box.PrintBox()
		return cli.Exit("", 1)
	}

	successMsg := "Установка завершена успешно!"
	log.Info(successMsg)
	box.AddLine(successMsg, "", utils.ColorGreen)
	box.AddDivider() // Пустая строка для разделения

	infoMsg := "Проверьте, что приложение доступно из командной строки, выполнив:"
	box.AddLine(infoMsg, "", "")
	box.AddLine("  "+paths.AppName+" version", "", "")
	box.AddDivider() // Пустая строка для разделения

	helpMsg := "Если команда " + paths.AppName + " не найдена, выполните вручную:"
	box.AddLine(helpMsg, "", utils.ColorYellow)
	box.AddLine("  source ~/.zshrc", "", "")
	box.AddLine("или, если используете bash", "", "")
	box.AddLine("  source ~/.bash_profile", "", "")

	box.PrintBox()
	return nil
}

// UninstallCommand обрабатывает команду удаления приложения.
//
// Выполняет следующие действия:
// 1. Останавливает и выгружает агент из launchd
// 2. Удаляет файлы конфигурации и логов
// 3. Удаляет исполняемый файл из системной директории
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае неудачного удаления
//
// Пример использования:
//
//	macbat uninstall
//
// Примечание: Требует прав администратора для выполнения
func UninstallCommand(c *cli.Context) error {
	box.AddLine("Удаление "+paths.AppName+"...", "", "")
	box.AddDivider() // Пустая строка для разделения

	if err := service.Uninstall(); err != nil {
		box.AddLine(fmt.Sprintf("Ошибка удаления: %v", err), "", "")
		box.PrintBox()
		return cli.Exit("", 1)
	}

	box.AddLine("Удаление завершено успешно!", "", utils.ColorGreen)
	box.AddDivider() // Пустая строка для разделения
	box.AddLine("Если команда "+paths.AppName+" всё ещё доступна, обновите PATH вручную:", "", utils.ColorYellow)
	box.AddLine("source ~/.zshrc  # или source ~/.bash_profile, если используете bash", "", "")
	box.PrintBox()
	return nil
}

// LoadCommand загружает и активирует агент в launchd.
//
// Выполняет следующие действия:
// 1. Проверяет, что конфигурационный файл существует
// 2. Загружает plist-файл агента в launchd
// 3. Активирует агент для автоматического запуска
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае неудачной загрузки
//
// Пример использования:
//
//	macbat load
//
// Примечания:
// - Требует предварительной установки приложения
// - Требует прав администратора
func LoadCommand(c *cli.Context) error {
	box.AddLine("Загрузка агента...", "", "")
	box.AddDivider() // Пустая строка для разделения

	state, err := service.Load()
	if err != nil {
		if state {
			box.AddLine("Агент уже загружен", "", utils.ColorYellow)
		} else {
			box.AddLine(fmt.Sprintf("Ошибка загрузки агента: %v", err), "", utils.ColorRed)
			box.PrintBox()
			return cli.Exit("", 1)
		}
	} else {
		box.AddLine("Агент успешно загружен", "", utils.ColorGreen)
	}

	box.PrintBox()
	return nil
}

// UnloadCommand останавливает и выгружает агент из launchd.
//
// Выполняет следующие действия:
// 1. Останавливает выполнение запущенного агента
// 2. Выгружает конфигурацию агента из launchd
// 3. Отключает автозапуск агента при загрузке системы
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае неудачной выгрузки
//
// Пример использования:
//
//	macbat unload
//
// Примечания:
// - Требует прав администратора
// - Не удаляет файлы приложения, только останавливает его работу
func UnloadCommand(c *cli.Context) error {
	box.AddLine("Выгрузка агента...", "", "")
	box.AddDivider() // Пустая строка для разделения

	state, err := service.Unload()
	if err != nil {
		if state {
			box.AddLine("Агент уже выгружен", "", utils.ColorYellow)
		} else {
			box.AddLine(fmt.Sprintf("Ошибка выгрузки агента: %v", err), "", utils.ColorRed)
			box.PrintBox()
			return cli.Exit("", 1)
		}
	} else {
		box.AddLine("Агент успешно выгружен", "", utils.ColorGreen)
	}

	box.PrintBox()
	return nil
}

// ReloadCommand перезагружает конфигурацию агента.
//
// Выполняет следующие действия:
// 1. Останавливает работающий агент
// 2. Загружает обновленную конфигурацию
// 3. Запускает агент с новыми настройками
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае неудачной перезагрузки
//
// Пример использования:
//
//	macbat reload
//
// Примечания:
// - Полезно после изменения конфигурации
// - Требует прав администратора
// - Сохраняет текущее состояние агента (включен/выключен)
func ReloadCommand(c *cli.Context) error {
	box.AddLine("Перезагрузка агента...", "", "")
	box.AddDivider() // Пустая строка для разделения

	state, err := service.Reload()
	if err != nil {
		box.AddLine(fmt.Sprintf("Ошибка перезагрузки агента: %v", err), "", utils.ColorRed)
		box.PrintBox()
		return cli.Exit("", 1)
	}

	if state {
		box.AddLine("Агент успешно перезагружен", "", utils.ColorGreen)
	} else {
		box.AddLine("Не удалось перезагрузить агент", "", utils.ColorYellow)
	}

	box.PrintBox()
	return nil
}

// ConfigCommand выводит текущую конфигурацию приложения.
//
// Отображает следующие параметры:
// - Минимальный и максимальный уровень заряда батареи
// - Интервалы проверки и уведомлений
// - Текущий статус агента
// - Пути к файлам конфигурации и логов
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае проблем с чтением конфигурации
//
// Пример использования:
//
//	macbat config
//
// Примечание:
// - Не требует прав администратора
// - Читает конфигурацию из стандартного расположения
func ConfigCommand(c *cli.Context) error {

	box.AddLine("Текущая конфигурация:", "", "")
	box.AddDivider()

	cfg, err := config.LoadConfig()
	if err != nil {
		box.AddLine(fmt.Sprintf("Ошибка загрузки конфигурации: %v", err), "", utils.ColorRed)
		box.PrintBox()
		return cli.Exit("", 1)
	}

	box.AddLine("Минимальный порог заряда:", fmt.Sprintf("%d%%", cfg.MinThreshold), utils.ColorYellow)
	box.AddLine("Максимальный порог заряда:", fmt.Sprintf("%d%%", cfg.MaxThreshold), utils.ColorGreen)
	box.AddDivider()
	box.AddLine("Интервал проверки при зарядке:", fmt.Sprintf("%d сек", cfg.CheckIntervalWhenCharging), utils.ColorGreen)
	box.AddLine("Интервал проверки при разрядке:", fmt.Sprintf("%d сек", cfg.CheckIntervalWhenDischarging), utils.ColorBold)
	box.AddDivider()
	box.AddLine("Интервал уведомлений:", fmt.Sprintf("%d мин", cfg.NotificationInterval), "")
	box.AddLine("Макс. уведомлений:", fmt.Sprintf("%d", cfg.MaxNotifications), "")
	box.PrintBox()
	return nil
}

// StatusCommand отображает текущее состояние приложения и батареи.
//
// Показывает:
// - Статус работы агента (запущен/остановлен)
// - Текущий заряд батареи и состояние питания
// - Статус зарядки (заряжается/разряжается/полностью заряжена)
// - Время до разрядки/зарядки (если доступно)
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку в случае проблем с получением статуса
//
// Пример использования:
//
//	macbat status
//
// Примечания:
// - Не требует прав администратора
// - Для получения точных данных о батарее требуются соответствующие разрешения
func StatusCommand(c *cli.Context) error {
	var cfg, err = config.LoadConfig()
	if err != nil {
		box.AddLine("Ошибка загрузки конфигурации: "+err.Error(), "", utils.ColorRed)
		box.PrintBox()
		os.Exit(1)
	}

	info, err := battery.GetInfo()
	if err != nil {
		box.AddLine("Ошибка получения данных батареи: "+err.Error(), "", utils.ColorRed)
		box.PrintBox()
		return cli.Exit("", 1)
	}

	// Статус агента
	agentColor := utils.ColorRed
	agentStatus := "не загружен"
	if service.IsAgentRunning() {
		agentStatus = "загружен"
		agentColor = utils.ColorGreen
	}

	// Статус зарядки
	chargeColor := utils.ColorBold
	chargeStatus := "отключена"
	if info.IsCharging {
		chargeStatus = "подключена"
		chargeColor = utils.ColorGreen
	}

	// Выводим основную информацию
	box.AddLine("Статус агента", agentStatus, agentColor)
	box.AddDivider()
	// Информация о батарее
	box.AddLine("Зарядка", chargeStatus, chargeColor)
	box.AddDivider()

	// Параметры конфигурации
	box.AddLine("Минимальный порог заряда", fmt.Sprintf("%d%%", cfg.MinThreshold), utils.ColorYellow)
	box.AddLine("Текущий уровень заряда", fmt.Sprintf("%d%%", info.CurrentCapacity), utils.ColorWhite)
	box.AddLine("Максимальный порог заряда", fmt.Sprintf("%d%%", cfg.MaxThreshold), utils.ColorGreen)
	box.AddDivider()
	box.AddLine("Интервал проверки при зарядке", fmt.Sprintf("%d сек", cfg.CheckIntervalWhenCharging), utils.ColorGreen)
	box.AddLine("Интервал проверки при разрядке", fmt.Sprintf("%d сек", cfg.CheckIntervalWhenDischarging), utils.ColorBold)
	box.AddDivider()
	box.AddLine("Интервал повтора уведомлений", fmt.Sprintf("%d мин", cfg.NotificationInterval), "")
	box.AddLine("Макс. количество уведомлений", fmt.Sprintf("%d шт", cfg.MaxNotifications), "")

	// Дополнительная информация о батарее
	if info.MaxCapacity > 0 || info.DesignCapacity > 0 || info.CycleCount > 0 {
		box.AddDivider()
		if info.MaxCapacity > 0 && info.DesignCapacity > 0 {
			degradation := 100 - int(float64(info.MaxCapacity)*100/float64(info.DesignCapacity))
			degradationColor := utils.ColorGreen

			if degradation < 60 {
				degradationColor = utils.ColorRed
			} else if degradation < 80 {
				degradationColor = utils.ColorYellow
			}

			healthColor := utils.ColorGreen
			if info.HealthPercent < 60 {
				healthColor = utils.ColorRed
			} else if info.HealthPercent < 80 {
				healthColor = utils.ColorYellow
			}

			box.AddLine("Уровень деградации", fmt.Sprintf("%d%%", degradation), degradationColor)
			box.AddLine("Уровень здоровья", fmt.Sprintf("%d%%", info.HealthPercent), healthColor)
			box.AddDivider()
		}

		if info.DesignCapacity > 0 {
			box.AddLine("Исходная ёмкость", fmt.Sprintf("%d mAh", info.DesignCapacity), "")
		}

		cycleColor := utils.ColorBold
		if info.CycleCount < 500 {
			cycleColor = utils.ColorGreen
		} else if info.CycleCount < 800 {
			cycleColor = utils.ColorYellow
		}

		if info.CycleCount > 0 {
			box.AddLine("Циклов заряда", fmt.Sprintf("%d шт", info.CycleCount), cycleColor)
		}
	}

	box.PrintBox()
	return nil
}

// VersionCommand отображает информацию о версии приложения.
//
// Выводит:
// - Номер версии
// - Дату сборки
// - Информацию о лицензии
//
// @param c *cli.Context Контекст выполнения команды
// @return error Всегда возвращает nil
//
// Пример использования:
//
//	macbat version
//	macbat -v
//
// Примечание: Не требует дополнительных привилегий
func VersionCommand(c *cli.Context) error {
	box.AddLine(paths.AppName, "", "")
	box.AddLine("Версия:", version, "")
	box.AddLine("Дата сборки:", c.App.Compiled.Format("2006-01-02 15:04:05"), "")
	box.AddLine("Лицензия:", "MIT", "")
	box.PrintBox()
	return nil
}

// LogsCommand отображает содержимое лог-файла приложения с возможностью фильтрации по уровням.
//
// @param c *cli.Context Контекст выполнения команды
// @return error Возвращает ошибку если не удалось прочитать лог-файл
//
// Примеры использования:
//
//	macbat log           # Выводит все логи
//	macbat log error    # Выводит только ошибки
//	macbat log debug    # Выводит отладочные сообщения
//	macbat log info     # Выводит информационные сообщения
//
// Примечания:
// - Лог-файл находится в стандартном расположении для логов macOS
// - Для просмотра логов в реальном времени используйте команду `tail -f`
func LogsCommand(c *cli.Context) error {
	// Получаем уровень логирования из аргумента (если указан)
	level := ""
	if c.Args().Present() {
		level = strings.ToLower(c.Args().First())
		validLevels := map[string]bool{"debug": true, "info": true, "error": true, "check": true}
		if !validLevels[level] {
			return cli.Exit("Некорректный уровень логирования.\nДопустимые значения: debug, info, error, check", 1)
		}
	}

	// Читаем логи с фильтрацией по уровню
	logs, err := service.ReadLogs(level)
	if err != nil {
		fmt.Println("Ошибка чтения логов:", err)
		return cli.Exit("", 1)
	}

	// Выводим логи
	if len(logs) == 0 {
		levelMsg := ""
		if level != "" {
			levelMsg = fmt.Sprintf(" уровня %s ", strings.ToUpper(level))
		}
		fmt.Printf("Логи%s отсутствуют\n", levelMsg)
	} else {
		// Выводим последние 100 строк, если логов много
		const maxLines = 100
		start := 0
		if len(logs) > maxLines {
			start = len(logs) - maxLines
			fmt.Printf("Показаны последние %d из %d строк логов.\n\n", maxLines, len(logs))
		}
		for i := start; i < len(logs); i++ {
			fmt.Println(logs[i])
		}
	}

	return nil
}
