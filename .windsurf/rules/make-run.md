---
trigger: always_on
description: *.go
---

Для работы с проектом используй утилиту make и написанный для работы Makefile, который имеет следующие цели:

# Авто-генерируем список целей с описаниями
  del-tag              Удалить указанный тег TAG=<tag>
  next-tag             Сформировать новый тег (увеличивает PATCH на 1) и запушить
  publish              Сформировать релиз, выложить на GitHub и обновить Homebrew formula
  build                Собрать бинарный файл с информацией о версии
  run                  Собрать и запустить приложение для разработки
  clean-build          Удалить скомпилированный бинарный файл
  help                 Показать справку по командам
  clean                Очистка
  deps                 Зависимости
  quick                Быстрая проверка
  dev                  Разработка
  info                 Информация о проекте
  test                 Запустить все тесты (через ссылки)
  test-fixed           Запустить исправленные тесты
  test-unit            Запустить unit тесты
  test-coverage        Тесты с покрытием кода
  test-bench           Бенчмарки
  test-race            Тесты с детектором гонок
  test-memory          Тесты памяти
  test-threading       Тесты многопоточности
  test-debug           Отладочный запуск
  test-specific        Конкретный тест (make test-specific TEST=TestName)
  test-list            Показать все тесты
  lint                 Линтер
  fmt                  Форматирование
  vet                  Проверка кода  

Часто используемые цели:
  make run       – сборка и запуск приложения
  make release   – сборка релизного бинарника
  make install   – установка бинарника в /usr/local/bin
  make clean     – полная очистка артефактов
  make test      – запуск всех тестов
  make info      – информация о проекте

Дополнительные цели:
  make deps      – установка зависимостей
  make quick     – быстрая проверка
  make dev       – разработка
  make fmt       – форматирование кода
  make vet       – проверка кода
  make test-fixed – запуск исправленных тестов
  make test-unit  – запуск unit тестов
  make test-coverage – тесты с отчетом о покрытии
  make test-race   – тесты с проверкой гонок
  make test-specific TEST=X – запуск конкретного теста

Дополнительные цели:
  make profile-cpu – CPU профилирование
  make profile-mem – профилирование памяти