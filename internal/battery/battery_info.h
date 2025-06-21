/**
 * @file battery.h
 * @brief Заголовочный файл для работы с батареей через IOKit Framework на macOS
 */

#ifndef BATTERY_H
#define BATTERY_H

#include <IOKit/IOKitLib.h>
#include <IOKit/ps/IOPowerSources.h>
#include <IOKit/ps/IOPSKeys.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdio.h>

typedef struct
{
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

// Функции для работы с батареей
BatteryInfo getBatteryInfo(void);
CFRunLoopRef getRunLoop(void);
void powerSourceChanged(void *context);

#ifdef __cplusplus
extern "C"
{
#endif
    int isPowerNotificationActive(void);
#ifdef __cplusplus
}
#endif

#endif // BATTERY_H