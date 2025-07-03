package main

import (
	"fmt"
	"macbat/internal/config"
	"macbat/internal/monitor"
	"macbat/internal/paths"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

//================================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
//================================================================================

// findOtherInstances ищет процессы с таким же именем, исключая текущий PID
// func findOtherInstances(name string, currentPid int32) ([]int32, error) {
// 	// Получаем список всех процессов
// 	processes, err := process.Processes()
// 	if err != nil {
// 		return nil, fmt.Errorf("не удалось получить список процессов: %w", err)
// 	}

// 	var foundPids []int32

// 	for _, p := range processes {
// 		// Пропускаем текущий процесс
// 		if p.Pid == currentPid {
// 			continue
// 		}

// 		pName, err := p.Name()
// 		if err != nil {
// 			// Некоторые системные процессы могут не давать доступ к имени, игнорируем их
// 			continue
// 		}

// 		if pName == name {
// 			foundPids = append(foundPids, p.Pid)
// 		}
// 	}

// 	return foundPids, nil
// }

// launchInBackground перезапускает приложение в фоновом режиме
func launchInBackground() {
	log.Info("Запускаю фоновый процесс (--background)...")

	// Получаем путь к исполняемому файлу
	// executablePath, err := os.Executable()
	// if err != nil {
	// 	log.Fatal(fmt.Sprintf("Не удалось получить путь к исполняемому файлу: %v", err))
	// }

	// Создаем команду для запуска этого же приложения с флагом --background
	cmd := exec.Command(paths.BinaryPath(), "--background")
	// Используем те же переменные окружения, дополнительных не нужно
	cmd.Env = os.Environ()

	// Отсоединяем от стандартных потоков ввода/вывода, чтобы процесс стал независимым
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Запускаем процесс и не ждем его завершения
	err := cmd.Start()
	if err != nil {
		log.Fatal(fmt.Sprintf("Не удалось запустить фоновый процесс: %v", err))
	}

	log.Info(fmt.Sprintf("Фоновый процесс запущен с PID: %d", cmd.Process.Pid))
	// Родительский процесс успешно завершается
}

// runBackgroundMainTask - это основная логика приложения, которая работает в фоне
// runBackgroundMainTask - это основная логика приложения, которая работает в фоне
// ИЗМЕНЕНИЕ: теперь эта функция просто инициализирует и запускает монитор.
func runBackgroundMainTask(cfg *config.Config, cfgManager *config.Manager, mode string) { // Добавили cfgManager
	log.Info("Фоновый процесс проверки заряда батареи начал работу.")

	// Настраиваем канал для обработки сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создаем и запускаем монитор.
	appMonitor := monitor.NewMonitor(cfg, cfgManager, log)

	// Запускаем монитор в отдельной горутине
	go appMonitor.Start(mode)

	// Ждем сигнала завершения
	sig := <-sigChan
	log.Info(fmt.Sprintf("Получен сигнал %v, завершаю работу...", sig))

	// Корректно останавливаем монитор
	appMonitor.Stop()

	// Даем время на завершение всех горутин
	time.Sleep(500 * time.Millisecond)
}
