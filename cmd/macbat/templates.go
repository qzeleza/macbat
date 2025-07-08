// cmd/macbat/cli/templates.go
package main

// RussianHelpTemplate содержит русские шаблоны для CLI справки
const RussianHelpTemplate = `НАЗВАНИЕ:
   {{template "helpNameTemplate" .}}

ИСПОЛЬЗОВАНИЕ:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[ПАРАМЕТРЫ]{{end}}{{if .Commands}} КОМАНДА [АРГУМЕНТЫ_КОМАНДЫ...]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[аргументы...]{{end}}{{end}}

{{if .Version}}{{if not .HideVersion}}ВЕРСИЯ:
   {{.Version}}
{{end}}{{end}}{{if .Description}}ОПИСАНИЕ:
   {{template "descriptionTemplate" .}}
{{end}}{{if len .Authors}}АВТОР{{with $length := len .Authors}}{{if ne 1 $length}}Ы{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}
{{end}}{{if .VisibleCommands}}КОМАНДЫ:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlagCategories}}ПАРАМЕТРЫ:{{range .VisibleFlagCategories}}
   {{if .Name}}{{.Name}}
   {{end}}{{range .Flags}}{{.}}
   {{end}}{{end}}{{else}}{{if .VisibleFlags}}ПАРАМЕТРЫ:
{{range $index, $flag := .VisibleFlags}}   {{$flag}}
{{end}}{{end}}{{end}}{{if .Copyright}}COPYRIGHT:
   {{template "copyrightTemplate" .}}
{{end}}`

const CommandHelpTemplate = `НАЗВАНИЕ:
   {{template "commandNameTemplate" .}}

ИСПОЛЬЗОВАНИЕ:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [ПАРАМЕТРЫ_КОМАНДЫ]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[аргументы...]{{end}}{{end}}{{if .Category}}

КАТЕГОРИЯ:
   {{.Category}}{{end}}{{if .Description}}

ОПИСАНИЕ:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

ПАРАМЕТРЫ:{{range .VisibleFlagCategories}}
   {{if .Name}}{{.Name}}
   {{end}}{{range .Flags}}{{.}}
   {{end}}{{end}}{{else}}{{if .VisibleFlags}}

ПАРАМЕТРЫ:
{{range .VisibleFlags}}   {{.}}
{{end}}{{end}}{{end}}`
