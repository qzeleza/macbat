package main

import (
	"flag"
	"fmt"
	"macbat/internal/background"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/monitor"
	"macbat/internal/paths"
	"macbat/internal/tray"
	"macbat/internal/version"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// ------------------------------------------------------------------
// переменные
// ------------------------------------------------------------------
var (
	log     *logger.Logger // логгер
	modeRun string         // режим запуска
)

// ------------------------------------------------------------------
// функции
// ------------------------------------------------------------------
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
	installFlag := flag.Bool("install", false, "Устанавливает приложение и запускает агента launchd")
	uninstallFlag := flag.Bool("uninstall", false, "Удаляет приложение и агента launchd")
	backgroundFlag := flag.Bool("background", false, "Запускает фоновый процесс мониторинга (для опытных пользователей)")
	guiAgentFlag := flag.Bool("gui-agent", false, "Запускает GUI агента")
	testFlag := flag.Bool("test", false, "Запускает тестовый режим (для опытных пользователей)")
	logFlag := flag.Bool("log", false, "Отображает журнал")
	configFlag := flag.Bool("config", false, "Открывает файл конфигурации для редактирования (для опытных пользователей)")
	versionFlag := flag.Bool("version", false, "Отображает версию")

	// --- Обработка флагов командной строки ---
	flag.Parse()

	if *versionFlag {
		fmt.Printf("macbat version: %s\ncommit: %s\nbuild date: %s\n", version.Version, version.CommitHash, version.BuildDate)
		return
	}
	if *configFlag {
		log.Line()
		log.Info("Открытие конфигурации...")
		editor := "nano"
		configPath := paths.ConfigPath()
		cmd := exec.Command(editor, configPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Error(fmt.Sprintf("Ошибка запуска редактора nano: %v", err))
		}
		log.Info("Конфигурация отредактирована.")
		log.Line()
		return
	}

	// --- Логика отображения логов ---
	if *logFlag {
		logs, err := os.ReadFile(paths.LogPath())
		if err != nil {
			fmt.Println("Ошибка чтения лог-файла:", err)
		} else {
			fmt.Println("\n---- Журнал приложения ----")
			fmt.Println(string(logs))
			fmt.Printf("%s\n", strings.Repeat("-", 80))
			return
		}
	}

	// --- Логика установки/удаления ---
	if *installFlag || !monitor.IsAppInstalled(log) {
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
		if err := Uninstall(log, conf); err != nil {
			log.Fatal(fmt.Sprintf("Ошибка во время удаления: %v", err))
		}
		log.Info("Удаление успешно завершено.")
		return
	}

	// --- Логика тестового режима ---
	if *testFlag {
		log.Line()
		background.Kill(log, "--background")
		background.Kill(log, "--gui-agent")
		monitor.UnloadAgent(log)
		// Запускаем основную задачу мониторинга в тестовом режиме
		log.Info("Запускаем мониторинг батареи в тестовом режиме...")
		modeRun = "test"
		RunGUIAgent(log, conf, cfgManager, modeRun)
		*backgroundFlag = true // Запускаем фоновый процесс
	}

	// --- Логика фонового процесса ---
	if *backgroundFlag {

		// Если фоновый процесс уже запущен, то выходим
		if background.IsRunning(log) {
			log.Info("Фоновый процесс уже запущен. Выход.")
			return
		}
		log.Line()

		// Если запущен в терминале, перезапускаем в фоновом режиме и выходим
		if term.IsTerminal(int(os.Stdout.Fd())) {
			background.LaunchDetached("--background", log)
			log.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")
			return
		}

		// Если мы здесь, значит процесс уже отсоединен от терминала
		// Записываем PID файл
		if err := background.WritePID(log); err != nil {
			log.Error(fmt.Sprintf("Не удалось записать PID файла: %v", err))
		}
		log.Line()

		if !monitor.IsAgentRunning(log) {
			log.Info("Агент не запущен. Запуск...")
			monitor.LoadAgent(log)
		}
		// Запускаем основную задачу мониторинга в обычном режиме
		log.Info("Запускаем основную задачу мониторинга в обычном режиме...")
		background.Run(log, conf, cfgManager, modeRun) // Запускаем основную задачу мониторинга

		// После завершения задачи удаляем PID файл
		defer func() {
			_ = os.Remove(paths.PIDBackgoundPath())
		}()
		return
	}

	// --- Логика GUI Агента ---
	if *guiAgentFlag {
		log.Line()
		log.Info("Запускаем GUI агента (иконка в трее)...")
		// Создаем lock-файл, так как этот процесс теперь главный для GUI.
		_ = os.WriteFile(paths.GUILockPath(), []byte(strconv.Itoa(os.Getpid())), 0644)
		// Удаляем lock-файл при выходе
		defer func() {
			_ = os.Remove(paths.GUILockPath())
		}()

		// Запускаем фоновый процесс, если он еще не запущен
		if !background.IsRunning(log) {
			log.Info("Запускаем фоновый процесс мониторинга батареи...")
			background.LaunchDetached("--background", log)
		} else {
			log.Info("Фоновый процесс уже запущен.")
		}
		log.Line()
		// Запускаем блокирующий цикл GUI
		tray.Start(log, modeRun)
		return
	}

	// --- Логика Лаунчера (запуск без флагов) ---
	RunGUIAgent(log, conf, cfgManager, modeRun)
}

func RunGUIAgent(log *logger.Logger, cfg *config.Config, cfgManager *config.Manager, mode string) {
	log.Line()
	log.Info("Запускаем приложение (режим лаунчера)...")
	if background.IsGUIRunning(log) {
		log.Info("Приложение уже запущено. Выход.")
		return
	}

	log.Info("Запускаем GUI агента...")
	background.LaunchDetached("--gui-agent", log)
	log.Info("Приложение успешно запущено в фоновом режиме. Лаунчер завершает работу.")
	log.Line()
}
