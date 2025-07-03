// Пакет main содержит настройку и обработку командной строки приложения macbat.
//
// Основные возможности:
// - Управление агентом мониторинга батареи
// - Настройка параметров мониторинга
// - Просмотр состояния батареи и логов
//
// @package main
package main

import (
	"macbat/internal/battery"
	"macbat/internal/config"
	"macbat/internal/log"

	"github.com/urfave/cli/v2"
)

// setupCommands настраивает команды и флаги CLI.
//
// @return []*cli.Command - список команд для CLI приложения.
//
// Команды:
//   - check: Проверяет состояние батареи и отправляет уведомления при необходимости.
//   - version: Выводит версию приложения.
//   - install: Устанавливает приложение как агент launchd.
//   - uninstall: Удаляет приложение из агентов launchd.
//   - status: Показывает статус агента launchd.
//   - start: Запускает агент launchd.
//   - stop: Останавливает агент launchd.
//   - restart: Перезапускает агент launchd.
//   - log: Выводит содержимое лог-файла.
//   - config: Выводит текущую конфигурацию.
//   - help: Показывает справку по командам.
func setupCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:   "install",
			Usage:  "Устанавливает приложение как агент launchd",
			Action: InstallCommand,
		},
		{
			Name:    "uninstall",
			Aliases: []string{"remove", "delete"},
			Usage:   "Удаляет приложение из агентов launchd",
			Action:  UninstallCommand,
		},
		{
			Name:   "status",
			Usage:  "Показывает статус агента launchd",
			Action: StatusCommand,
		},
		{
			Name:    "load",
			Aliases: []string{"start", "run"},
			Usage:   "Запускает агент launchd",
			Action:  LoadCommand,
		},
		{
			Name:    "check",
			Aliases: []string{"battery", "bat"},
			Usage:   "Проверяет состояние батареи",
			Action:  CheckCommand,
		},
		{
			Name:    "unload",
			Aliases: []string{"stop"},
			Usage:   "Останавливает агент launchd",
			Action:  UnloadCommand,
		},
		{
			Name:    "reload",
			Aliases: []string{"restart"},
			Usage:   "Перезапускает агент launchd",
			Action:  ReloadCommand,
		},
		{
			Name:      "logs",
			Aliases:   []string{"log"},
			Usage:     "Выводит логи (опционально: debug, info, error, check)",
			ArgsUsage: "[уровень]",
			Action:    LogsCommand,
		},
		{
			Name:    "config",
			Aliases: []string{"cfg", "conf"},
			Usage:   "Выводит текущую конфигурацию",
			Action:  ConfigCommand,
		},
		{
			Name:    "version",
			Aliases: []string{"ver", "v"},
			Usage:   "Выводит версию приложения",
			Action:  VersionCommand,
		},
		{
			Name:               "set",
			Aliases:            []string{"update"},
			CustomHelpTemplate: RussianSubcommandHelpTemplate,
			Usage:              "Устанавливает параметры конфигурации",
			Subcommands:        SetCommands(),
		},
	}
}

// SetCommands возвращает команды для установки параметров конфигурации.
//
// Команды:
//   - min: Установка минимального порога заряда батареи
//   - max: Установка максимального порога заряда батареи
//   - check-interval-charging: Настройка интервала проверки состояния батареи при зарядке
//   - check-interval-discharging: Настройка интервала проверки состояния батареи при разрядке
//   - interval: Настройка интервала уведомлений
//   - maxnote: Установка максимального количества уведомлений
//
// Пример использования:
//
//	macbat set min 20
//	macbat set max 80
//	macbat set check-interval-charging 60
//	macbat set check-interval-discharging 60
//	macbat set interval 5
//	macbat set maxnote 3
func SetCommands() []*cli.Command {
	info, err := battery.GetInfo()
	if err != nil {
		log.Error("Ошибка при получении информации о батарее: " + err.Error())
		return []*cli.Command{}
	}
	return []*cli.Command{
		{
			Name:      "min",
			Usage:     "Устанавливает минимальный порог заряда (%)",
			ArgsUsage: "<значение>",
			Action: func(c *cli.Context) error {
				if err := config.UpdateConfig("min-threshold", c.Args().First(), info.CurrentCapacity, info.IsCharging); err != nil {
					log.Error("Ошибка при обновлении конфигурации: " + err.Error())
					return err
				}
				log.Info("Минимальный порог заряда установлен на " + c.Args().First())
				box.AddLine("Минимальный порог заряда установлен на "+c.Args().First(), "", "")
				box.PrintBox()
				return nil
			},
		},
		{
			Name:      "max",
			Usage:     "Устанавливает максимальный порог заряда (%)",
			ArgsUsage: "<значение>",
			Action: func(c *cli.Context) error {
				if err := config.UpdateConfig("max-threshold", c.Args().First(), info.CurrentCapacity, info.IsCharging); err != nil {
					log.Error("Ошибка при обновлении конфигурации: " + err.Error())
					return err
				}
				log.Info("Максимальный порог заряда установлен на " + c.Args().First())
				box.AddLine("Максимальный порог заряда установлен на "+c.Args().First(), "", "")
				box.PrintBox()
				return nil
			},
		},
		{
			Name:      "check-interval-charging",
			Usage:     "Устанавливает интервал проверки (секунды)",
			ArgsUsage: "<значение>",
			Action: func(c *cli.Context) error {

				// Устанавливаем интервал проверки для состояния зарядки
				if err := config.UpdateConfig("check-interval-charging", c.Args().First(), info.CurrentCapacity, info.IsCharging); err != nil {
					log.Error("Ошибка при обновлении интервала проверки при зарядке: " + err.Error())
					return err
				}
				log.Info("Интервал проверки при зарядке установлен на " + c.Args().First())
				box.AddLine("Интервал проверки при зарядке установлен на "+c.Args().First(), "", "")
				box.PrintBox()
				return nil
			},
		},
		{
			Name:      "check-interval-discharging",
			Usage:     "Устанавливает интервал проверки (секунды)",
			ArgsUsage: "<значение>",
			Action: func(c *cli.Context) error {

				// Устанавливаем интервал проверки для состояния разрядки
				if err := config.UpdateConfig("check-interval-discharging", c.Args().First(), info.CurrentCapacity, info.IsCharging); err != nil {
					log.Error("Ошибка при обновлении интервала проверки при разрядке: " + err.Error())
					return err
				}
				log.Info("Интервал проверки при разрядке установлен на " + c.Args().First())
				box.AddLine("Интервал проверки при разрядке установлен на "+c.Args().First(), "", "")
				box.PrintBox()
				return nil
			},
		},
		{
			Name:      "interval",
			Usage:     "Устанавливает интервал уведомлений (минуты)",
			ArgsUsage: "<значение>",
			Action: func(c *cli.Context) error {
				if err := config.UpdateConfig("notification-interval", c.Args().First(), info.CurrentCapacity, info.IsCharging); err != nil {
					log.Error("Ошибка при обновлении интервала уведомлений: " + err.Error())
					return err
				}
				log.Info("Интервал уведомлений установлен на " + c.Args().First())
				box.AddLine("Интервал уведомлений установлен на "+c.Args().First(), "", "")
				box.PrintBox()
				return nil
			},
		},
		{
			Name:      "maxnote",
			Usage:     "Устанавливает максимум уведомлений подряд",
			ArgsUsage: "<значение>",
			Action: func(c *cli.Context) error {
				if err := config.UpdateConfig("max-notifications", c.Args().First(), info.CurrentCapacity, info.IsCharging); err != nil {
					log.Error("Ошибка при обновлении максимального количества уведомлений: " + err.Error())
					return err
				}
				log.Info("Максимум уведомлений подряд установлен на " + c.Args().First())
				box.AddLine("Максимум уведомлений подряд установлен на "+c.Args().First(), "", "")
				box.PrintBox()
				return nil
			},
		},
	}
}
