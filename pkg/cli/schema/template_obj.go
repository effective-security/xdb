package schema

import (
	"github.com/effective-security/xdb/schema"
)

type tableDefinition struct {
	DB         string
	Package    string
	Imports    []string
	Name       string
	StructName string
	SchemaName string
	TableName  string
	Columns    schema.Columns
	Indexes    schema.Indexes
	PrimaryKey *schema.Column
}

var codeHeaderTemplateText = `// DO NOT EDIT!
// This file is MACHINE GENERATED
// DB: {{ .DB }}

package {{ .Package }}

import (
	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/schema"
	"github.com/pkg/errors"
	{{range .Imports}}{{/*
		*/}}"{{ . }}"
	{{ end }}
)

`

var codeTableColTemplateText = `

// {{ .StructName }}Col provides column definitions for table '{{ .SchemaName }}.{{ .TableName }}'.
{{- if .PrimaryKey }}
// Primary key: {{ .PrimaryKey.Name }}
{{- end}}
{{- if .Indexes }}
// Indexes:
{{- range .Indexes }}
//   {{ .Name }}:{{if .IsPrimary }} PRIMARY{{end}}{{if .IsUnique }} UNIQUE{{end}} [{{ join .ColumnNames "," }}]
{{- end }}
{{- end }}
var {{ .StructName }}Col = struct {
{{- range .Columns }}
{{- $fieldName := goName .Name }}
	{{$fieldName}} schema.Column // {{.Name}} {{.Type}}
{{- end }}
}{
	{{- range .Columns }}
	{{ goName .Name}}: schema.Column{{.StructString}},
	{{- end }}
}
`

var codeModelTemplateText = `

// {{ .StructName }} represents one row from table '{{ .SchemaName }}.{{ .TableName }}'.
{{- if .PrimaryKey }}
// Primary key: {{ .PrimaryKey.Name }}
{{- end}}
{{- if .Indexes }}
// Indexes:
{{- range .Indexes }}
//   {{ .Name }}:{{if .IsPrimary }} PRIMARY{{end}}{{if .IsUnique }} UNIQUE{{end}} [{{ join .ColumnNames "," }}]
{{- end }}
{{- end }}
type {{ .StructName }} struct {
{{- range .Columns }}
{{- $fieldName := goName .Name }}
	// {{$fieldName}} represents '{{.Name}}' column of '{{.Type}}'
	{{$fieldName}} {{ sqlToGoType . }} ` + "`" + `{{ .Tag }}` + "`" + `
{{- end }}
}

// ScanRow scans one row for {{ .TableName }}.
func(m *{{ .StructName }}) ScanRow(rows xdb.Row) error {
	err := rows.Scan(
{{- range $i, $e := .Columns }}
		&m.{{ goName $e.Name }},
{{- end }}
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

`
