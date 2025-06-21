/**
 * @file battery_info.go
 * @brief Модуль для работы с батареей через IOKit Framework на macOS
 * @details Использует нативный IOKit API для энергоэффективного получения данных
 */

package battery

import (
	"fmt"
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
		DesignCapacity:  int(cInfo.designCapacity),
		CycleCount:      int(cInfo.cycleCount),
		Voltage:         int(cInfo.voltage),
		Amperage:        int(cInfo.amperage),
		IsCharging:      cInfo.isCharging != 0,
		IsPlugged:       cInfo.isPlugged != 0,
		TimeToEmpty:     int(cInfo.timeToEmpty),
		TimeToFull:      int(cInfo.timeToFull),
	}

	// Рассчитываем здоровье батареи
	if info.DesignCapacity > 0 {
		info.HealthPercent = int(float64(info.MaxCapacity) * 100 / float64(info.DesignCapacity))
	}

	// Валидация данных
	if info.CurrentCapacity < 0 || info.MaxCapacity <= 0 {
		return nil, fmt.Errorf("некорректные данные о заряде батареи")
	}

	return info, nil
}
