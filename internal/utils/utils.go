package utils

// BoolToYesNo преобразует булево значение в "Да" или "Нет"
// @param b bool - булево значение
// @return string - "Да" или "Нет"
func BoolToYesNo(b bool) string {
	if b {
		return "да"
	}
	return "нет"
}
