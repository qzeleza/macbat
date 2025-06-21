package main

import (
	"fmt"
	"log"
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

func main() {
	// Проверяем, запущен ли этот процесс как дочерний (фоновый)
	if os.Getenv(childProcessEnv) == "1" {
		// Запускаем фоновую задачу
		runBackgroundTask()
		return
	}

	// === Основная логика проверки ===

	// 1. Получаем информацию о текущем процессе
	currentPid := int32(os.Getpid())
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Не удалось получить путь к исполняемому файлу: %v", err)
	}
	executableName := filepath.Base(executablePath)

	// 2. Ищем другие запущенные экземпляры этого же приложения
	pids, err := findOtherInstances(executableName, currentPid)
	if err != nil {
		log.Fatalf("Ошибка при поиске других экземпляров: %v", err)
	}

	// 3. Если найдены другие экземпляры, выводим их PID и выходим
	if len(pids) > 0 {
		fmt.Println("Обнаружены другие запущенные экземпляры приложения с PID:")
		for _, pid := range pids {
			fmt.Println(pid)
		}
		fmt.Println("Выход.")
		os.Exit(1)
	}

	// 4. Если мы первые, запускаем себя в фоновом режиме
	fmt.Println("Я первый!")
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
	fmt.Println("Запускаю основной процесс в фоновом режиме...")

	// Получаем путь к исполняемому файлу
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Не удалось получить путь к исполняемому файлу: %v", err)
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
		log.Fatalf("Не удалось запустить фоновый процесс: %v", err)
	}

	fmt.Printf("Процесс запущен в фоне с PID: %d\n", cmd.Process.Pid)
	// Родительский процесс успешно завершается
}

// runBackgroundTask - это основная логика вашего приложения, которая работает в фоне
func runBackgroundTask() {

	// 1. Создаем конфигурацию с интервалами опроса.
	config := monitor.Config{
		MinThreshold:                 20,
		MaxThreshold:                 80,
		NotificationInterval:         30 * time.Second, // Для демо уменьшим до 30 секунд
		MaxNotifications:             3,
		CheckIntervalWhenCharging:    3 * time.Second, // Когда заряжается, проверяем каждые 30 секунд
		CheckIntervalWhenDischarging: 6 * time.Second, // Когда разряжается, проверяем каждые 30 минут
		LogFilePath:                  "/tmp/macbat.log",
		LogRotationLines:             100, // Ротация после каждых 5 записей
		// CheckIntervalWhenCharging:    30 * time.Second, // Когда заряжается, проверяем каждые 30 секунд
		// CheckIntervalWhenDischarging: 30 * time.Minute, // Когда разряжается, проверяем каждые 30 минут
		//
		// --- ГЛАВНЫЙ ПЕРЕКЛЮЧАТЕЛЬ ---
		// true: использовать симулятор для тестирования.
		// false: попытаться использовать реальные данные ОС (требует реализации getRealBatteryInfo).
		//
		UseSimulator: true,
		LogEnabled:   true,
		DebugEnabled: true,
	}

	// 2. Создаем логгер.
	log := logger.New(config.LogFilePath, config.LogRotationLines, config.LogEnabled, config.DebugEnabled)
	log.Info("Фоновый процесс проверки заряда батареи начал работу.")

	// 4. Создаем и запускаем монитор.
	monitor := monitor.NewMonitor(config, log)
	monitor.Start() // Этот вызов заблокирует программу в бесконечном цикле.

}
