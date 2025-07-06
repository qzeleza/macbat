package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/qzeleza/macbat/internal/logger"
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

// BoolToYesNo преобразует булево значение в "Да" или "Нет"
// @param b bool - булево значение
// @return string - "Да" или "Нет"
func BoolToYesNo(b bool) string {
	if b {
		return "да"
	}
	return "нет"
}

func GetMaxLabelLength(labels []string) int {
	maxLength := 0
	for _, label := range labels {
		length := utf8.RuneCountInString(label)
		if length > maxLength {
			maxLength = length
		}
	}
	return maxLength
}

// FormatTimeToHMS конвертирует количество секунд в строку
// «HHч MMм SSс» или «MMм SSс», если часов = 0.
//
// @param seconds Количество секунд
// @return string Строка формата «1ч 23м 45с» либо «05м 12с»
func FormatTimeToColonHMS(minutes int) string {
	if minutes < 0 {
		minutes = 0
	}
	h := minutes / 60
	m := minutes % 60
	return fmt.Sprintf("%02dч %02dм", h, m)
}

// ExtractMenuItemText извлекает текст из входной строки, находящийся в кавычках.
// Удаляет все не-ASCII символы (например, эмодзи) и приводит пробелы к единому формату.
// Возвращает очищенный текст или пустую строку, если текст в кавычках не найден.
//
// @param input string - входная строка, содержащая текст в кавычках.
// @return string - очищенный текст без не-ASCII символов и с нормализованными пробелами.
func ExtractMenuItemText(input string) string {
	// Найти текст в кавычках
	re := regexp.MustCompile(`"([^"]*)"`)
	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return ""
	}

	text := matches[1]

	// Убрать эмодзи (все не-ASCII символы)
	var result []rune
	for _, r := range text {
		if r <= 127 { // ASCII символы
			result = append(result, r)
		}
	}

	// Преобразовать обратно в строку и очистить пробелы
	cleaned := string(result)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}
