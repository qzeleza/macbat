// cmd/macbat/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// main является точкой входа в приложение.
// Ответственность ограничена только инициализацией и запуском приложения.
func main() {
	// Привязываем горутину к главному потоку для GUI компонентов
	runtime.LockOSThread()

	// Создаем контекст с возможностью отмены для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку системных сигналов для корректного завершения
	setupSignalHandling(cancel)

	// Создаем приложение и запускаем его
	application, err := New(ctx)
	if err != nil {
		fmt.Printf("Критическая ошибка инициализации: %v\n", err)
		os.Exit(1)
	}

	// Запускаем приложение с таймаутом
	if err := runWithTimeout(ctx, application); err != nil {
		fmt.Printf("Ошибка выполнения: %v\n", err)
		os.Exit(1)
	}
}

// setupSignalHandling настраивает корректную обработку системных сигналов
// для graceful shutdown приложения
func setupSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()
}

// runWithTimeout запускает приложение с таймаутом для предотвращения зависания
func runWithTimeout(ctx context.Context, application *App) error {
	// Создаем контекст с таймаутом для критических операций
	timeoutCtx, cancel := context.WithTimeout(ctx, AppStartupTimeout)
	defer cancel()

	// Канал для передачи результата выполнения
	resultChan := make(chan error, 1)

	// Запускаем приложение в отдельной горутине
	go func() {
		resultChan <- application.Run(os.Args)
	}()

	// Ожидаем завершения или таймаута
	select {
	case err := <-resultChan:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("превышен таймаут запуска приложения (%v)", AppStartupTimeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}
