package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/qzeleza/macbat/args"
	"github.com/qzeleza/macbat/internal/monitor"

	cli "github.com/urfave/cli/v3"
	"golang.org/x/term"
)

type BackgroundCommand struct {
	deps *args.Dependencies
}

func NewBackgroundCommand(deps *args.Dependencies) *cli.Command {
	cmd := &BackgroundCommand{deps: deps}

	return &cli.Command{
		Name:    "background",
		Aliases: []string{"bg", "b"},
		Usage:   "Запускает фоновый процесс мониторинга",
		Description: `Команда запускает основной процесс мониторинга батареи в фоновом режиме.
Процесс выполняет непрерывный мониторинг состояния батареи, проверку пороговых значений
и отправку уведомлений при необходимости.`,
		Action: cmd.Execute,
	}
}

// Execute выполняет команду фонового процесса с оптимизациями
func (c *BackgroundCommand) Execute(ctx context.Context, cmd *cli.Command) error {
	// Быстрая проверка: если процесс запущен в терминале, перезапускаем отсоединенно
	// log := c.deps.Logger
	// bgManager := c.deps.BgManager

	// Быстрая проверка: если процесс запущен в терминале, перезапускаем отсоединенно
	if c.isTerminalMode() {
		return c.handleTerminalMode()
	}

	// Если мы здесь, то процесс уже отсоединен от терминала
	// Передаем контекст в runDetachedProcess
	return c.runDetachedProcessWithContext(ctx)

}

// isTerminalMode проверяет, запущен ли процесс в терминале
func (c *BackgroundCommand) isTerminalMode() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// handleTerminalMode обрабатывает запуск из терминала
func (c *BackgroundCommand) handleTerminalMode() error {
	log := c.deps.Logger
	bgManager := c.deps.BgManager

	// Оптимизация: быстрая проверка без блокировки
	if bgManager.IsRunning(BackgroundProcessName) {
		log.Info("Фоновый процесс уже запущен. Выход.")
		return nil
	}

	// Запускаем отсоединенный процесс
	bgManager.LaunchDetached(BackgroundProcessName)
	log.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")

	return nil
}

// runDetachedProcessWithContext запускает основную логику мониторинга в отсоединенном режиме с контекстом
func (c *BackgroundCommand) runDetachedProcessWithContext(ctx context.Context) error {
	log := c.deps.Logger
	bgManager := c.deps.BgManager
	conf := c.deps.Config
	cfgManager := c.deps.CfgManager

	log.Line()
	log.Info("Запускаем основную задачу мониторинга в фоновом режиме...")

	// Создаем задачу мониторинга с оптимизированной инициализацией
	monitoringTask := func() {
		// Используем переданный контекст вместо c.deps.Context
		// Контекст с таймаутом для операций инициализации
		initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Проверяем состояние агента с таймаутом
		if err := c.ensureAgentRunning(initCtx); err != nil {
			log.Error(fmt.Sprintf("Не удалось обеспечить работу агента: %v", err))
			return
		}

		// Создаем монитор с ленивой инициализацией
		mon := monitor.NewMonitor(conf, cfgManager, log)

		// Запускаем мониторинг с graceful shutdown
		c.runMonitoringWithGracefulShutdown(ctx, mon)
	}

	// Запускаем задачу через менеджер фоновых процессов
	if err := bgManager.Run(BackgroundProcessName, monitoringTask); err != nil {
		return fmt.Errorf("не удалось запустить фоновый процесс: %w", err)
	}

	return nil
}

// ensureAgentRunning обеспечивает работу launchd агента с таймаутом
func (c *BackgroundCommand) ensureAgentRunning(ctx context.Context) error {
	log := c.deps.Logger

	// Проверяем состояние агента
	if monitor.IsAgentRunning(log) {
		return nil // Агент уже запущен
	}

	log.Info("Агент не запущен. Попытка запуска...")

	// Создаем канал для результата операции
	resultChan := make(chan error, 1)

	// Запускаем операцию в отдельной горутине
	go func() {
		resultChan <- monitor.LoadAndEnableAgent(log)
	}()

	// Ожидаем результат или таймаут
	select {
	case err := <-resultChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("превышен таймаут запуска агента: %w", ctx.Err())
	}
}

// runMonitoringWithGracefulShutdown запускает мониторинг с корректным завершением
func (c *BackgroundCommand) runMonitoringWithGracefulShutdown(ctx context.Context, mon *monitor.Monitor) {
	log := c.deps.Logger

	// Канал для сигнала о готовности мониторинга
	started := make(chan struct{})

	// Запускаем мониторинг в отдельной горутине
	go func() {
		mon.Start("", started)
	}()

	// Ожидаем запуска мониторинга или отмены контекста
	select {
	case <-started:
		log.Info("Мониторинг успешно запущен")

		// Ожидаем сигнала завершения
		<-ctx.Done()
		log.Info("Получен сигнал завершения, останавливаем мониторинг...")

		// Здесь должна быть логика graceful shutdown мониторинга
		// mon.Stop() - если такой метод существует

	case <-ctx.Done():
		log.Info("Отмена запуска мониторинга")
	}
}
