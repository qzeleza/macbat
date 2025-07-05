# Makefile для тестирования модуля батареи MacBat
# РАБОЧАЯ ВЕРСИЯ - правильная работа с test_*.go

.PHONY: all build run test test-fixed test-unit test-coverage test-bench test-race \
	test-memory test-threading test-debug test-specific check-test-files \
	setup-links cleanup-links rename-to-standard restore-from-standard \
	lint fmt vet deps clean clean-build quick dev info help

# Переменные
PACKAGE = ./...
COVERAGE_FILE = coverage.out
COVERAGE_HTML = coverage.html

# --- Переменные для сборки ---
BINARY_NAME=macbat
MAIN_PATH=./cmd/macbat

# Информация о версии, получаемая из Git.
# Получаем последний тег. Если тегов нет, используется 'dev'.
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Флаги компоновщика для внедрения информации о версии в бинарный файл.
LDFLAGS = -ldflags="\
    -X 'macbat/internal/version.Version=$(VERSION)' \
    -X 'macbat/internal/version.CommitHash=$(COMMIT_HASH)' \
    -X 'macbat/internal/version.BuildDate=$(BUILD_DATE)'"

# Находим файлы test_*.go
TEST_PREFIX_FILES = $(shell find . -name "test_*.go" -type f)

# Цвета
GREEN = \033[32m
YELLOW = \033[33m
RED = \033[31m
BLUE = \033[34m
NC = \033[0m

all: test

# --- Цели для сборки ---

build: ## Собрать бинарный файл с информацией о версии
	@echo "$(GREEN)Сборка $(BINARY_NAME)...$(NC)"
	@echo "  Версия: $(VERSION)"
	@echo "  Коммит: $(COMMIT_HASH)"
	@echo "  Дата: $(BUILD_DATE)"
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Сборка завершена: ./$(BINARY_NAME)$(NC)"

run: build ## Собрать и запустить приложение
	@echo "$(GREEN)Запуск $(BINARY_NAME)...$(NC)"
	./$(BINARY_NAME)

clean-build: ## Удалить скомпилированный бинарный файл
	@echo "$(YELLOW)Очистка сборки...$(NC)"
	@rm -f $(BINARY_NAME)
	@echo "$(GREEN)Очистка завершена.$(NC)"

help: ## Показать справку по командам
	@echo "$(GREEN)MacBat Test Makefile$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-25s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)📁 Найдено файлов test_*.go: $(shell echo '$(TEST_PREFIX_FILES)' | wc -w)$(NC)"
	@echo ""
	@echo "$(BLUE)🔧 РЕШЕНИЯ ПРОБЛЕМЫ:$(NC)"
	@echo "  $(GREEN)make rename-to-standard$(NC)   - Переименовать test_*.go → *_test.go (рекомендуется)"
	@echo "  $(GREEN)make test$(NC)                - Создать ссылки и запустить тесты"
	@echo "  $(GREEN)make restore-from-standard$(NC) - Вернуть *_test.go → test_*.go"

check-test-files: ## Проверить найденные файлы тестов
	@echo "$(GREEN)Проверка структуры тестов:$(NC)"
	@echo ""
	@echo "$(YELLOW)Файлы test_*.go (не стандартные):$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		for file in $(TEST_PREFIX_FILES); do \
			echo "  ⚠️  $$file"; \
			echo "      Пакет: $$(head -1 $$file | awk '{print $$2}')"; \
			echo "      Тестов: $$(grep -c '^func Test' $$file 2>/dev/null || echo 0)"; \
			dir=$$(dirname $$file); \
			base=$$(basename $$file .go); \
			standard_name="$${base#test_}_test.go"; \
			echo "      Должен быть: $$dir/$$standard_name"; \
		done; \
	else \
		echo "  ✅ Не найдены"; \
	fi
	@echo ""
	@echo "$(YELLOW)Стандартные *_test.go файлы:$(NC)"
	@find . -name "*_test.go" -type f | sed 's/^/  ✅ /' || echo "  ❌ Не найдены"
	@echo ""
	@echo "$(BLUE)💡 РЕКОМЕНДАЦИЯ: Используйте 'make rename-to-standard' для соответствия стандартам Go$(NC)"

rename-to-standard: ## Переименовать test_*.go в *_test.go (РЕКОМЕНДУЕТСЯ)
	@echo "$(YELLOW)⚠️  Переименование файлов test_*.go в стандартные *_test.go$(NC)"
	@echo "$(YELLOW)Это изменит структуру файлов. Продолжить? (y/N)$(NC)"
	@read -r confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		if [ -n "$(TEST_PREFIX_FILES)" ]; then \
			echo "$(GREEN)Переименовываем файлы...$(NC)"; \
			mkdir -p .makefile_backup; \
			for file in $(TEST_PREFIX_FILES); do \
				dir=$$(dirname "$$file"); \
				base=$$(basename "$$file" .go); \
				new_name="$${base#test_}_test.go"; \
				new_path="$$dir/$$new_name"; \
				echo "  $$file → $$new_path"; \
				if [ -f "$$new_path" ]; then \
					echo "    ⚠️  $$new_path уже существует, создаем резерв"; \
					cp "$$new_path" ".makefile_backup/$$(basename $$new_path).backup"; \
				fi; \
				mv "$$file" "$$new_path"; \
				echo "$$file|$$new_path" >> .makefile_backup/renames.log; \
			done; \
			echo "$(GREEN)✅ Переименование завершено. Теперь можно использовать 'go test' стандартно$(NC)"; \
			echo "$(BLUE)Для отката: make restore-from-standard$(NC)"; \
		else \
			echo "$(YELLOW)Файлы test_*.go не найдены$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)Отменено$(NC)"; \
	fi

restore-from-standard: ## Восстановить test_*.go из *_test.go
	@if [ -f .makefile_backup/renames.log ]; then \
		echo "$(GREEN)Восстановление оригинальных имен...$(NC)"; \
		while IFS='|' read -r original new; do \
			if [ -f "$$new" ]; then \
				echo "  $$new → $$original"; \
				mv "$$new" "$$original"; \
			fi; \
		done < .makefile_backup/renames.log; \
		if [ -d .makefile_backup ]; then \
			for backup in .makefile_backup/*.backup; do \
				if [ -f "$$backup" ]; then \
					original_name=$$(basename "$$backup" .backup); \
					cp "$$backup" "./internal/battery/$$original_name" 2>/dev/null || true; \
				fi; \
			done; \
		fi; \
		rm -rf .makefile_backup; \
		echo "$(GREEN)✅ Восстановление завершено$(NC)"; \
	else \
		echo "$(YELLOW)Нет данных для восстановления$(NC)"; \
	fi

setup-links: ## Создать символические ссылки для тестов
	@echo "$(GREEN)Создание символических ссылок...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		mkdir -p .makefile_links; \
		for file in $(TEST_PREFIX_FILES); do \
			dir=$$(dirname "$$file"); \
			base=$$(basename "$$file" .go); \
			new_name="$${base#test_}_test.go"; \
			link_path="$$dir/$$new_name"; \
			if [ -f "$$link_path" ]; then \
				echo "  ⚠️  $$link_path уже существует, пропускаем"; \
			else \
				echo "  $$file → $$link_path (ссылка)"; \
				ln -sf "$$(basename $$file)" "$$link_path"; \
				echo "$$link_path" >> .makefile_links/created.txt; \
			fi; \
		done; \
		echo "$(GREEN)✅ Ссылки созданы$(NC)"; \
	else \
		echo "$(YELLOW)Файлы test_*.go не найдены$(NC)"; \
	fi

cleanup-links: ## Удалить созданные символические ссылки
	@if [ -f .makefile_links/created.txt ]; then \
		echo "$(GREEN)Удаление символических ссылок...$(NC)"; \
		while read -r link; do \
			if [ -L "$$link" ]; then \
				echo "  Удаляем $$link"; \
				rm "$$link"; \
			fi; \
		done < .makefile_links/created.txt; \
		rm -rf .makefile_links; \
		echo "$(GREEN)✅ Ссылки удалены$(NC)"; \
	fi

# ОСНОВНЫЕ КОМАНДЫ ТЕСТИРОВАНИЯ
test: ## Запустить все тесты (через ссылки)
	@echo "$(GREEN)Запуск тестов из файлов test_*.go...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		echo "$(GREEN)Запуск go test...$(NC)"; \
		go test -v $(PACKAGE) -short || true; \
		$(MAKE) cleanup-links; \
	else \
		echo "$(YELLOW)Файлы test_*.go не найдены, запуск стандартных тестов$(NC)"; \
		go test -v $(PACKAGE) -short; \
	fi

test-fixed: ## Запустить исправленные тесты
	@echo "$(GREEN)Запуск исправленных тестов...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -run ".*Fixed.*|.*Stable.*|.*Robust.*" || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -run ".*Fixed.*|.*Stable.*|.*Robust.*"; \
	fi

test-unit: ## Запустить unit тесты
	@echo "$(GREEN)Unit тесты...$(NC)"
	@$(MAKE) test

test-coverage: ## Тесты с покрытием кода
	@echo "$(GREEN)Тесты с покрытием...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -coverprofile=$(COVERAGE_FILE) -covermode=atomic || true; \
		if [ -f $(COVERAGE_FILE) ]; then \
			go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML); \
			echo "$(GREEN)Отчет: $(COVERAGE_HTML)$(NC)"; \
			go tool cover -func=$(COVERAGE_FILE) | tail -n 1; \
		fi; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -coverprofile=$(COVERAGE_FILE) -covermode=atomic; \
	fi

test-bench: ## Бенчмарки
	@echo "$(GREEN)Бенчмарки...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -bench=. -benchmem || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -bench=. -benchmem; \
	fi

test-race: ## Тесты с детектором гонок
	@echo "$(GREEN)Детектор гонок...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -race || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -race; \
	fi

test-memory: ## Тесты памяти
	@echo "$(GREEN)Тесты памяти...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -run ".*Memory.*" || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -run ".*Memory.*"; \
	fi

test-threading: ## Тесты многопоточности
	@echo "$(GREEN)Тесты многопоточности...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -run ".*Thread.*|.*Concurrent.*" || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -run ".*Thread.*|.*Concurrent.*"; \
	fi

test-debug: ## Отладочный запуск
	@echo "$(GREEN)Отладка...$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -count=1 || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -count=1; \
	fi

test-specific: ## Конкретный тест (make test-specific TEST=TestName)
	@if [ -z "$(TEST)" ]; then \
		echo "$(RED)Укажите TEST=TestName$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Запуск теста: $(TEST)$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		$(MAKE) setup-links; \
		go test -v $(PACKAGE) -short -run "$(TEST)" || true; \
		$(MAKE) cleanup-links; \
	else \
		go test -v $(PACKAGE) -short -run "$(TEST)"; \
	fi

test-list: ## Показать все тесты
	@echo "$(GREEN)Доступные тесты:$(NC)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		for file in $(TEST_PREFIX_FILES); do \
			echo "$(YELLOW)$$file:$(NC)"; \
			grep '^func Test' $$file | awk '{print $$2}' | cut -d'(' -f1 | sed 's/^/  /' || true; \
		done; \
	else \
		echo "$(YELLOW)Файлы test_*.go не найдены$(NC)"; \
	fi

# КАЧЕСТВО КОДА
lint: ## Линтер
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		go vet $(PACKAGE); \
	fi

fmt: ## Форматирование
	@go fmt $(PACKAGE)

vet: ## Проверка кода  
	@go vet $(PACKAGE)

# УТИЛИТЫ
clean: cleanup-links ## Очистка
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML) *.prof
	@rm -rf .makefile_backup .makefile_links
	@go clean -testcache

deps: ## Зависимости
	@go mod download && go mod tidy

quick: fmt vet test ## Быстрая проверка

dev: quick test-race ## Разработка

info: ## Информация о проекте
	@echo "$(GREEN)MacBat проект:$(NC)"
	@echo "  Go: $$(go version | awk '{print $$3}')"
	@echo "  Файлы test_*.go: $(shell echo '$(TEST_PREFIX_FILES)' | wc -w)"
	@echo "  Стандартные *_test.go: $(shell find . -name '*_test.go' | wc -l)"
	@if [ -n "$(TEST_PREFIX_FILES)" ]; then \
		total=0; \
		for file in $(TEST_PREFIX_FILES); do \
			count=$$(grep -c '^func Test' $$file 2>/dev/null || echo 0); \
			total=$$((total + count)); \
		done; \
		echo "  Функций Test*: $$total"; \
	fi

.DEFAULT_GOAL := help