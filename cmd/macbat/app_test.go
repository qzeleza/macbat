package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestNewApp проверяет создание приложения
func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}

	if app == nil {
		t.Fatal("Приложение не создано")
	}

	if app.Logger() == nil {
		t.Fatal("Логгер не инициализирован")
	}
}

// TestCommands проверяет наличие всех команд
func TestCommands(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}

	expectedCommands := []string{
		"install",
		"uninstall",
		"log",
		"config",
	}

	commands := app.cli.Commands
	if len(commands) != len(expectedCommands) {
		t.Errorf("Ожидалось %d команд, получено %d", len(expectedCommands), len(commands))
	}

	// Проверяем наличие каждой команды
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Команда %s не найдена", expected)
		}
	}
}

// TestCommandAliases проверяет алиасы команд
func TestCommandAliases(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}

	tests := []struct {
		command string
		aliases []string
	}{
		{"install", []string{"i"}},
		{"uninstall", []string{"u", "remove"}},
		{"log", []string{"l", "logs"}},
		{"config", []string{"c", "cfg"}},
	}

	for _, tt := range tests {
		for _, cmd := range app.cli.Commands {
			if cmd.Name == tt.command {
				for _, expectedAlias := range tt.aliases {
					found := false
					for _, alias := range cmd.Aliases {
						if alias == expectedAlias {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Алиас %s не найден для команды %s", expectedAlias, tt.command)
					}
				}
			}
		}
	}
}

// TestHiddenFlags проверяет скрытые флаги
func TestHiddenFlags(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}

	hiddenFlags := []string{"background", "gui-agent"}

	for _, flagName := range hiddenFlags {
		found := false
		for _, flag := range app.cli.Flags {
			if boolFlag, ok := flag.(*cli.BoolFlag); ok {
				if boolFlag.Name == flagName {
					found = true
					if !boolFlag.Hidden {
						t.Errorf("Флаг %s должен быть скрытым", flagName)
					}
				}
			}
		}
		if !found {
			t.Errorf("Скрытый флаг %s не найден", flagName)
		}
	}
}

// TestRussianTemplates проверяет русские шаблоны
func TestRussianTemplates(t *testing.T) {
	templates := []struct {
		name     string
		template string
		keywords []string
	}{
		{
			name:     "AppHelp",
			template: RussianAppHelpTemplate,
			keywords: []string{"НАЗВАНИЕ", "ИСПОЛЬЗОВАНИЕ", "КОМАНДЫ", "ОПЦИИ"},
		},
		{
			name:     "CommandHelp",
			template: RussianCommandHelpTemplate,
			keywords: []string{"НАЗВАНИЕ", "ИСПОЛЬЗОВАНИЕ", "ОПИСАНИЕ"},
		},
		{
			name:     "SubcommandHelp",
			template: RussianSubcommandHelpTemplate,
			keywords: []string{"НАЗВАНИЕ", "ИСПОЛЬЗОВАНИЕ", "КОМАНДЫ"},
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			for _, keyword := range tt.keywords {
				if !strings.Contains(tt.template, keyword) {
					t.Errorf("Шаблон %s не содержит ключевое слово: %s", tt.name, keyword)
				}
			}
		})
	}
}

// TestBackgroundModes проверяет константы фоновых режимов
func TestBackgroundModes(t *testing.T) {
	if BackgroundModeMonitor != "--background" {
		t.Errorf("BackgroundModeMonitor = %s, ожидалось --background", BackgroundModeMonitor)
	}

	if BackgroundModeGUI != "--gui-agent" {
		t.Errorf("BackgroundModeGUI = %s, ожидалось --gui-agent", BackgroundModeGUI)
	}
}

// TestCommandFlags проверяет флаги команд
func TestCommandFlags(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}

	tests := []struct {
		command       string
		expectedFlags []string
	}{
		{"install", []string{"force"}},
		{"uninstall", []string{"keep-config", "keep-logs"}},
		{"log", []string{"lines", "follow", "level"}},
		{"config", []string{"editor", "show"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			var cmd *cli.Command
			for _, c := range app.cli.Commands {
				if c.Name == tt.command {
					cmd = c
					break
				}
			}

			if cmd == nil {
				t.Fatalf("Команда %s не найдена", tt.command)
			}

			for _, flagName := range tt.expectedFlags {
				found := false
				for _, flag := range cmd.Flags {
					switch f := flag.(type) {
					case *cli.BoolFlag:
						if f.Name == flagName {
							found = true
						}
					case *cli.StringFlag:
						if f.Name == flagName {
							found = true
						}
					case *cli.IntFlag:
						if f.Name == flagName {
							found = true
						}
					}
				}

				if !found {
					t.Errorf("Флаг %s не найден в команде %s", flagName, tt.command)
				}
			}
		})
	}
}

// TestVersionOutput проверяет вывод версии
func TestVersionOutput(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Запускаем с флагом версии
	app.Run([]string{"macbat", "--version"})

	// Восстанавливаем stdout
	w.Close()
	os.Stdout = oldStdout

	// Читаем вывод
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "macbat") {
		t.Error("Вывод версии не содержит имя приложения")
	}
}

// BenchmarkAppCreation измеряет производительность создания приложения
func BenchmarkAppCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app, err := NewApp()
		if err != nil {
			b.Fatal(err)
		}
		_ = app
	}
}

// BenchmarkCommandExecution измеряет производительность выполнения команды
func BenchmarkCommandExecution(b *testing.B) {
	app, err := NewApp()
	if err != nil {
		b.Fatal(err)
	}

	args := []string{"macbat", "--version"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Run(args)
	}
}
