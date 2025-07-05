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

	// --- Инициализация логгера
	log = logger.New(paths.LogPath(), 100, true, true)

	// --- Инициализация менеджера фоновых процессов ---
	bgManager := background.New(log)

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
	logFlag := flag.Bool("log", false, "Отображает журнал")
	configFlag := flag.Bool("config", false, "Открывает файл конфигурации для редактирования (для опытных пользователей)")
	versionFlag := flag.Bool("version", false, "Отображает версию")
	helpFlag := flag.Bool("help", false, "Отображает помощь")

	// --- Обработка флагов командной строки ---
	flag.Parse()

	// --- Вывод справки о флагах командной строки ---
	if *helpFlag {
		flag.Usage()
		return
	}

	// --- Вывод версии приложения ---
	if *versionFlag {
		fmt.Printf("Версия macbat: %s\nХеш коммита: %s\nДата сборки: %s\n", version.Version, version.CommitHash, version.BuildDate)
		return
	}

	// --- Редактирование конфигурации ---
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

	// --- Отображение логов ---
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

	// --- Установка/удаление приложения ---
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

	// --- Запуск фонового процесса ---
	if *backgroundFlag {
		// Если запущен в терминале, перезапускаем в фоновом режиме и выходим
		if term.IsTerminal(int(os.Stdout.Fd())) {
			if bgManager.IsRunning("--background") {
				log.Info("Фоновый процесс уже запущен. Выход.")
				return
			}
			bgManager.LaunchDetached("--background")
			log.Info("Перезапуск в фоновом режиме для отсоединения от терминала.")
			return
		}

		// Если мы здесь, значит процесс уже отсоединен от терминала
		log.Info("Запускаем основную задачу мониторинга в фоновом режиме...")
		task := func() {
			if !monitor.IsAgentRunning(log) {
				log.Info("Агент не запущен. Запуск...")
				monitor.LoadAgent(log)
			}
			mon := monitor.NewMonitor(conf, cfgManager, log)
			mon.Start(modeRun, nil) // Канал started не нужен в данном контексте
		}

		if err := bgManager.Run("--background", task); err != nil {
			log.Error(fmt.Sprintf("Не удалось запустить фоновый процесс: %v", err))
		}
		return
	}

	// --- Запуск GUI Агента ---
	if *guiAgentFlag {
		task := func() {
			// Запускаем фоновый процесс мониторинга, если он еще не запущен
			if !bgManager.IsRunning("--background") {
				log.Info("Запускаем фоновый процесс мониторинга батареи...")
				bgManager.LaunchDetached("--background")
			} else {
				log.Info("Фоновый процесс мониторинга уже запущен.")
			}
			log.Line()
			// Запускаем блокирующий цикл GUI
			trayApp := tray.New(log, conf, cfgManager, bgManager)
			trayApp.Start()
		}

		if err := bgManager.Run("--gui-agent", task); err != nil {
			log.Error(fmt.Sprintf("Не удалось запустить GUI агент: %v", err))
		}
		return
	}

	// --- Запуск Лаунчера (запуск без флагов) ---
	log.Line()
	log.Info("Запускаем приложение (режим лаунчера)...")
	if bgManager.IsRunning("--gui-agent") {
		log.Info("Приложение уже запущено. Выход.")
		return
	}

	log.Info("Запускаем GUI агента...")
	bgManager.LaunchDetached("--gui-agent")
	log.Info("Приложение успешно запущено в фоновом режиме. Лаунчер завершает работу.")
	log.Line()
}
