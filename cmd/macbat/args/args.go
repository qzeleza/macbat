package args

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qzeleza/macbat/internal/background"
	"github.com/qzeleza/macbat/internal/config"
	"github.com/qzeleza/macbat/internal/logger"
	"github.com/qzeleza/macbat/internal/paths"
	"github.com/qzeleza/macbat/internal/version"

	cli "github.com/urfave/cli/v3"
)

// Dependencies содержит все зависимости приложения.
// Используется для dependency injection и улучшения тестируемости.
type Dependencies struct {
	Logger     *logger.Logger
	BgManager  *background.Manager
	CfgManager *config.Manager
	Config     *config.Config
	Context    context.Context
}

// App представляет основное приложение со всеми его зависимостями
type App struct {
	deps    *Dependencies
	cliArgs *cli.Command
}

// New создает новый экземпляр приложения с инициализированными зависимостями.
// Возвращает ошибку, если не удалось инициализировать какую-либо зависимость.
func New(ctx context.Context) (*App, error) {
	deps, err := initializeDependencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации зависимостей: %w", err)
	}

	app := &App{
		deps:    deps,
		cliArgs: createCLIApp(deps),
	}

	return app, nil
}

// Run запускает CLI приложение с переданными аргументами
func (a *App) Run(args []string) error {
	return a.cliArgs.Run(context.Background(), args)
}

// initializeDependencies инициализирует все зависимости приложения.
// Использует ленивую инициализацию для оптимизации производительности.
func initializeDependencies(ctx context.Context) (*Dependencies, error) {
	// Инициализируем логгер первым, так как он нужен для всех остальных компонентов
	log := logger.New(paths.LogPath(), MaxLogSizeMB, true, DebugMode)

	// Логируем аргументы запуска для отладки
	log.Debug(fmt.Sprintf("Аргументы запуска: %s", strings.Join(os.Args, " ")))

	// Инициализируем менеджер фоновых процессов
	bgManager := background.New(log)

	// Инициализируем менеджер конфигурации
	cfgManager, err := config.New(log, paths.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("ошибка создания менеджера конфигурации: %w", err)
	}

	// Загружаем конфигурацию
	conf, err := cfgManager.Load()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	return &Dependencies{
		Logger:     log,
		BgManager:  bgManager,
		CfgManager: cfgManager,
		Config:     conf,
		Context:    ctx,
	}, nil
}

// createCLIApp создает и настраивает CLI приложение с русскими шаблонами
func createCLIApp(deps *Dependencies) *cli.Command {
	app := &cli.Command{
		Name:        AppName,
		Usage:       AppUsage,
		Description: AppDescription,
		Version:     version.GetVersion(),

		// Применяем кастомные русские шаблоны
		CustomHelpTemplate: RussianHelpTemplate,

		// Регистрируем все команды
		Commands: []*cli.Command{
			NewInstallCommand(deps),
			NewUninstallCommand(deps),
			NewBackgroundCommand(deps),
			NewGUICommand(deps),
			NewLogCommand(deps),
			NewConfigCommand(deps),
		},

		// Действие по умолчанию - запуск лаунчера
		Action: func(ctx *cli.Context) error {
			return RunLauncher(deps)
		},
	}

	return app
}
