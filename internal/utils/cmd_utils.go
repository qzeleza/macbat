package utils

import (
	"fmt"
	"macbat/internal/logger"
	"os"
	"path/filepath"
)

// CheckWriteAccess проверяет доступность директории для записи.
// Для этого она пытается создать и немедленно удалить временный файл.
//
// @param dir string - директория для проверки.
// @param log *logger.Logger - логгер для вывода отладочной информации.
// @return error - ошибка, если директория недоступна для записи.
// Возвращает nil, если права на запись имеются.
// CheckWriteAccess проверяет права на запись в указанную директорию.
func CheckWriteAccess(dir string, log *logger.Logger) error {
	log.Debug(fmt.Sprintf("Проверка прав на запись в директорию: %s", dir))

	// Имя тестового файла. Точка в начале делает его скрытым в Unix-системах.
	const testFileName = ".write_access_test"
	testFilePath := filepath.Join(dir, testFileName)

	// Гарантируем удаление тестового файла после завершения функции.
	// defer выполнится в самом конце, даже если возникнут ошибки после этого места.
	defer os.Remove(testFilePath)

	// Пытаемся записать небольшой объем данных в тестовый файл.
	// 0644 - стандартные права для файла (чтение/запись для владельца, чтение для остальных).
	testData := []byte("test")
	err := os.WriteFile(testFilePath, testData, 0644)

	// Если возникла ошибка - значит, прав на запись нет.
	if err != nil {
		// Возвращаем ошибку с подробным контекстом.
		return fmt.Errorf("директория '%s' недоступна для записи: %w", dir, err)
	}

	// Если мы дошли до сюда, значит, запись удалась.
	// defer, запланированный ранее, позаботится об удалении файла.
	log.Debug("Права на запись в директорию имеются.")

	return nil
}
