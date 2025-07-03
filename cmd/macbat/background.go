package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"macbat/internal/config"
	"macbat/internal/monitor"
	"macbat/internal/paths"
	"github.com/sirupsen/logrus"
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
	logrus.Info("Запускаю фоновый процесс (--background)...")

	// Получаем путь к исполняемому файлу
	// executablePath, err := os.Executable()
	// if err != nil {
	// 	log.Fatal(fmt.Sprintf("Не удалось получить путь к исполняемому файлу: %v", err))
	// }

	binPath := paths.BinaryPath()
	logrus.Info(fmt.Sprintf("Путь к исполняемому файлу для фонового процесса: %s", binPath))
	// Создаем команду для запуска этого же приложения с флагом --background
	cmd := exec.Command(binPath, "--background")
	// Используем те же переменные окружения, дополнительных не нужно
	cmd.Env = os.Environ()

	// Отсоединяем от стандартных потоков ввода/вывода, чтобы процесс стал независимым
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	logrus.Info("Команда для запуска фонового процесса создана. Запускаю процесс...")
	// Запускаем процесс и не ждем его завершения
	err := cmd.Start()
	if err != nil {
		logrus.Error(fmt.Sprintf("Не удалось запустить фоновый процесс: %v", err))
		logrus.Info("Попытка запуска фонового процесса не удалась. Продолжаю без завершения программы.")
		return
	}

	logrus.Info(fmt.Sprintf("Фоновый процесс успешно запущен с PID: %d", cmd.Process.Pid))
	// Родительский процесс успешно завершается
}

// runBackgroundMainTask - это основная логика приложения, которая работает в фоне
// runBackgroundMainTask - это основная логика приложения, которая работает в фоне
// ИЗМЕНЕНИЕ: теперь эта функция просто инициализирует и запускает монитор.
func runBackgroundMainTask(cfg *config.Config, cfgManager *config.Manager, mode string) { // Добавили cfgManager
	logrus.Info("Запуск в фоновом режиме...")

	logrus.Info("Проверка существования PID-файла перед запуском фонового процесса.")
	pidFile := paths.PidFile()
	if _, err := os.Stat(pidFile); err == nil {
		logrus.Info(fmt.Sprintf("PID-файл уже существует: %s. Читаю содержимое.", pidFile))
		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			logrus.Error(fmt.Sprintf("Ошибка чтения PID-файла: %v", err))
		} else {
			pidStr := string(pidBytes)
			pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
			if err != nil {
				logrus.Error(fmt.Sprintf("Ошибка парсинга PID из файла: %v", err))
			} else {
				if isProcessRunning(pid) {
					logrus.Info(fmt.Sprintf("Процесс с PID %d из PID-файла уже запущен. Завершаю текущий запуск.", pid))
					os.Exit(0)
				} else {
					logrus.Info(fmt.Sprintf("Процесс с PID %d из PID-файла не найден. Продолжаю запуск.", pid))
				}
			}
		}
	} else {
		logrus.Info("PID-файл не существует. Создаю новый PID-файл для текущего процесса.")
	}

	// Записываем PID текущего процесса в файл
	currentPID := os.Getpid()
	logrus.Info(fmt.Sprintf("Запись PID текущего процесса (%d) в файл: %s", currentPID, pidFile))
	err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", currentPID)), 0644)
	if err != nil {
		logrus.Error(fmt.Sprintf("Ошибка записи PID в файл: %v", err))
	}
	logrus.Info("PID успешно записан в файл.")

	logrus.Info("Фоновый процесс проверки заряда батареи начал работу.")

	// Настраиваем канал для обработки сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Создаем и запускаем монитор.
	appMonitor := monitor.NewMonitor(cfg, cfgManager, logrus)

	// Запускаем монитор в отдельной горутине
	go appMonitor.Start(mode)

	// Ждем сигнала завершения
	sig := <-sigChan
	logrus.Info(fmt.Sprintf("Получен сигнал %v, завершаю работу...", sig))

	// Корректно останавливаем монитор
	appMonitor.Stop()

	// Даем время на завершение всех горутин
	time.Sleep(500 * time.Millisecond)
}
