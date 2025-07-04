package utils

import (
	"strings"
	"unicode/utf8"
)

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

// FormatMenuLine форматирует строку для меню, выравнивая значение по правому краю.
// @param label string - метка (текст слева)
// @param value string - значение (текст справа)
// @param totalWidth int - общая желаемая ширина строки в рунах
// @return string - отформатированная строка
func FormatMenuLine(label, value string, totalWidth int) string {
	labelWidth := utf8.RuneCountInString(label)
	valueWidth := utf8.RuneCountInString(value)

	// Рассчитываем количество пробелов для вставки
	paddingWidth := totalWidth - labelWidth - valueWidth
	if paddingWidth < 1 {
		paddingWidth = 1 // Гарантируем как минимум один пробел
	}

	// Собираем строку: "Метка" + "...пробелы..." + "Значение"
	return label + strings.Repeat(" ", paddingWidth) + value
}
