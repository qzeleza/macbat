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

# Файл скрипта для авто-инкремента номера сборки
BUILD_SCRIPT = ./scripts/update_build_number.sh

# Информация о версии, получаемая из Git.
# Получаем последний тег. Если тегов нет, используется 'dev'.
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Получаем номер сборки (скрипт обновляет внутренний счётчик)
BUILD_NUMBER := $(shell bash $(BUILD_SCRIPT) $(VERSION))
# Добавляем номер сборки к версии
VERSION := $(VERSION)+$(BUILD_NUMBER)

# Путь модуля
MODULE_PATH = github.com/qzeleza/macbat

# Флаги компоновщика для внедрения информации о версии в бинарный файл.
LDFLAGS = -ldflags="\
    -X '$(MODULE_PATH)/internal/version.Version=$(VERSION)' \
    -X '$(MODULE_PATH)/internal/version.CommitHash=$(COMMIT_HASH)' \
    -X '$(MODULE_PATH)/internal/version.BuildDate=$(BUILD_DATE)' \
    -X '$(MODULE_PATH)/internal/version.BuildNumber=$(BUILD_NUMBER)'"

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
# --- Настройки GitHub ---
REPO          = qzeleza/macbat          # owner/repo на GitHub
GH            ?= gh                     # GitHub CLI
RELEASE_TITLE ?= "MacBat $(VERSION)"

# Цель: publish – полный цикл публикации релиза на GitHub
# 1. Сборка релизного бинарника (make release)
# 2. Создание тега версии и пуш в origin
# 3. Создание релиза через gh cli и загрузка бинарника
# 4. Формирование tar.gz исходников, вычисление sha256
# 5. Обновление Homebrew formula macbat.rb (version + sha256)
# 6. Коммит formula и пуш в origin
# Требования: установлен GitHub CLI (`gh`) и переменная окружения GH_TOKEN с правами на репозиторий.
publish: release ## Сформировать релиз, выложить на GitHub и обновить Homebrew formula
    @echo "$(YELLOW)▶️  Публикация релиза $(VERSION)$(NC)"
    @if ! $(GH) auth status >/dev/null 2>&1; then \
        echo "$(RED)🔑 GH CLI не авторизован. Выполните 'gh auth login' или задайте GH_TOKEN$(NC)"; exit 1; fi
    # --- Git tag ---
    @git tag -a $(VERSION) -m "Публикация релиза $(VERSION)" || true
    @git push origin $(VERSION)
    # --- GitHub release ---
    $(GH) release create $(VERSION) ./$(BINARY_NAME) \
      --repo $(REPO) \
      --title $(RELEASE_TITLE) \
      --notes "Автоматический релиз $(VERSION), build $(BUILD_NUMBER)"
    # --- Source tarball & sha256 ---
    @git archive --format=tar.gz --prefix=macbat-$(VERSION)/ $(VERSION) -o macbat-$(VERSION).tar.gz
    @SHA=$$(shasum -a 256 macbat-$(VERSION).tar.gz | awk '{print $$1}'); \
        sed -i '' -e "s/^  url \\".*\\"/  url \"https:\/\/github.com\/$(REPO)\/archive\/refs\/tags\/$(VERSION).tar.gz\"/" macbat.rb; \
        sed -i '' -e "s/^  version \".*\"/  version \"$(VERSION)\"/" macbat.rb; \
        sed -i '' -e "s/^  sha256 \".*\"/  sha256 \"$$SHA\"/" macbat.rb; \
        echo "$(GREEN)Формула macbat.rb обновлена (sha256=$$SHA)$(NC)";
    # --- Commit formula ---
    @git add macbat.rb
    @git commit -m "brew formula: update to $(VERSION) ($$SHA)" || true
    @git push origin HEAD
    @echo "$(GREEN)✅ Релиз $(VERSION) опубликован$(NC)"

build: ## Собрать бинарный файл с информацией о версии
	@echo "$(GREEN)Сборка $(BINARY_NAME)...$(NC)"
	@echo "  Версия: $(VERSION)"
	@echo "  Коммит: $(COMMIT_HASH)"
	@echo "  Дата: $(BUILD_DATE)"
	@echo "  Номер сборки: $(BUILD_NUMBER)"
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Сборка завершена: ./$(BINARY_NAME)$(NC)"

run: build ## Собрать и запустить приложение для разработки
	@echo "$(YELLOW)Удаляем запущенные процессы $(BINARY_NAME)...$(NC)"
	killall $(BINARY_NAME) 2>/dev/null || true
	@echo "$(GREEN)Запуск $(BINARY_NAME) в режиме разработки...$(NC)"
	./$(BINARY_NAME)
	@echo "$(CYAN)Просмотр логов:$(NC)"
	./$(BINARY_NAME) --log 
	@echo "$(CYAN)Проверка запущенных процессов:$(NC)"
	ps -ax | grep -v grep | grep '/$(BINARY_NAME)' --color=always

release: clean
	@echo "$(YELLOW)Сборка $(BINARY_NAME) для релиза...$(NC)"
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "$(GREEN)Сборка завершена: ./$(BINARY_NAME)$(NC)"

install: clean release
	@echo "$(YELLOW)Установка $(BINARY_NAME) в /usr/local/bin...$(NC)"
	@cp ./$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)Установка завершена.$(NC)"

clean-build: ## Удалить скомпилированный бинарный файл
	@echo "$(YELLOW)Очистка сборки...$(NC)"
	@rm -f $(BINARY_NAME)
	@echo "$(GREEN)Очистка завершена.$(NC)"

help: ## Показать справку по командам
    @echo "$(GREEN)MacBat Makefile$(NC)"
    @echo ""
    # Авто-генерируем список целей с описаниями
    @grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS=":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
    @echo ""
    @echo "$(CYAN)Часто используемые цели:$(NC)"
    @echo "  $(GREEN)make run$(NC)       – сборка и запуск приложения"
    @echo "  $(GREEN)make release$(NC)   – сборка релизного бинарника"
    @echo "  $(GREEN)make install$(NC)   – установка бинарника в /usr/local/bin"
    @echo "  $(GREEN)make clean$(NC)     – полная очистка артефактов"
    @echo "  $(GREEN)make test$(NC)      – запуск всех тестов"
    @echo "  $(GREEN)make info$(NC)      – информация о проекте"
	@echo ""
	@echo "$(CYAN)Дополнительные цели:$(NC)"
	@echo "  $(GREEN)make deps$(NC)      – установка зависимостей"
	@echo "  $(GREEN)make quick$(NC)     – быстрая проверка"
	@echo "  $(GREEN)make dev$(NC)       – разработка"
	@echo "  $(GREEN)make fmt$(NC)       – форматирование кода"
	@echo "  $(GREEN)make vet$(NC)       – проверка кода"
	@echo "  $(GREEN)make test-fixed$(NC) – запуск исправленных тестов"
	@echo "  $(GREEN)make test-unit$(NC)  – запуск unit тестов"
	@echo "  $(GREEN)make test-coverage$(NC) – тесты с отчетом о покрытии"
	@echo "  $(GREEN)make test-race$(NC)   – тесты с проверкой гонок"
	@echo "  $(GREEN)make test-specific TEST=X$(NC) – запуск конкретного теста"
	@echo ""
	@echo "$(CYAN)Дополнительные цели:$(NC)"
	@echo "  $(GREEN)make profile-cpu$(NC) – CPU профилирование"
	@echo "  $(GREEN)make profile-mem$(NC) – профилирование памяти"


# УТИЛИТЫ
clean: clean-build cleanup-links ## Очистка
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


.DEFAULT_GOAL := help