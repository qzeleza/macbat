package main

import (
	"fmt"
	"macbat/internal/logger"
	"macbat/internal/monitor"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Константа для переменной окружения, чтобы определить, является ли процесс дочерним
const childProcessEnv = "IS_CHILD_PROCESS"

var (
	log    *logger.Logger
	config *monitor.Config
)

func main() {

	// === Основная логика проверки ===

	// 1. Создаем конфигурацию с интервалами опроса.
	config = &monitor.Config{
		MinThreshold:                 21,
		MaxThreshold:                 81,
		NotificationInterval:         30 * time.Second, // Для демо уменьшим до 30 секунд
		MaxNotifications:             3,
		LogFilePath:                  "/tmp/macbat.log",
		LogRotationLines:             100,              // Ротация после каждых 5 записей
		CheckIntervalWhenCharging:    30 * time.Second, // Когда заряжается, проверяем каждые 30 секунд
		CheckIntervalWhenDischarging: 30 * time.Minute, // Когда разряжается, проверяем каждые 30 минут
		UseSimulator:                 false,            // true: использовать симулятор для тестирования.
		LogEnabled:                   true,             // true: включить логирование.
		DebugEnabled:                 false,            // true: включить DEBUG логирование.
	}

	// 2. Создаем логгер.
	log = logger.New(config.LogFilePath, config.LogRotationLines, config.LogEnabled, config.DebugEnabled)

	// Проверяем, запущен ли этот процесс как дочерний (фоновый)
	if os.Getenv(childProcessEnv) == "1" {
		// Запускаем фоновую задачу
		runBackgroundMainTask(*config)
		return
	}

	// 3. Получаем информацию о текущем процессе
	currentPid := int32(os.Getpid())
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось получить путь к исполняемому файлу: %v", err))
	}
	executableName := filepath.Base(executablePath)

	// 4. Ищем другие запущенные экземпляры этого же приложения
	pids, err := findOtherInstances(executableName, currentPid)
	if err != nil {
		log.Fatal(fmt.Sprintf("Ошибка при поиске других экземпляров: %v", err))
	}

	// 5. Если найдены другие экземпляры, выводим их PID и выходим
	if len(pids) > 0 {
		log.Info("Обнаружены другие запущенные экземпляры приложения с PID:")
		for _, pid := range pids {
			log.Info(fmt.Sprintf("%d", pid))
		}
		log.Info("Выход.")
		os.Exit(1)
	}

	// 6. Если мы первые, запускаем себя в фоновом режиме
	log.Info("Инициализация основного первого фонового процесса...")
	launchInBackground()
}

// findOtherInstances ищет процессы с таким же именем, исключая текущий PID
func findOtherInstances(name string, currentPid int32) ([]int32, error) {
	// Получаем список всех процессов
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список процессов: %w", err)
	}

	var foundPids []int32

	for _, p := range processes {
		// Пропускаем текущий процесс
		if p.Pid == currentPid {
			continue
		}

		pName, err := p.Name()
		if err != nil {
			// Некоторые системные процессы могут не давать доступ к имени, игнорируем их
			continue
		}

		if pName == name {
			foundPids = append(foundPids, p.Pid)
		}
	}

	return foundPids, nil
}

// launchInBackground перезапускает приложение в фоновом режиме
func launchInBackground() {
	log.Info("Запускаю основной процесс в фоновом режиме...")

	// Получаем путь к исполняемому файлу
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось получить путь к исполняемому файлу: %v", err))
	}

	// Создаем команду для запуска этого же приложения
	cmd := exec.Command(executablePath)
	// Устанавливаем переменную окружения, чтобы дочерний процесс знал о своей роли
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=1", childProcessEnv))

	// Отсоединяем от стандартных потоков ввода/вывода, чтобы процесс стал независимым
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Запускаем процесс и не ждем его завершения
	err = cmd.Start()
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось запустить фоновый процесс: %v", err))
	}

	fmt.Printf("Процесс запущен в фоне с PID: %d\n", cmd.Process.Pid)
	// Родительский процесс успешно завершается
}

// runBackgroundMainTask - это основная логика приложения, которая работает в фоне
func runBackgroundMainTask(config monitor.Config) {

	log.Info("Фоновый процесс проверки заряда батареи начал работу.")

	// 4. Создаем и запускаем монитор.
	monitor := monitor.NewMonitor(config, log)
	monitor.Start() // Этот вызов заблокирует программу в бесконечном цикле.

}
