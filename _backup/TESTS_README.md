# Руководство по тестированию MacBat

Этот документ описывает систему тестирования для модуля мониторинга батареи MacBat.

## Обзор тестов

Система тестирования разделена на несколько категорий:

### 1. Unit тесты (`battery_test.go`)
Основные функциональные тесты с использованием моков:

- **TestLowBatteryThreshold** - Тестирует срабатывание порога низкого заряда
- **TestHighBatteryThreshold** - Тестирует срабатывание порога высокого заряда  
- **TestSmartNotificationInterval** - Проверяет интервальную систему уведомлений (кратность 5%)
- **TestNoRepeatNotifications** - Убеждается что уведомления не повторяются
- **TestChargingStateChange** - Тестирует сброс счетчика при изменении состояния зарядки
- **TestPeriodicCheckInterval** - Проверяет периодическую проверку с заданным интервалом
- **BenchmarkBatteryCheck** - Бенчмарк производительности

### 2. Интеграционные тесты (`integration_test.go`)
Тесты с реальным IOKit API (только на macOS):

- **TestRealBatteryInfo** - Получение реальной информации о батарее
- **TestBatteryInfoConsistency** - Консистентность данных при повторных вызовах
- **TestPowerSourceNotifications** - Система уведомлений об изменениях питания
- **TestBatteryHealthCalculation** - Корректность расчета здоровья батареи
- **TestConcurrentBatteryAccess** - Безопасность многопоточного доступа
- **TestBatteryDataValidation** - Валидность всех полей батареи
- **TestBatteryChangeDetection** - Обнаружение изменений батареи
- **TestEdgeCases** - Граничные случаи (очень низкий/высокий заряд, старая батарея)

### 3. Тесты конфигурации (`config_helper_test.go`)
Тестирование настроек и вспомогательных функций:

- **TestConfigValidation** - Валидация параметров конфигурации
- **TestConfigDefaults** - Проверка значений по умолчанию
- **TestNotificationLevelCalculation** - Расчет уровня уведомлений
- **TestTimingAccuracy** - Точность временных интервалов
- **TestStateTransitions** - Переходы состояний батареи
- **TestBoundaryConditions** - Граничные условия срабатывания
- **TestConfigFileParsing** - Загрузка конфигурации из файла
- **TestMemoryUsage** - Проверка утечек памяти

## Быстрый старт

### Установка зависимостей
```bash
make deps
```

### Запуск всех unit тестов
```bash
make test
# или
go test -v ./... -short
```

### Запуск интеграционных тестов (только macOS)
```bash
make test-integration
# или  
go test -v ./... -tags=integration
```

### Покрытие кода
```bash
make test-coverage
```

## Детальное описание команд

### Основные команды тестирования

| Команда | Описание |
|---------|----------|
| `make test` | Запуск всех unit тестов |
| `make test-integration` | Интеграционные тесты (macOS) |
| `make test-coverage` | Тесты с измерением покрытия |
| `make test-race` | Тесты с детектором гонок |
| `make test-bench` | Бенчмарки производительности |

### Специализированные тесты

| Команда | Описание |
|---------|----------|
| `make test-low-battery` | Только тесты логики низкого заряда |
| `make test-high-battery` | Только тесты логики высокого заряда |
| `make test-smart-notifications` | Тесты умной системы уведомлений |
| `make test-state-changes` | Тесты изменений состояния |
| `make test-validation` | Тесты валидации конфигурации |
| `make test-timing` | Тесты временных интервалов |

### Запуск конкретного теста
```bash
make test-specific TEST=TestLowBatteryThreshold
```

### Отладка и разработка

| Команда | Описание |
|---------|----------|
| `make test-debug` | Подробный вывод тестов |
| `make test-verbose` | Максимально подробный вывод |
| `make quick` | Быстрая проверка (fmt + vet + test) |
| `make dev` | Проверки для разработки |

## Ключевые тестируемые сценарии

### 1. Логика порогов заряда

**Низкий заряд (≤20%):**
- ✅ Без зарядки → уведомление отправляется
- ✅ С зарядкой → уведомление НЕ отправляется

**Высокий заряд (≥80%):**
- ✅ С зарядкой → уведомление отправляется  
- ✅ Без зарядки → уведомление НЕ отправляется

### 2. Интервальная система уведомлений

**Умные уведомления (кратность 5%):**
- Уведомления отправляются только при уровнях: 5%, 10%, 15%, 20%, 25%...
- При уровнях 1%, 2%, 3%, 4%, 6%, 7%... уведомления НЕ отправляются
- Это предотвращает спам при постепенном изменении заряда

### 3. Состояния зарядки

**Сброс счетчика уведомлений:**
- При подключении зарядки счетчик сбрасывается
- При отключении зарядки счетчик сбрасывается
- Это гарантирует уведомление при смене состояния

### 4. Периодические проверки

**Временные интервалы:**
- Проверка батареи происходит с заданным интервалом
- Интервал настраивается в конфигурации
- При использовании IOKit используются события, при fallback - периодический опрос

## Моки и тестовые данные

### MockBatteryInfoProvider
Предоставляет тестовые данные о батарее:
```go
mockBatteryProvider.SetBatteryInfo(BatteryInfo{
    CurrentCapacity: 15,
    IsCharging:      false,
    IsPlugged:       false,
})
```

### MockNotificationSystem  
Перехватывает системные уведомления:
```go
notifications := mockNotificationSystem.GetNotifications()
if len(notifications) > 0 {
    // Проверяем содержимое уведомления
}
```

## Граничные случаи

### Тестируемые границы:
- **Точно на пороге:** 20% и 80%
- **На 1% выше/ниже порога:** 19%, 21%, 79%, 81%
- **Критически низкий заряд:** ≤5%
- **Полный заряд:** ≥95%
- **Очень старая батарея:** >1000 циклов
- **Поврежденная батарея:** здоровье <80%

## Требования к окружению

### Для unit тестов:
- Go 1.19+
- Любая ОС

### Для интеграционных тестов:
- macOS  
- Реальная батарея
- Права доступа к IOKit

### Опциональные инструменты:
- `golangci-lint` для линтинга
- `pprof` для профилирования

## Непрерывная интеграция (CI)

Для CI используйте команду:
```bash
make test-ci
```

Она запускает:
- Unit тесты с детектором гонок
- Измерение покрытия кода
- Исключает интеграционные тесты (не требуют macOS)

## Покрытие кода

Цель покрытия: **≥85%**

Текущие результаты покрытия можно посмотреть:
```bash
make test-coverage
open coverage.html
```

## Профилирование производительности

### CPU профилирование:
```bash
make profile-cpu
go tool pprof cpu.prof
```

### Память профилирование:
```bash  
make profile-mem
go tool pprof mem.prof
```

## Отчеты

Создать полный отчет о тестировании:
```bash
make report
cat test_report.txt
```

## Проблемы и решения

### Тесты не проходят на Linux/Windows
**Решение:** Используйте только unit тесты: `make test`
Интеграционные тесты требуют macOS и IOKit.

### Тесты периодически падают
**Причина:** Race conditions или проблемы с временными интервалами  
**Решение:** Запустите `make test-race` для обнаружения гонок

### Низкое покрытие кода
**Решение:** Добавьте больше граничных случаев в unit тесты

### Медленные тесты
**Решение:** Оптимизируйте временные интервалы в тестах или используйте моки

## Добавление новых тестов

### Новый unit тест:
1. Добавьте в `battery_test.go`
2. Используйте моки для изоляции
3. Проверьте все граничные случаи
4. Добавьте описательное имя теста

### Новый интеграционный тест:
1. Добавьте в `integration_test.go`  
2. Используйте тег `// +build integration`
3. Проверьте работу на macOS
4. Документируйте требования к окружению

### Новый тест конфигурации:
1. Добавьте в `config_helper_test.go`
2. Проверьте валидацию параметров
3. Тестируйте граничные значения
4. Убедитесь в корректности значений по умолчанию

---

**Команда для получения справки:**
```bash
make help
```