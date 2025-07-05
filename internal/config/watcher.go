package config

import (
	"fmt"
	"time"

	"macbat/internal/logger"

	"github.com/fsnotify/fsnotify"
)

/**
 * @brief Запускает наблюдателя за файлом конфигурации.
 *
 * Эта функция создает нового наблюдателя за файловой системой для отслеживания изменений
 * в файле конфигурации. При обнаружении события записи (изменения) файла, она
 * перезагружает конфигурацию и отправляет обновленный объект в предоставленный канал.
 * Функция предназначена для выполнения в отдельной горутине.
 *
 * @param configPath Путь к файлу конфигурации.
 * @param updateChan Канал, в который будет отправлена обновленная конфигурация.
 * @param log Объект для логирования информационных сообщений и ошибок.
 */
func Watch(configPath string, updateChan chan<- *Config, log *logger.Logger) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(fmt.Sprintf("Критическая ошибка: не удалось создать наблюдателя за файлами: %v", err))
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					log.Info(fmt.Sprintf("Обнаружено изменение в файле конфигурации: %s. Перезагрузка...", event.Name))
					time.Sleep(100 * time.Millisecond) // Короткая пауза на случай множественных событий сохранения от редактора.

					cfgManager, err := New(log, configPath)
					if err != nil {
						log.Error(fmt.Sprintf("Не удалось создать менеджер конфигурации для перезагрузки: %v", err))
						continue
					}

					newCfg, err := cfgManager.Load()
					if err != nil {
						log.Error(fmt.Sprintf("Не удалось перезагрузить конфигурацию после изменения: %v", err))
						continue
					}
					// Отправляем новую конфигурацию в основной цикл через канал.
					updateChan <- newCfg
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error(fmt.Sprintf("Ошибка наблюдателя за файлами: %v", err))
			}
		}
	}()

	err = watcher.Add(configPath)
	if err != nil {
		log.Error(fmt.Sprintf("Критическая ошибка: не удалось добавить файл %s в наблюдение: %v", configPath, err))
		return
	}

	log.Info(fmt.Sprintf("Наблюдатель запущен для файла: %s", configPath))

	// Блокируем горутину, чтобы она не завершилась.
	// Так как эта функция сама должна быть запущена в горутине,
	// она будет жить, пока жив основной процесс.
	<-make(chan struct{})
}
