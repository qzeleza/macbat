package main

import (
	"fmt"
	"macbat/internal/config"
	"macbat/internal/logger"
	"macbat/internal/paths"
	"os"
	"path/filepath"
)

// Константа для переменной окружения, чтобы определить, является ли процесс дочерним
const childProcessEnv = "IS_CHILD_PROCESS"

var (
	log *logger.Logger
)

func main() {

	// === Основная логика проверки ===

	// 1. Создаем логгер.
	log = logger.New(paths.LogPath(), 100, true, false)

	// 2. Инициализируем менеджер конфигурации
	// New вернет менеджер, использующий путь по умолчанию.
	cfgManager, err := config.New(log, paths.ConfigPath())
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось инициализировать менеджер конфигурации: %v", err))
	}

	// 3. Загружаем конфигурацию
	conf, err := cfgManager.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось загрузить конфигурацию: %v", err))
	}

	// 4. Проверяем, установлено ли приложение
	if !isAppInstalled(log) {

		// // Удаляем старый лог-файл.
		// err = os.Remove(paths.LogPath())
		// if err != nil && !os.IsNotExist(err) {
		// 	// Если файл не существует, это не ошибка. В иных случаях - выводим предупреждение.
		// 	log.Debug("Предупреждение: не удалось удалить старый лог-файл")
		// }

		log.Info("Приложение не установлено. Производим установку...")
		err = Install(log, conf)
		if err != nil {
			log.Fatal(fmt.Sprintf("Не удалось установить приложение: %v", err))
		}

	}
	// 5. Проверяем, запущен ли этот процесс как дочерний (фоновый)
	if os.Getenv(childProcessEnv) == "1" {
		// Запускаем фоновую задачу
		runBackgroundMainTask(conf, cfgManager)
		return
	}

	// 6. Получаем информацию о текущем процессе
	currentPid := int32(os.Getpid())
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось получить путь к исполняемому файлу: %v", err))
	}
	executableName := filepath.Base(executablePath)

	// 7. Ищем другие запущенные экземпляры этого же приложения
	pids, err := findOtherInstances(executableName, currentPid)
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка при поиске других экземпляров: %v", err))
	}

	// 8. Если найдены другие экземпляры, выводим их PID и выходим
	if len(pids) > 0 {
		log.Info("Обнаружены другие запущенные экземпляры приложения с PID:")
		for _, pid := range pids {
			log.Info(fmt.Sprintf("%d", pid))
		}
		log.Info("Выход.")
		os.Exit(1)
	}

	// 9. Если мы первые, запускаем себя в фоновом режиме
	log.Info("Инициализация основного первого фонового процесса...")
	launchInBackground()
}
