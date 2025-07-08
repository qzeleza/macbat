package main

import (
	"fmt"
	"os"
)

func main() {
	// Создаем и инициализируем приложение
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации: %v\n", err)
		os.Exit(1)
	}

	// Запускаем приложение
	if err := app.Run(os.Args); err != nil {
		app.Logger().Fatal(fmt.Sprintf("Ошибка запуска приложения: %v", err))
	}
}
