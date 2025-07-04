package utils

import "unicode/utf8"

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
