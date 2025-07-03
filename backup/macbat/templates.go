// Пакет main содержит шаблоны для вывода справки и помощи в командной строке.
//
// @package main
package main

// RussianHelpTemplate - шаблон справки на русском языке для основного приложения.
//
// Содержит форматированный вывод:
// - Название и описание приложения
// - Версия
// - Информация об авторе
// - Список доступных команд
// - Глобальные флаги
//
// Используется для форматирования вывода команды help.
//
// @const string RussianHelpTemplate

const RussianHelpTemplate = `
{{.Name}}: {{.Description}}
Версия: {{.Version}}
{{if .Authors}}Автор: {{range $i, $author := .Authors}}{{if $i}}, {{end}}{{$author.Name}}{{if $author.Email}} <{{$author.Email}}>{{end}}{{end}}
{{end}}
{{if .Usage}}Начало работы: {{.Usage}}{{end}}
Использование: {{.Name}} [команда [команды]] [аргументы...]

{{if .Commands}}Команды:{{range .VisibleCategories}}{{if .Name}}
{{.Name}}:{{range .VisibleCommands}}
  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}
{{end}}
`

// RussianCommandHelpTemplate - шаблон справки для команд на русском языке.
//
// Содержит форматированный вывод:
// - Название и описание команды
// - Синтаксис использования
// - Доступные флаги команды
//
// Используется для форматирования вывода справки по конкретной команде.
//
// @const string RussianCommandHelpTemplate
const RussianCommandHelpTemplate = `{{.HelpName}} - {{.Usage}}
{{if .Description}}
Описание:
  {{.Description}}
{{end}}
Использование:
  {{.HelpName}}{{if .VisibleFlags}} [команды]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{end}}

Флаги:{{range .VisibleFlags}}
  {{.}}{{end}}
`

// RussianSubcommandHelpTemplate - шаблон справки для подкоманд на русском языке.
//
// Содержит форматированный вывод:
// - Название и описание группы команд
// - Список доступных подкоманд
// - Доступные флаги для группы команд
//
// Используется для форматирования вывода справки по группам команд.
//
// @const string RussianSubcommandHelpTemplate
const RussianSubcommandHelpTemplate = `
{{.HelpName}} - {{.Usage}}
{{if .Description}}
Описание:
  {{.Description}}
{{end}}
Использование:
  {{.HelpName}} [команда]{{if .VisibleFlags}} [команды]{{end}}
  {{if .ArgsUsage}}{{.ArgsUsage}}{{end}}

Команды:{{range .VisibleCategories}}{{if .Name}}{{.Name}}:{{range .VisibleCommands}}
  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}
{{if .VisibleFlags}}
Флаги:
{{range .VisibleFlags}}  {{.}}
{{end}}{{end}}
`
