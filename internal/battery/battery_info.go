/**
 * @file battery_info.go
 * @brief Модуль для работы с батареей через IOKit Framework на macOS
 * @details Использует нативный IOKit API для энергоэффективного получения данных
 */

package battery

import (
	"fmt"
	"math"
	"runtime"
)

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

// Объявляем структуру BatteryInfo
typedef struct {
    int currentCapacity;
    int maxCapacity;
    int designCapacity;
    int cycleCount;
    int voltage;
    int amperage;
    int isCharging;
    int isPlugged;
    int timeToEmpty;
    int timeToFull;
} BatteryInfo;

// Объявляем функции из C кода
extern BatteryInfo getBatteryInfo(void);

// Объявляем функции CoreFoundation
typedef struct __CFRunLoop *CFRunLoopRef;
extern void CFRunLoopRun(void);
*/
import "C"

/**
 * @struct BatteryInfo
 * @brief Структура с информацией о батарее
 */
type BatteryInfo struct {
	CurrentCapacity int  // Текущий заряд в процентах
	MaxCapacity     int  // Максимальная емкость
	RawMaxCapacity  int  // Максимальная емкость по данным IORegistry (сырое значение)
	DesignCapacity  int  // Проектная емкость
	CycleCount      int  // Количество циклов зарядки
	Voltage         int  // Напряжение в мВ
	Amperage        int  // Сила тока в мА
	IsCharging      bool // Флаг зарядки
	IsPlugged       bool // Подключено к сети
	TimeToEmpty     int  // Время до разряда в минутах
	TimeToFull      int  // Время до полной зарядки в минутах
	HealthPercent   int  // Здоровье батареи в процентах
}

// Получение информации о батарее
func GetBatteryInfo() (*BatteryInfo, error) {

	// Проверяем, что ОС - macOS (darwin - системное имя macOS в Go).
	if runtime.GOOS != "darwin" {
		return &BatteryInfo{}, fmt.Errorf("чтение реальных данных о батарее поддерживается только на macOS (обнаружена ОС: %s)", runtime.GOOS)
	}

	// Вызываем C функцию для получения данных
	cInfo := C.getBatteryInfo()

	// Создаем указатель на BatteryInfo
	info := &BatteryInfo{
		CurrentCapacity: int(cInfo.currentCapacity),
		MaxCapacity:     int(cInfo.maxCapacity),
		RawMaxCapacity:  int(cInfo.rawMaxCapacity),
		DesignCapacity:  int(cInfo.designCapacity),
		CycleCount:      int(cInfo.cycleCount),
		Voltage:         int(cInfo.voltage),
		Amperage:        int(cInfo.amperage),
		IsCharging:      cInfo.isCharging != 0,
		IsPlugged:       cInfo.isPlugged != 0,
		TimeToEmpty:     int(cInfo.timeToEmpty),
		TimeToFull:      int(cInfo.timeToFull),
	}

	// Приводим значения к процентам, если получены "сырые" единицы (мА·ч).
	maxForPercent := info.MaxCapacity
	if maxForPercent <= 0 && info.RawMaxCapacity > 0 {
		maxForPercent = info.RawMaxCapacity
	}

	if maxForPercent > 0 &&
		(info.CurrentCapacity > maxForPercent || info.CurrentCapacity > 100 || maxForPercent > 100) {
		percent := math.Round(float64(info.CurrentCapacity) * 100 / float64(maxForPercent))
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		info.CurrentCapacity = int(percent)
	}

	// Дополнительная защита от значений вне диапазона.
	if info.CurrentCapacity < 0 {
		info.CurrentCapacity = 0
	}
	if info.CurrentCapacity > 100 {
		info.CurrentCapacity = 100
	}

	// Рассчитываем здоровье батареи
	rawCapacity := info.RawMaxCapacity
	if rawCapacity <= 0 {
		rawCapacity = info.MaxCapacity
	}
	if info.DesignCapacity > 0 && rawCapacity > 0 {
		health := math.Round(float64(rawCapacity) * 100 / float64(info.DesignCapacity))
		if health < 0 {
			health = 0
		}
		if health > 100 {
			health = 100
		}
		info.HealthPercent = int(health)
	}

	// Валидация данных
	if info.CurrentCapacity < 0 || info.MaxCapacity <= 0 {
		return nil, fmt.Errorf("некорректные данные о заряде батареи")
	}

	return info, nil
}
