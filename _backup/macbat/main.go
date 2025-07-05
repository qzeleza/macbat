// Пакет main содержит точку входа приложения и настройку CLI.
//
// @package main
package main

import (
	// Встроенные пакеты
	"fmt"
	"os"
	"time"

	// Внешние зависимости
	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	// Внутренние пакеты
	"macbat/internal/config"
	"macbat/internal/log"
	"macbat/internal/paths"
	"macbat/internal/utils"
)

// Константы приложения
const (
	// version содержит версию приложения в формате SemVer (Major.Minor.Patch)
	version = "2.1.0"

	// DefaultWindowWidth стандартная ширина окна, если не удалось определить размер терминала
	DefaultWindowWidth = 63
	// MinWindowWidth минимальная ширина окна для читаемости
	MinWindowWidth = 50
)

// getWindowWidth возвращает ширину окна, рассчитанную как 5/9 от ширины терминала
// Минимальное значение - MinWindowWidth символов
func getWindowWidth() int {
	// Получаем размер терминала
	width := utils.GetTerminalWidth()
	// Вычисляем 5/9 от ширины, но не менее MinWindowWidth символов
	if width <= 0 {
		return DefaultWindowWidth // Значение по умолчанию, если не удалось определить ширину
	}
	calculated := int(float64(width) * 4 / 9)
	if calculated < MinWindowWidth {
		return MinWindowWidth
	}
	return calculated
}

// WindowWidth определяет ширину выводимого в терминале окна в символах
// Рассчитывается как 5/9 от доступной ширины терминала
// Минимальное значение - MinWindowWidth символов
var WindowWidth = getWindowWidth()

// box - буфер для форматированного вывода в терминале
// Используется для создания аккуратных текстовых блоков с границами
// и центрирования текста в окне терминала
var box = utils.NewWindowBuffer(WindowWidth)

// Точка входа в приложение.
// Инициализирует конфигурацию, настраивает обработчики сигналов и запускает CLI.
//
// @return void
// main является точкой входа в приложение.
// Инициализирует конфигурацию, настраивает обработчики сигналов и запускает CLI.
//
// @return void
func main() {
	// Если аргументы командной строки отсутствуют, выходим без сообщений
	// Это позволяет использовать приложение как CLI-утилиту без вывода справки
	if len(os.Args) == 1 {
		os.Exit(0)
	}

	// Инициализируем конфигурацию при запуске
	if err := config.InitConfig(); err != nil {
		log.Error("Ошибка при инициализации конфигурации: " + err.Error())
		os.Exit(1)
	}

	// Инициализация системы логирования
	// Логи будут записываться в файл, указанный в paths.LogPath()
	log.Info("")
	log.Info("Запуск приложения " + paths.AppName)

	// Настройка и запуск CLI приложения
	// Определение команд, флагов и их обработчиков
	app := &cli.App{
		Name:        cases.Title(language.Russian).String(paths.AppName),
		HelpName:    paths.AppName,
		Usage:       "для начала работы запустите команду " + paths.AppName + " install",
		Description: "утилита для мониторинга батареи macOS",
		Version:     version,
		Compiled:    time.Now(),
		Authors: []*cli.Author{
			{
				Name:  "Zeleza",
				Email: "zeleza@mail.ru",
			},
		},
		CustomAppHelpTemplate: RussianHelpTemplate,
		CommandNotFound: func(c *cli.Context, command string) {
			fmt.Fprintf(c.App.Writer, "Неизвестная команда: %q\n", command)
			cli.ShowAppHelp(c)
		},
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			if isSubcommand {
				return err
			}
			fmt.Fprintf(c.App.Writer, "Ошибка: %v\n\n", err)
			cli.ShowAppHelp(c)
			return nil
		},
		Commands: setupCommands(),
	}

	// Устанавливаем кастомные шаблоны справки
	cli.AppHelpTemplate = RussianHelpTemplate
	cli.CommandHelpTemplate = RussianCommandHelpTemplate
	cli.SubcommandHelpTemplate = RussianSubcommandHelpTemplate

	// Запуск приложения
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
