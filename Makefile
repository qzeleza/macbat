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
# VERSION := $(VERSION)+$(BUILD_NUMBER)

# Путь модуля
MODULE_PATH = github.com/qzeleza/macbat

# Значение флага -ldflags для внедрения информации о версии.
LDFLAGS = "\
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
REPO = qzeleza/macbat
TAP_REPO = qzeleza/homebrew-macbat
GH ?= gh                     
RELEASE_TITLE = "MacBat $(VERSION)"

# Цель: del-tag – удалить произвольный тег локально и в origin.
# Использование: make del-tag TAG=v2.1.1
del-tag: ## Удалить указанный тег TAG=<tag>
	@if [ -z "$(TAG)" ]; then \
		echo "$(RED)Не указан TAG. Пример: make del-tag TAG=v2.1.1$(NC)"; exit 1; fi
	@echo "$(YELLOW)Удаление тега $(TAG)...$(NC)"
	@git tag -d $(TAG) 2>/dev/null || true
	@git push --delete origin $(TAG) 2>/dev/null || true
	@echo "$(GREEN)Тег $(TAG) удалён локально и на origin$(NC)"

# Цель: tag – создание тега версии и пуш в origin
next-tag: ## Сформировать новый тег (увеличивает PATCH на 1) и запушить
	# берём последний корректный тег вида vX.Y.Z
	@PREV=$$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1); \
	if [ -z "$$PREV" ]; then PREV=v0.0.0; fi; \
	echo "$(YELLOW)Предыдущий тег: $$PREV$(NC)"; \
	NEW=$$(echo $$PREV | sed 's/^v//' | awk -F. '{OFS="."; $$NF=$$NF+1; print $$0}'); \
	NEW_TAG=v$$NEW; \
	if git rev-parse $(NEW_TAG) >/dev/null 2>&1; then \
		echo "$(YELLOW)Тег $(NEW_TAG) уже существует – удаляем локально и в origin...$(NC)"; \
		git tag -d $(NEW_TAG) >/dev/null; \
		git push --delete origin $(NEW_TAG) >/dev/null 2>&1 || true; \
		$(GH) release delete $(NEW_TAG) --yes >/dev/null 2>&1 || true; \
	fi; \
	echo "$(GREEN)Новый тег: $$NEW_TAG$(NC)"; \
	git tag -a $$NEW_TAG -m "Выпуск $$NEW_TAG"; \
	git push origin $$NEW_TAG; \
	echo "$(GREEN)Тег $$NEW_TAG создан и отправлен$(NC)"

# Цель: publish – полный цикл публикации релиза на GitHub
# 1. Сборка релизного бинарника (make release)
# 2. Создание тега версии и пуш в origin
# 3. Создание релиза через gh cli и загрузка бинарника
# 4. Формирование tar.gz исходников, вычисление sha256
# 5. Обновление Homebrew formula macbat.rb (version + sha256)
# 6. Коммит formula и пуш в origin
# Требования: установлен GitHub CLI (`gh`) и переменная окружения GH_TOKEN с правами на репозиторий.
publish: release next-tag ## Сформировать релиз, выложить на GitHub и обновить Homebrew formula
	
	@echo "$(YELLOW)▶️  Публикация релиза $(VERSION)$(NC)"
	@if ! $(GH) auth status >/dev/null 2>&1; then \
		echo "$(RED)🔑 GH CLI не авторизован. Выполните 'gh auth login' или задайте GH_TOKEN$(NC)"; exit 1; fi
	

	
	# --- GitHub release ---
	@echo "$(YELLOW)Создание релиза $(VERSION)...$(NC)"
	$(GH) release create $(VERSION) ./$(BINARY_NAME) \
		--repo $(REPO) \
		--title $(RELEASE_TITLE) \
		--notes "Автоматический релиз $(VERSION), build $(BUILD_NUMBER)"
	
	@echo "$(YELLOW)Сборка и упаковка бинарников для macOS (amd64/arm64)...$(NC)"; \
	mkdir -p dist; \
	for ARCH in amd64 arm64; do \
		GOOS=darwin GOARCH=$$ARCH CGO_ENABLED=1 go build -ldflags=$(LDFLAGS) -o dist/macbat $$PWD/$(MAIN_PATH); \
		tar -czf dist/macbat-darwin-$$ARCH.tar.gz -C dist macbat; \
		echo "$(CYAN)– $$ARCH собран$(NC)"; \
		$(GH) release upload $(VERSION) dist/macbat-darwin-$$ARCH.tar.gz --clobber; \
		mv dist/macbat dist/macbat-$$ARCH; \
	done; \
	SHA_AMD64=$$(shasum -a 256 dist/macbat-darwin-amd64.tar.gz | awk '{print $$1}'); \
	SHA_ARM64=$$(shasum -a 256 dist/macbat-darwin-arm64.tar.gz | awk '{print $$1}'); \
	echo "$(YELLOW)Генерация новой Homebrew formula $(BINARY_NAME).rb...$(NC)"; \
	FORMULA_TMP=$$(mktemp); \
	printf 'class Macbat < Formula\n' > $$FORMULA_TMP; \
	printf '  desc "Утилита мониторинга аккумулятора (binary)"\n' >> $$FORMULA_TMP; \
	printf '  homepage "https://github.com/$(REPO)"\n' >> $$FORMULA_TMP; \
	printf '  version "%s"\n\n' "$(VERSION)" >> $$FORMULA_TMP; \
	printf '  on_macos do\n' >> $$FORMULA_TMP; \
	printf '    if Hardware::CPU.arm?\n' >> $$FORMULA_TMP; \
	printf '      url "https://github.com/$(REPO)/releases/download/%s/macbat-darwin-arm64.tar.gz"\n' "$(VERSION)" >> $$FORMULA_TMP; \
	printf '      sha256 "%s"\n' "$$SHA_ARM64" >> $$FORMULA_TMP; \
	printf '    else\n' >> $$FORMULA_TMP; \
	printf '      url "https://github.com/$(REPO)/releases/download/%s/macbat-darwin-amd64.tar.gz"\n' "$(VERSION)" >> $$FORMULA_TMP; \
	printf '      sha256 "%s"\n' "$$SHA_AMD64" >> $$FORMULA_TMP; \
	printf '    end\n  end\n\n' >> $$FORMULA_TMP; \
	printf '  def install\n    bin.install "macbat"\n  end\n\n' >> $$FORMULA_TMP; \
	printf '  test do\n    system "#{bin}/macbat", "--version"\n  end\nend\n' >> $$FORMULA_TMP; \
	mv $$FORMULA_TMP $(BINARY_NAME).rb; \

	echo "$(GREEN)Formula с бинарниками создана$(NC)";
	echo "$(YELLOW)Обновление tap-репозитория...$(NC)"; \
	TMP=$$(mktemp -d); \
	git clone --depth=1 "https://github.com/$(TAP_REPO).git" $$TMP; \
	mkdir -p $$TMP/Formula; cp $(BINARY_NAME).rb $$TMP/Formula/; \
	(cd $$TMP && git add Formula/$(BINARY_NAME).rb && git commit -m '$(BINARY_NAME) $(VERSION) (bin)' && git push origin HEAD); \
	rm -rf $$TMP; \
	echo "$(GREEN)✅ Релиз $(VERSION) с бинарниками опубликован$(NC)"

build: ## Собрать бинарный файл с информацией о версии
	@echo "$(GREEN)Сборка $(BINARY_NAME)...$(NC)"
	@echo "  Версия: $(VERSION)"
	@echo "  Коммит: $(COMMIT_HASH)"
	@echo "  Дата: $(BUILD_DATE)"
	@echo "  Номер сборки: $(BUILD_NUMBER)"
	CGO_ENABLED=1 go build -ldflags=$(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
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
	CGO_ENABLED=1 go build -ldflags=$(LDFLAGS) -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)
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