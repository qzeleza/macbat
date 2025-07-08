package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qzeleza/macbat/internal/background"
	"github.com/qzeleza/macbat/internal/logger"
	"github.com/qzeleza/macbat/internal/monitor"
	"github.com/qzeleza/macbat/internal/tray"
	"golang.org/x/term"
)

// BackgroundMode представляет режим работы фонового процесса
type BackgroundMode string

const (
	// BackgroundModeMonitor режим мониторинга батареи
	BackgroundModeMonitor BackgroundMode = "--background"

	// BackgroundModeGUI режим GUI агента
	BackgroundModeGUI BackgroundMode = "--gui-agent"
)

// runBackgroundMode запускает фоновый процесс мониторинга
func (a *App) runBackgroundMode() error {
	bgManager := background.New(a.logger)

	// Если запущен в терминале, перезапускаем в фоновом режиме
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if bgManager.IsRunning(string(BackgroundModeMonitor)) {
			a.logger.Info("Фоновый процесс уже запущен. Выход.")
			return nil
		}

		bgManager.LaunchDetached(string(BackgroundModeMonitor))
		a.logger.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")
		return nil
	}

	// Процесс уже отсоединен от терминала
	a.logger.Line()
	a.logger.Info("Запускаем основную задачу мониторинга в фоновом режиме...")

	// Создаем задачу мониторинга
	task := a.createMonitorTask()

	// Запускаем фоновый процесс
	if err := bgManager.Run(string(BackgroundModeMonitor), task); err != nil {
		return fmt.Errorf("не удалось запустить фоновый процесс: %w", err)
	}

	return nil
}

// runGUIAgentMode запускает GUI агент
func (a *App) runGUIAgentMode() error {
	a.logger.Info("Запуск в режиме GUI-агента...")

	bgManager := background.New(a.logger)

	// Захватываем lock-файл
	if err := bgManager.Lock(string(BackgroundModeGUI)); err != nil {
		return fmt.Errorf("не удалось запустить GUI агент: %w", err)
	}
	defer bgManager.Unlock(string(BackgroundModeGUI))

	// Записываем PID
	if err := bgManager.WritePID(string(BackgroundModeGUI)); err != nil {
		a.logger.Error(fmt.Sprintf("Не удалось записать PID: %v", err))
	}

	// Регистрируем обработчики сигналов
	bgManager.HandleSignals(string(BackgroundModeGUI))

	// Запускаем фоновый процесс мониторинга если нужно
	if err := a.ensureBackgroundMonitor(bgManager); err != nil {
		a.logger.Error(fmt.Sprintf("Ошибка запуска мониторинга: %v", err))
	}

	a.logger.Line()

	// Запускаем GUI в основном потоке
	trayApp := tray.New(a.logger, a.cfg, a.cfgManager, bgManager)
	trayApp.Start()

	return nil
}

// runLauncherMode запускает приложение в режиме лаунчера
func (a *App) runLauncherMode() error {
	a.logger.Line()
	a.logger.Info("Запускаем приложение (режим лаунчера)...")

	bgManager := background.New(a.logger)

	// Проверяем, запущен ли GUI агент
	if bgManager.IsRunning(string(BackgroundModeGUI)) {
		a.logger.Info("Приложение уже запущено. Выход.")
		return nil
	}

	// Запускаем GUI агент в фоне
	a.logger.Info("Запускаем GUI агента...")
	bgManager.LaunchDetached(string(BackgroundModeGUI))

	a.logger.Info("Приложение успешно запущено в фоновом режиме. Лаунчер завершает работу.")
	a.logger.Line()

	return nil
}

// createMonitorTask создает задачу мониторинга батареи
func (a *App) createMonitorTask() func() {
	return func() {
		// Проверяем, запущен ли агент
		if !monitor.IsAgentRunning(a.logger) {
			a.logger.Info("Агент не запущен. Попытка запуска...")
			if err := monitor.LoadAndEnableAgent(a.logger); err != nil {
				a.logger.Error(fmt.Sprintf("Не удалось запустить агента: %v", err))
				return
			}
		}

		// Создаем и запускаем монитор
		mon := monitor.NewMonitor(a.cfg, a.cfgManager, a.logger)
		mon.Start("", nil) // modeRun и канал не используются
	}
}

// ensureBackgroundMonitor проверяет и запускает фоновый монитор если нужно
func (a *App) ensureBackgroundMonitor(bgManager *background.Manager) error {
	if !bgManager.IsRunning(string(BackgroundModeMonitor)) {
		a.logger.Info("Запускаем фоновый процесс мониторинга батареи...")
		bgManager.LaunchDetached(string(BackgroundModeMonitor))
	} else {
		a.logger.Info("Фоновый процесс мониторинга уже запущен.")
	}
	return nil
}

// StopAllBackgroundProcesses останавливает все фоновые процессы
func StopAllBackgroundProcesses(log *logger.Logger) {
	bgManager := background.New(log)

	log.Info("Остановка всех фоновых процессов...")

	// Останавливаем процесс мониторинга
	if bgManager.IsRunning(string(BackgroundModeMonitor)) {
		bgManager.Kill(string(BackgroundModeMonitor))
		log.Info("Процесс мониторинга остановлен")
	}

	// Останавливаем GUI агент
	if bgManager.IsRunning(string(BackgroundModeGUI)) {
		bgManager.Kill(string(BackgroundModeGUI))
		log.Info("GUI агент остановлен")
	}
}

// CheckBackgroundProcesses проверяет состояние фоновых процессов
func CheckBackgroundProcesses(log *logger.Logger) map[string]bool {
	bgManager := background.New(log)

	status := make(map[string]bool)
	status["monitor"] = bgManager.IsRunning(string(BackgroundModeMonitor))
	status["gui"] = bgManager.IsRunning(string(BackgroundModeGUI))

	return status
}

// RestartBackgroundProcess перезапускает указанный фоновый процесс
func RestartBackgroundProcess(log *logger.Logger, mode BackgroundMode) error {
	bgManager := background.New(log)

	// Останавливаем процесс
	if bgManager.IsRunning(string(mode)) {
		bgManager.Kill(string(mode))
		// Небольшая задержка для корректной остановки
		time.Sleep(time.Second)
	}

	// Запускаем заново
	bgManager.LaunchDetached(string(mode))

	log.Info(fmt.Sprintf("Процесс %s перезапущен", mode))
	return nil
}

// CleanupBackgroundFiles очищает временные файлы фоновых процессов
func CleanupBackgroundFiles(log *logger.Logger) error {
	patterns := []string{
		"/tmp/macbat-*.lock",
		"/tmp/macbat-*.pid",
	}

	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Error(fmt.Sprintf("Ошибка поиска файлов %s: %v", pattern, err))
			continue
		}

		for _, file := range files {
			if err := os.Remove(file); err != nil {
				log.Error(fmt.Sprintf("Не удалось удалить %s: %v", file, err))
			} else {
				log.Debug(fmt.Sprintf("Удален файл: %s", file))
			}
		}
	}

	return nil
}
