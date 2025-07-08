package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"net/mail"

	"github.com/qzeleza/macbat/internal/commands"
	"github.com/qzeleza/macbat/internal/config"
	"github.com/qzeleza/macbat/internal/logger"
	"github.com/qzeleza/macbat/internal/paths"
	"github.com/qzeleza/macbat/internal/version"
	"github.com/urfave/cli/v3"
)

const (
	// debugMode определяет режим отладки
	debugMode = true

	// appName имя приложения
	appName = "macbat"

	// appUsage описание приложения
	appUsage = "Утилита для управления батареей macOS"
)

// App представляет основное приложение CLI
type App struct {
	cli        *cli.Command       // CLI приложение
	logger     *logger.Logger     // Логгер
	run        *commands.Commands // Команды
	cfg        *config.Config     // Конфигурация
	cfgManager *config.Manager    // Менеджер конфигурации
}

// init выполняет начальную инициализацию при импорте пакета
func init() {
	// Привязываем горутину к главному потоку ОС для GUI
	runtime.LockOSThread()
}

// NewApp создает и инициализирует новое приложение
func NewApp() (*App, error) {
	// Инициализация логгера
	log := logger.New(paths.LogPath(), 100, true, debugMode)

	// Выводим аргументы запуска для отладки
	log.Debug(fmt.Sprintf("Приложение запущено с аргументами: %s", strings.Join(os.Args, " ")))

	// Инициализация конфигурации
	cfgManager, err := config.New(log, paths.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации менеджера конфигурации: %w", err)
	}

	// Загрузка конфигурации
	conf, err := cfgManager.Load()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Инициализация команд
	commands := commands.NewCommands(log, conf)

	app := &App{
		logger:     log,        // логгер
		run:        commands,   // команды
		cfg:        conf,       // конфигурация
		cfgManager: cfgManager, // менеджер конфигурации
	}

	// Создаем CLI приложение
	app.cli = app.createCLI()

	// Устанавливаем глобальный логгер
	setGlobalLogger(log)

	return app, nil
}

// createCLI создает структуру CLI приложения
func (a *App) createCLI() *cli.Command {
	// Устанавливаем русские шаблоны
	setupRussianTemplates()

	return &cli.Command{
		Name:    appName,
		Usage:   appUsage,
		Version: version.GetVersion(),
		Authors: []any{
			mail.Address{Name: "qzeleza", Address: "support@qzeleza.com"},
		},
		Commands: []*cli.Command{
			a.installCommand(),
			a.uninstallCommand(),
			a.logCommand(),
			a.configCommand(),
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:   "background",
				Usage:  "Запускает фоновый процесс мониторинга (для опытных пользователей)",
				Hidden: true,
			},
			&cli.BoolFlag{
				Name:   "gui-agent",
				Usage:  "Запускает GUI агента",
				Hidden: true,
			},
		},
		Action: a.defaultAction,
		Before: a.beforeAction,
		After:  a.afterAction,
	}
}

// beforeAction выполняется перед любой командой
func (a *App) beforeAction(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	// Можно добавить общую инициализацию здесь
	a.logger.Debug("Начало выполнения команды")
	return ctx, nil
}

// afterAction выполняется после любой команды
func (a *App) afterAction(ctx context.Context, cmd *cli.Command) error {
	// Можно добавить общую очистку здесь
	a.logger.Debug("Завершение выполнения команды")
	return nil
}

// Run запускает приложение
func (a *App) Run(args []string) error {
	return a.cli.Run(context.Background(), args)
}

// Logger возвращает логгер приложения
func (a *App) Logger() *logger.Logger {
	return a.logger
}

// setGlobalLogger устанавливает глобальный логгер для обратной совместимости
var log *logger.Logger

func setGlobalLogger(l *logger.Logger) {
	log = l
}

// getLogger возвращает глобальный логгер
func getLogger() *logger.Logger {
	if log == nil {
		// Создаем логгер по умолчанию если не инициализирован
		log = logger.New(paths.LogPath(), 100, true, debugMode)
	}
	return log
}
