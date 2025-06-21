/**
 * @file termwindow.go
 * @brief Модуль для создания текстовых окон с псевдографикой в терминале
 * @author Generated
 * @date 2025
 *
 * Данный модуль предоставляет возможности для создания красивых текстовых окон
 * в терминале с использованием Unicode псевдографики (двойные линии). Поддерживает цветной текст,
 * автоматическое форматирование и выравнивание, горизонтальные разделители.
 */

package utils

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"unicode/utf8"
)

// Регулярное выражение для поиска ANSI цветовых кодов
var ansiColorRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Константы для форматирования
const (
	LeftMargin    = 2   // Отступ от левой границы окна
	RightMargin   = 2   // Отступ от правой границы окна
	ValueGap      = 7   // Отступ значения от самого длинного параметра
	BorderWidth   = 2   // Ширина границ окна (левая + правая)
	DividerSymbol = "-" // Символ для обозначения разделителя в AddLine
)

// ANSI коды цветов для текста
const (
	ColorReset  = "\033[0m"  // Сброс цвета
	ColorRed    = "\033[31m" // Красный
	ColorGreen  = "\033[32m" // Зеленый
	ColorYellow = "\033[33m" // Желтый
	ColorBlue   = "\033[34m" // Синий
	ColorPurple = "\033[35m" // Фиолетовый
	ColorCyan   = "\033[36m" // Голубой
	ColorWhite  = "\033[37m" // Белый
	ColorBold   = "\033[1m"  // Жирный
)

// Unicode символы для построения рамки окна (двойные линии)
const (
	BoxTopLeft     = "╔" // Левый верхний угол
	BoxTopRight    = "╗" // Правый верхний угол
	BoxBottomLeft  = "╚" // Левый нижний угол
	BoxBottomRight = "╝" // Правый нижний угол
	BoxHorizontal  = "═" // Горизонтальная линия
	BoxDivider     = "─" // Горизонтальный разделитель
	BoxVertical    = "║" // Вертикальная линия
	BoxCrossLeft   = "╠" // Пересечение с левой стороны
	BoxCrossRight  = "╣" // Пересечение с правой стороны
)

/**
 * @brief Структура элемента буфера
 *
 * Представляет один элемент в буфере окна - либо строку с параметром и значением,
 * либо горизонтальный разделитель.
 */
type BufferItem struct {
	Parameter string // Название параметра
	Value     string // Значение параметра
	Color     string // ANSI код цвета для значения
	IsDivider bool   // Флаг: является ли элемент разделителем
}

/**
 * @brief Основная структура буфера окна
 *
 * Содержит все элементы буфера и настройки для отображения окна.
 */
type WindowBuffer struct {
	items       []BufferItem // Массив элементов буфера
	minWidth    int          // Минимальная ширина окна
	maxParamLen int          // Максимальная длина параметра в буфере
}

/**
 * @brief Создает новый буфер окна
 * @param minWidth Минимальная ширина окна в символах
 * @return Указатель на новый экземпляр WindowBuffer
 *
 * Инициализирует новый буфер окна с заданной минимальной шириной.
 * Если содержимое требует большей ширины, окно будет расширено автоматически.
 */
func NewWindowBuffer(minWidth int) *WindowBuffer {
	return &WindowBuffer{
		items:       make([]BufferItem, 0),
		minWidth:    minWidth,
		maxParamLen: 0,
	}
}

/**
 * @brief Добавляет строку с параметром и значением в буфер
 * @param parameter Название параметра (если "-", то добавляется разделитель)
 * @param value Значение параметра
 * @param color ANSI код цвета для значения (может быть пустым)
 *
 * Добавляет новую строку в буфер окна. Если в качестве параметра передан символ "-",
 * то вместо строки добавляется горизонтальный разделитель.
 * Автоматически обновляет максимальную длину параметра для правильного выравнивания.
 */
func (wb *WindowBuffer) AddLine(parameter, value, color string) {
	// Проверяем, является ли это запросом на добавление разделителя
	if parameter == DividerSymbol {
		wb.items = append(wb.items, BufferItem{
			IsDivider: true,
		})
		return
	}

	// Обновляем максимальную длину параметра
	// Используем utf8.RuneCountInString для корректного подсчета символов в Unicode
	paramLen := utf8.RuneCountInString(parameter)
	if paramLen > wb.maxParamLen {
		wb.maxParamLen = paramLen
	}

	wb.items = append(wb.items, BufferItem{
		Parameter: parameter,
		Value:     value,
		Color:     color,
		IsDivider: false,
	})
}

/**
 * @brief Добавляет горизонтальный разделитель в буфер
 *
 * Добавляет горизонтальную линию-разделитель, которая будет отображена
 * как линия, соединяющая левую и правую границы окна.
 */
func (wb *WindowBuffer) AddDivider() {
	wb.items = append(wb.items, BufferItem{
		IsDivider: true,
	})
}

/**
 * @brief Вычисляет необходимую ширину окна
 * @return Ширина окна в символах
 *
 * Анализирует все элементы буфера и вычисляет минимальную ширину окна,
 * необходимую для корректного отображения всего содержимого.
 * Учитывает отступы, границы и самую длинную строку.
 */
func (wb *WindowBuffer) calculateWindowWidth() int {
	maxContentWidth := 0

	// Проходим по всем элементам и находим максимальную ширину содержимого
	for _, item := range wb.items {
		if !item.IsDivider {
			// Вычисляем: левый отступ + длина параметра + отступ + длина значения + правый отступ
			contentWidth := LeftMargin + utf8.RuneCountInString(item.Parameter) + ValueGap + utf8.RuneCountInString(item.Value) + RightMargin
			if contentWidth > maxContentWidth {
				maxContentWidth = contentWidth
			}
		}
	}

	// Добавляем ширину границ (левая и правая)
	totalWidth := maxContentWidth + BorderWidth

	// Возвращаем максимум из вычисленной и минимальной ширины
	if totalWidth < wb.minWidth {
		return wb.minWidth
	}
	return totalWidth
}

/**
 * @brief Форматирует одну строку согласно ширине окна
 * @param item Элемент буфера для форматирования
 * @param windowWidth Ширина окна в символах
 * @return Отформатированная строка с границами
 *
 * Преобразует элемент буфера в отформатированную строку с правильным
 * выравниванием, отступами и границами. Для разделителей создает
 * горизонтальную линию.
 */
func (wb *WindowBuffer) formatLine(item BufferItem, windowWidth int) string {

	if item.IsDivider {
		// Создаем горизонтальный разделитель
		innerWidth := windowWidth - BorderWidth // Вычитаем символы границ
		return BoxCrossLeft + strings.Repeat(BoxDivider, innerWidth) + BoxCrossRight
	}

	// Формат строки: "║  параметр     значение  ║"
	//                 ^2  ^maxParam+5  ^value ^2^

	// Левый отступ
	leftPadding := strings.Repeat(" ", LeftMargin)

	// Параметр выравнивается по левому краю
	paramFormatted := item.Parameter

	// Отступ: ValueGap символов от самого длинного параметра
	// Используем utf8.RuneCountInString для корректного подсчета символов в Unicode
	paramLen := utf8.RuneCountInString(item.Parameter)
	gap := strings.Repeat(" ", wb.maxParamLen+ValueGap-paramLen)

	// Значение
	valueFormatted := item.Value

	// Вычисляем оставшееся место для правого отступа
	// Используем длину строки без цветовых кодов для корректного расчета
	cleanValue := stripAnsiCodes(valueFormatted)
	// Учитываем, что правый отступ уже включен в contentWidth
	contentLen := utf8.RuneCountInString(leftPadding) + utf8.RuneCountInString(stripAnsiCodes(paramFormatted)) + utf8.RuneCountInString(gap) + utf8.RuneCountInString(cleanValue)
	if contentLen > windowWidth {
		windowWidth = contentLen + BorderWidth
	}

	// Вычисляем ширину внутреннего содержимого (без учета границ)
	// windowWidth уже включает в себя левую и правую границы
	innerWidth := windowWidth - BorderWidth // Вычитаем только границы

	// Убедимся, что не пытаемся создать отрицательное количество пробелов
	// Учитываем правую границу (1 символ) при расчете отступа
	var rightPadding string
	paddingNeeded := innerWidth - contentLen
	if paddingNeeded > 0 {
		rightPadding = strings.Repeat(" ", paddingNeeded)
	} else {
		// Если места не хватает, не добавляем отступ
		rightPadding = " "
	}

	// Применяем цвет, если указан
	if item.Color != "" {
		valueFormatted = item.Color + valueFormatted + ColorReset
	}

	// Собираем строку с учетом правого отступа и границы
	return BoxVertical + leftPadding + paramFormatted + gap + valueFormatted + rightPadding + BoxVertical
}

/**
 * @brief Выводит сформированное окно в терминал
 *
 * Отображает полное окно с рамкой, содержимым и разделителями в терминале.
 * Автоматически вычисляет оптимальную ширину и форматирует все строки.
 * Если буфер пуст, ничего не выводит.
 */
func (wb *WindowBuffer) PrintBox() {
	if len(wb.items) == 0 {
		return
	}

	windowWidth := wb.calculateWindowWidth()
	innerWidth := windowWidth - BorderWidth

	// Верхняя граница окна
	fmt.Println(BoxTopLeft + strings.Repeat(BoxHorizontal, innerWidth) + BoxTopRight)

	// Строки содержимого
	for _, item := range wb.items {
		fmt.Println(wb.formatLine(item, windowWidth))
	}

	// Нижняя граница окна
	fmt.Println(BoxBottomLeft + strings.Repeat(BoxHorizontal, innerWidth) + BoxBottomRight)
}

/**
 * @brief Очищает буфер окна
 *
 * Удаляет все элементы из буфера и сбрасывает максимальную длину параметра.
 * Полезно для повторного использования буфера с новым содержимым.
 */
func (wb *WindowBuffer) Clear() {
	wb.items = make([]BufferItem, 0)
	wb.maxParamLen = 0
}

/**
 * @brief Устанавливает минимальную ширину окна
 * @param width Новая минимальная ширина в символах
 *
 * Изменяет минимальную ширину окна. Влияет на последующие вызовы Render().
 */
func (wb *WindowBuffer) SetMinWidth(width int) {
	wb.minWidth = width
}

/**
 * @brief Возвращает текущее количество элементов в буфере
 * @return Количество элементов в буфере
 */
func (wb *WindowBuffer) GetItemCount() int {
	return len(wb.items)
}

/**
 * @brief Возвращает текущую минимальную ширину окна
 * @return Минимальная ширина окна в символах
 */
func (wb *WindowBuffer) GetMinWidth() int {
	return wb.minWidth
}

/**
 * @brief Удаляет ANSI цветовые коды из строки
 * @param s Входная строка с цветовыми кодами
 * @return Строка без цветовых кодов
 */
func stripAnsiCodes(s string) string {
	return ansiColorRegex.ReplaceAllString(s, "")
}

// GetTerminalWidth возвращает ширину терминала в символах
// @return int Ширина терминала в символах или 80 по умолчанию
func GetTerminalWidth() int {
	// Сначала пробуем получить размер через syscall
	ws := &struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}{}

	r1, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(ws)),
	)

	if r1 == 0 && ws.Col > 0 {
		return int(ws.Col)
	}

	// Если не сработало, пробуем через переменную окружения
	if colsStr := os.Getenv("COLUMNS"); colsStr != "" {
		if cols, err := strconv.Atoi(colsStr); err == nil && cols > 0 {
			return cols
		}
	}

	// Если ничего не помогло, возвращаем разумное значение по умолчанию
	return 80
}
