package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

// setupRussianTemplates устанавливает русские шаблоны для CLI
func setupRussianTemplates() {
	cli.RootCommandHelpTemplate = RussianAppHelpTemplate
	cli.CommandHelpTemplate = RussianCommandHelpTemplate
	cli.SubcommandHelpTemplate = RussianSubcommandHelpTemplate
}

// RussianAppHelpTemplate - русский шаблон справки для приложения
const RussianAppHelpTemplate = `НАЗВАНИЕ:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

ИСПОЛЬЗОВАНИЕ:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[глобальные опции]{{end}}{{if .Commands}} команда [опции команды]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[аргументы...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

ВЕРСИЯ:
   {{.Version}}{{end}}{{end}}{{if .Description}}

ОПИСАНИЕ:
   {{.Description}}{{end}}{{if len .Authors}}

АВТОР{{with $length := len .Authors}}{{if ne 1 $length}}Ы{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

КОМАНДЫ:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

ГЛОБАЛЬНЫЕ ОПЦИИ:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}

АВТОРСКИЕ ПРАВА:
   {{.Copyright}}{{end}}
`

// RussianCommandHelpTemplate - русский шаблон справки для команды
const RussianCommandHelpTemplate = `НАЗВАНИЕ:
   {{.HelpName}} - {{.Usage}}

ИСПОЛЬЗОВАНИЕ:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [опции команды]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[аргументы...]{{end}}{{end}}{{if .Category}}

КАТЕГОРИЯ:
   {{.Category}}{{end}}{{if .Description}}

ОПИСАНИЕ:
   {{.Description}}{{end}}{{if .VisibleFlags}}

ОПЦИИ:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

// RussianSubcommandHelpTemplate - русский шаблон справки для подкоманды
const RussianSubcommandHelpTemplate = `НАЗВАНИЕ:
   {{.HelpName}} - {{.Usage}}

ИСПОЛЬЗОВАНИЕ:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} команда{{if .VisibleFlags}} [опции команды]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[аргументы...]{{end}}{{end}}{{if .Description}}

ОПИСАНИЕ:
   {{.Description}}{{end}}

КОМАНДЫ:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

ОПЦИИ:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

// CompactRussianHelpTemplate - компактный шаблон для embedded систем
const CompactRussianHelpTemplate = `{{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

ИСПОЛЬЗОВАНИЕ:
   {{.HelpName}} {{if .VisibleFlags}}[опции]{{end}}{{if .Commands}} команда{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{end}}
{{if .Commands}}
КОМАНДЫ:{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{if .VisibleFlags}}

ОПЦИИ:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`

// ErrorTemplate - шаблон для вывода ошибок
const ErrorTemplate = `ОШИБКА:
   {{.}}

Попробуйте '{{.App.Name}} --help' для получения дополнительной информации.
`

// UsageErrorTemplate - шаблон для ошибок использования
const UsageErrorTemplate = `Неправильное использование: {{.Message}}

ИСПОЛЬЗОВАНИЕ:
   {{.App.HelpName}} {{if .Command}}{{.Command.HelpName}}{{end}} [опции] [аргументы...]

Попробуйте '{{.App.Name}} {{if .Command}}{{.Command.HelpName}} {{end}}--help' для получения дополнительной информации.
`

// VersionTemplate - шаблон для вывода версии
const VersionTemplate = `{{.Name}} версия {{.Version}}
`

// AuthorTemplate - шаблон для информации об авторе
const AuthorTemplate = `{{with $length := len .Authors}}{{if ne 1 $length}}АВТОРЫ{{else}}АВТОР{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}
`

// FlagTemplates содержит шаблоны для различных типов флагов
var FlagTemplates = struct {
	Bool   string
	String string
	Int    string
}{
	Bool:   "--{{.Name}}{{if .Aliases}}, {{range .Aliases}}-{{.}}{{end}}{{end}}\t{{.Usage}}{{if .Value}} (по умолчанию: {{.Value}}){{end}}",
	String: "--{{.Name}}{{if .Aliases}}, {{range .Aliases}}-{{.}}{{end}}{{end}} value\t{{.Usage}}{{if .Value}} (по умолчанию: \"{{.Value}}\"){{end}}",
	Int:    "--{{.Name}}{{if .Aliases}}, {{range .Aliases}}-{{.}}{{end}}{{end}} value\t{{.Usage}}{{if .Value}} (по умолчанию: {{.Value}}){{end}}",
}

// setupCompactTemplates устанавливает компактные шаблоны для embedded систем
func setupCompactTemplates() {
	cli.RootCommandHelpTemplate = CompactRussianHelpTemplate
	cli.CommandHelpTemplate = CompactCommandTemplate
}

// CompactCommandTemplate - компактный шаблон команды
const CompactCommandTemplate = `{{.HelpName}}{{if .Usage}} - {{.Usage}}{{end}}
{{if .UsageText}}{{.UsageText}}{{else}}Использование: {{.HelpName}}{{if .VisibleFlags}} [опции]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{end}}{{end}}{{if .VisibleFlags}}

Опции:{{range .VisibleFlags}}
  {{.}}{{end}}{{end}}
`

// DebugTemplate - шаблон для отладочной информации
const DebugTemplate = `=== ОТЛАДОЧНАЯ ИНФОРМАЦИЯ ===
Приложение: {{.Name}} v{{.Version}}
Команда: {{if .Command}}{{.Command.Name}}{{else}}нет{{end}}
Аргументы: {{.Args}}
Флаги:{{range .VisibleFlags}}
  {{.Names}}: {{.Value}}{{end}}
===========================
`

// CustomizeTemplates позволяет настроить шаблоны под конкретные нужды
func CustomizeTemplates(opts TemplateOptions) {
	if opts.Compact {
		setupCompactTemplates()
	} else {
		setupRussianTemplates()
	}

	if opts.ShowDebug {
		// Добавляем отладочную информацию к шаблонам
		cli.RootCommandHelpTemplate = DebugTemplate + "\n" + cli.RootCommandHelpTemplate
	}

	if opts.CustomHeader != "" {
		// Добавляем кастомный заголовок
		cli.RootCommandHelpTemplate = opts.CustomHeader + "\n\n" + cli.RootCommandHelpTemplate
	}
}

// TemplateOptions опции для настройки шаблонов
type TemplateOptions struct {
	// Compact использовать компактные шаблоны
	Compact bool

	// ShowDebug показывать отладочную информацию
	ShowDebug bool

	// CustomHeader кастомный заголовок
	CustomHeader string

	// NoColors отключить цвета (для embedded)
	NoColors bool
}

// LocalizeFlag локализует описание флага
func LocalizeFlag(flag cli.Flag) string {
	switch f := flag.(type) {
	case *cli.BoolFlag:
		return localizeBoolFlag(f)
	case *cli.StringFlag:
		return localizeStringFlag(f)
	case *cli.IntFlag:
		return localizeIntFlag(f)
	default:
		return flag.String()
	}
}

// localizeBoolFlag локализует булевый флаг
func localizeBoolFlag(flag *cli.BoolFlag) string {
	aliases := ""
	if len(flag.Aliases) > 0 {
		aliases = ", -" + strings.Join(flag.Aliases, ", -")
	}

	defaultVal := ""
	if flag.Value {
		defaultVal = " (по умолчанию: да)"
	}

	return fmt.Sprintf("--%-20s %s%s", flag.Name+aliases, flag.Usage, defaultVal)
}

// localizeStringFlag локализует строковый флаг
func localizeStringFlag(flag *cli.StringFlag) string {
	aliases := ""
	if len(flag.Aliases) > 0 {
		aliases = ", -" + strings.Join(flag.Aliases, ", -")
	}

	defaultVal := ""
	if flag.Value != "" {
		defaultVal = fmt.Sprintf(" (по умолчанию: \"%s\")", flag.Value)
	}

	return fmt.Sprintf("--%-20s %s%s", flag.Name+aliases+" ЗНАЧЕНИЕ", flag.Usage, defaultVal)
}

// localizeIntFlag локализует числовой флаг
func localizeIntFlag(flag *cli.IntFlag) string {
	aliases := ""
	if len(flag.Aliases) > 0 {
		aliases = ", -" + strings.Join(flag.Aliases, ", -")
	}

	defaultVal := ""
	if flag.Value != 0 {
		defaultVal = fmt.Sprintf(" (по умолчанию: %d)", flag.Value)
	}

	return fmt.Sprintf("--%-20s %s%s", flag.Name+aliases+" ЧИСЛО", flag.Usage, defaultVal)
}
