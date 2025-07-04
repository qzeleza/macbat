package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/getlantern/systray"
	"golang.org/x/term"

	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
)

var log *logger.Logger
var modeRun string

func main() {

	modeRun = "run"

	// Инициализация логгера
	log = logger.New(paths.LogPath(), 100, true, true)

	// --- Инициализация конфигурации ---
	cfgManager, err := config.New(log, paths.ConfigPath())
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка инициализации менеджера конфигурации: %v", err))
	}
	conf, err := cfgManager.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка загрузки конфигурации: %v", err))
	}

	// --- Обработка флагов командной строки ---
	installFlag := flag.Bool("install", false, "Установить приложение и агент launchd")
	uninstallFlag := flag.Bool("uninstall", false, "Удалить приложение и агент launchd")
	backgroundFlag := flag.Bool("background", false, "Запуск фонового процесса мониторинга")
	guiAgentFlag := flag.Bool("gui-agent", false, "Внутренний флаг для запуска GUI агента")
	testFlag := flag.Bool("test", false, "Запуск тестового режима")
	flag.Parse()

	// --- Логика установки/удаления ---
	if *installFlag || !isAppInstalled(log) {
		log.Line()
		log.Info("Установка приложения...")
		if err := Install(log, conf); err != nil {
			log.Fatal(fmt.Sprintf("Ошибка во время установки: %v", err))
		}
		log.Info("Установка успешно завершена.")
		// Если запрошена установка, то выходим
		if *installFlag {
			return
		}
	}
	if *uninstallFlag {
		log.Line()
		log.Info("Запрошено удаление приложения...")
		if err := Uninstall(log); err != nil {
			log.Fatal(fmt.Sprintf("Ошибка во время удаления: %v", err))
		}
		log.Info("Удаление успешно завершено.")
		return
	}

	// --- Логика тестового режима ---
	if *testFlag {
		log.Line()
		killBackgroundGo() // Завершаем фоновый процесс
		// Запускаем основную задачу мониторинга в тестовом режиме
		log.Info("Запуск мониторинга батареи в тестовом режиме...")
		modeRun = "test"
		runBackgroundMainTask(conf, cfgManager, modeRun)
		return
	}

	// --- Логика фонового процесса ---
	if *backgroundFlag {

		// Если фоновый процесс уже запущен, то выходим
		if isBackgroundRunning() {
			log.Info("Фоновый процесс уже запущен. Выход.")
			return
		}
		log.Line()

		// Если запущен в терминале, перезапускаем в фоновом режиме и выходим
		if term.IsTerminal(int(os.Stdout.Fd())) {
			launchDetached("--background")
			log.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")
			return
		}

		// Если мы здесь, значит процесс уже отсоединен от терминала
		// Записываем PID файл
		if err := writePID(); err != nil {
			log.Error(fmt.Sprintf("Не удалось записать PID файла: %v", err))
		}
		log.Line()

		// Запускаем основную задачу мониторинга в обычном режиме
		log.Info("Запуск мониторинга батареи в обычном режиме...")
		// killBackgroundGo()                               // Завершаем фоновый процесс
		runBackgroundMainTask(conf, cfgManager, modeRun) // Запускаем основную задачу мониторинга

		// После завершения задачи удаляем PID файл
		defer func() {
			_ = os.Remove(paths.PIDPath())
		}()
		return
	}

	// --- Логика GUI Агента ---
	if *guiAgentFlag {
		log.Line()
		log.Info("Запуск агента GUI (иконка в трее)...")
		// Создаем lock-файл, так как этот процесс теперь главный для GUI.
		_ = os.WriteFile(paths.GUILockPath(), []byte(strconv.Itoa(os.Getpid())), 0644)

		// Запускаем фоновый процесс, если он еще не запущен
		if !isBackgroundRunning() {
			log.Info("Запуск фонового процесса мониторинга батареи...")
			launchDetached("--background")
		} else {
			log.Info("Фоновый процесс уже запущен.")
		}
		log.Line()
		// Запускаем блокирующий цикл GUI
		systray.Run(onReady, onExit)
		return
	}

	// --- Логика Лаунчера (запуск без флагов) ---
	log.Line()
	log.Info("Запуск приложения (режим лаунчера)...")
	if isGUIRunning() {
		log.Info("Приложение уже запущено. Выход.")
		return
	}

	log.Info("Запуск GUI агента...")
	launchDetached("--gui-agent")
	log.Info("Приложение успешно запущено в фоновом режиме. Лаунчер завершает работу.")
	log.Line()
}

func onExit() {
	// Здесь можно выполнить очистку перед выходом
	log.Info("Выход из приложения")
	// Удаляем lock-файл GUI перед выходом
	_ = os.Remove(paths.GUILockPath())
	os.Exit(0)
}
