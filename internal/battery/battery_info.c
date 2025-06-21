/**
 * @file battery.c
 * @brief Реализация функций для работы с батареей через IOKit Framework на macOS
 */

#include "battery_info.h"

// Получение информации о батарее через IOKit
BatteryInfo getBatteryInfo()
{
    BatteryInfo info = {0};

    // Получаем информацию об источниках питания
    CFTypeRef powerInfo = IOPSCopyPowerSourcesInfo();
    if (!powerInfo)
        return info;

    CFArrayRef powerSources = IOPSCopyPowerSourcesList(powerInfo);
    if (!powerSources)
    {
        CFRelease(powerInfo);
        return info;
    }

    // Ищем внутреннюю батарею
    CFIndex count = CFArrayGetCount(powerSources);
    for (CFIndex i = 0; i < count; i++)
    {
        CFTypeRef powerSource = CFArrayGetValueAtIndex(powerSources, i);
        CFDictionaryRef description = IOPSGetPowerSourceDescription(powerInfo, powerSource);
        if (!description)
            continue;

        // Проверяем тип источника питания
        CFTypeRef type = CFDictionaryGetValue(description, CFSTR(kIOPSTypeKey));
        if (type && CFStringCompare(type, CFSTR(kIOPSInternalBatteryType), 0) == kCFCompareEqualTo)
        {
            // Текущий заряд
            CFNumberRef currentCap = CFDictionaryGetValue(description, CFSTR(kIOPSCurrentCapacityKey));
            if (currentCap)
                CFNumberGetValue(currentCap, kCFNumberIntType, &info.currentCapacity);

            // Максимальная емкость
            // CFNumberRef maxCap = CFDictionaryGetValue(description, CFSTR(kIOPSMaxCapacityKey));
            // if (maxCap)
            //     CFNumberGetValue(maxCap, kCFNumberIntType, &info.maxCapacity);

            // Статус зарядки
            CFTypeRef isCharging = CFDictionaryGetValue(description, CFSTR(kIOPSIsChargingKey));
            if (isCharging == kCFBooleanTrue)
                info.isCharging = 1;

            // Источник питания
            CFTypeRef powerSourceState = CFDictionaryGetValue(description, CFSTR(kIOPSPowerSourceStateKey));
            if (powerSourceState)
            {
                if (CFStringCompare(powerSourceState, CFSTR(kIOPSACPowerValue), 0) == kCFCompareEqualTo)
                {
                    info.isPlugged = 1;
                }
            }

            // Время до разряда
            CFNumberRef timeToEmpty = CFDictionaryGetValue(description, CFSTR(kIOPSTimeToEmptyKey));
            if (timeToEmpty)
                CFNumberGetValue(timeToEmpty, kCFNumberIntType, &info.timeToEmpty);

            // Время до полной зарядки
            CFNumberRef timeToFull = CFDictionaryGetValue(description, CFSTR(kIOPSTimeToFullChargeKey));
            if (timeToFull)
                CFNumberGetValue(timeToFull, kCFNumberIntType, &info.timeToFull);

            break;
        }
    }

    CFRelease(powerSources);
    CFRelease(powerInfo);

    // Получаем дополнительную информацию из IORegistry
    io_registry_entry_t powerSource = IOServiceGetMatchingService(kIOMainPortDefault,
                                                                  IOServiceMatching("IOPMPowerSource"));

    if (powerSource != 0)
    {
        CFMutableDictionaryRef properties = NULL;
        if (IORegistryEntryCreateCFProperties(powerSource, &properties,
                                              kCFAllocatorDefault, kNilOptions) == KERN_SUCCESS)
        {

            // Максимальная емкость батареи в mAh, которую она может удерживать в текущем состоянии.
            CFNumberRef maxCap = CFDictionaryGetValue(properties, CFSTR("AppleRawMaxCapacity"));
            if (maxCap)
                CFNumberGetValue(maxCap, kCFNumberIntType, &info.maxCapacity);

            // Проектная емкость
            CFNumberRef designCap = CFDictionaryGetValue(properties, CFSTR("DesignCapacity"));
            if (designCap)
                CFNumberGetValue(designCap, kCFNumberIntType, &info.designCapacity);

            // Количество циклов
            CFNumberRef cycles = CFDictionaryGetValue(properties, CFSTR("CycleCount"));
            if (cycles)
                CFNumberGetValue(cycles, kCFNumberIntType, &info.cycleCount);

            // Напряжение
            CFNumberRef voltage = CFDictionaryGetValue(properties, CFSTR("Voltage"));
            if (voltage)
                CFNumberGetValue(voltage, kCFNumberIntType, &info.voltage);

            // Сила тока
            CFNumberRef amperage = CFDictionaryGetValue(properties, CFSTR("Amperage"));
            if (amperage)
                CFNumberGetValue(amperage, kCFNumberIntType, &info.amperage);

            CFRelease(properties);
        }
        IOObjectRelease(powerSource);
    }

    return info;
}

#ifdef __cplusplus
extern "C"
{
#endif
#ifdef __cplusplus
}
#endif