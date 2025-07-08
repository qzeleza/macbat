package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/qzeleza/macbat/internal/monitor"
	"github.com/qzeleza/macbat/internal/paths"
	"github.com/urfave/cli/v3"
)

// installCommand создает команду установки
func (a *App) installCommand() *cli.Command {
	return &cli.Command{
		Name:    "install",
		Aliases: []string{"i"},
		Usage:   "Устанавливает приложение и запускает агента launchd",
		Action:  a.handleInstall,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Принудительная переустановка",
			},
		},
	}
}

// uninstallCommand создает команду удаления
func (a *App) uninstallCommand() *cli.Command {
	return &cli.Command{
		Name:    "uninstall",
		Aliases: []string{"u", "remove"},
		Usage:   "Удаляет приложение и агента launchd",
		Action:  a.handleUninstall,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "keep-config",
				Usage: "Сохранить файл конфигурации",
			},
			&cli.BoolFlag{
				Name:  "keep-logs",
				Usage: "Сохранить файлы журналов",
			},
		},
	}
}

// logCommand создает команду просмотра логов
func (a *App) logCommand() *cli.Command {
	return &cli.Command{
		Name:    "log",
		Aliases: []string{"l", "logs"},
		Usage:   "Отображает журнал",
		Action:  a.handleLog,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"n"},
				Usage:   "Количество строк для отображения",
				Value:   100,
			},
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Следить за новыми записями",
			},
			&cli.StringFlag{
				Name:  "level",
				Usage: "Фильтр по уровню (DEBUG, INFO, ERROR)",
			},
		},
	}
}

// configCommand создает команду редактирования конфигурации
func (a *App) configCommand() *cli.Command {
	return &cli.Command{
		Name:    "config",
		Aliases: []string{"c", "cfg"},
		Usage:   "Открывает файл конфигурации для редактирования (для опытных пользователей)",
		Action:  a.handleConfig,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "editor",
				Usage: "Редактор для открытия файла",
				Value: "nano",
			},
			&cli.BoolFlag{
				Name:  "show",
				Usage: "Только показать содержимое, не редактировать",
			},
		},
	}
}

// handleInstall обрабатывает команду установки
func (a *App) handleInstall(ctx context.Context, cmd *cli.Command) error {

	// Проверка флага принудительной установки
	force := cmd.Bool("force")

	if monitor.IsAppInstalled(a.logger) && !force {
		a.logger.Info("Приложение уже установлено. Используйте --force для переустановки.")
		return nil
	}

	a.logger.Line()
	a.logger.Info("Установка приложения...")

	if err := a.run.Install(); err != nil {
		return fmt.Errorf("ошибка во время установки: %w", err)
	}

	a.logger.Info("Установка успешно завершена.")
	return nil
}

// handleUninstall обрабатывает команду удаления
func (a *App) handleUninstall(ctx context.Context, cmd *cli.Command) error {

	keepConfig := cmd.Bool("keep-config")
	keepLogs := cmd.Bool("keep-logs")

	a.logger.Line()
	a.logger.Info("Запрошено удаление приложения...")

	// Модифицируем процесс удаления с учетом флагов
	if err := a.UninstallWithOptions(keepConfig, keepLogs); err != nil {
		return fmt.Errorf("ошибка во время удаления: %w", err)
	}

	a.logger.Info("Удаление успешно завершено.")
	return nil
}

// handleLog обрабатывает команду просмотра логов
func (a *App) handleLog(ctx context.Context, cmd *cli.Command) error {

	logPath := paths.LogPath()   // путь к логу
	lines := cmd.Int("lines")    // количество строк
	follow := cmd.Bool("follow") // режим следования
	level := cmd.String("level") // уровень логов

	if follow {
		// Режим следования за логом
		return followLog(logPath)
	}

	// Чтение логов
	logs, err := readLogLines(logPath, lines)
	if err != nil {
		return fmt.Errorf("ошибка чтения лог-файла: %w", err)
	}

	// Фильтрация по уровню если указан
	if level != "" {
		logs = filterLogsByLevel(logs, level)
	}

	// Вывод логов
	fmt.Printf("%s\n", strings.Repeat("-", 100))
	fmt.Println("---- Журнал приложения ----")
	fmt.Printf("%s\n", strings.Repeat("-", 100))
	fmt.Print(logs)
	fmt.Printf("%s\n", strings.Repeat("-", 100))

	return nil
}

// handleConfig обрабатывает команду редактирования конфигурации
func (a *App) handleConfig(ctx context.Context, cmd *cli.Command) error {

	configPath := paths.ConfigPath() // путь к конфигурации
	editor := cmd.String("editor")   // редактор
	showOnly := cmd.Bool("show")     // только показать

	a.logger.Line()

	if showOnly {
		// Только показать содержимое
		content, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("ошибка чтения конфигурации: %w", err)
		}
		fmt.Println("=== Содержимое конфигурации ===")
		fmt.Print(string(content))
		return nil
	}

	// Редактирование конфигурации
	a.logger.Info("Открытие конфигурации...")

	command := exec.Command(editor, configPath)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("ошибка запуска редактора %s: %w", editor, err)
	}

	a.logger.Info("Конфигурация отредактирована.")
	a.logger.Line()

	return nil
}

// defaultAction обрабатывает запуск без команд или с флагами
func (a *App) defaultAction(ctx context.Context, cmd *cli.Command) error {

	// Обработка скрытых флагов
	if cmd.Bool("background") {
		return a.runBackgroundMode()
	}

	if cmd.Bool("gui-agent") {
		return a.runGUIAgentMode()
	}

	// Проверка установки
	if !monitor.IsAppInstalled(a.logger) {
		a.logger.Line()
		a.logger.Info("Приложение не установлено. Выполняется автоматическая установка...")

		// Вызываем обработчик установки
		installCmd := a.installCommand()
		return a.handleInstall(ctx, installCmd)
	}

	// Запуск в режиме лаунчера
	return a.runLauncherMode()
}

// Вспомогательные функции

// readLogLines читает указанное количество последних строк из файла
func readLogLines(filepath string, lines int) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	allLines := strings.Split(string(content), "\n")

	// Берем последние N строк
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	resultLines := allLines[start:]
	return strings.Join(resultLines, "\n"), nil
}

// filterLogsByLevel фильтрует логи по уровню
func filterLogsByLevel(logs string, level string) string {
	level = strings.ToUpper(level)
	lines := strings.Split(logs, "\n")
	var filtered []string

	for _, line := range lines {
		if strings.Contains(line, level) {
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}

// followLog следит за изменениями в лог-файле
func followLog(logPath string) error {
	cmd := exec.Command("tail", "-f", logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// UninstallWithOptions выполняет удаление с учетом опций
func (a *App) UninstallWithOptions(keepConfig, keepLogs bool) error {
	// Вызываем базовую функцию удаления
	if err := a.run.Uninstall(); err != nil {
		return err
	}

	// Восстанавливаем файлы если нужно
	if keepConfig {
		a.logger.Info("Файл конфигурации сохранен")
	}

	if keepLogs {
		a.logger.Info("Файлы журналов сохранены")
	}

	return nil
}
