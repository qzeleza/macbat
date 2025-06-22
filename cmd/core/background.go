package main

import (
	"fmt"
	"macbat/internal/config"
	"macbat/internal/monitor"
	"os"
	"os/exec"

	"github.com/shirou/gopsutil/v3/process"
)

//================================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
//================================================================================

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
// runBackgroundMainTask - это основная логика приложения, которая работает в фоне
// ИЗМЕНЕНИЕ: теперь эта функция просто инициализирует и запускает монитор.
func runBackgroundMainTask(cfg *config.Config, cfgManager *config.Manager) { // Добавили cfgManager

	log.Info("Фоновый процесс проверки заряда батареи начал работу.")

	// Создаем и запускаем монитор.
	appMonitor := monitor.NewMonitor(cfg, cfgManager, log)
	appMonitor.Start() // Этот вызов заблокирует программу в бесконечном цикле.
}
