package schema

import (
	"github.com/effective-security/xdb/schema"
)

type tableDefinition struct {
	DB              string
	Package         string
	Imports         []string
	Name            string
	Dialect         string
	StructName      string
	SchemaName      string
	TableName       string
	TableStructName string
	Columns         schema.Columns
	Indexes         schema.Indexes
	PrimaryKey      *schema.Column
}

type schemaDefinition struct {
	DB      string
	Package string
	Imports []string
	Dialect string
	Tables  []*schema.TableInfo
	Defs    []*tableDefinition
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

// Dialect provides Dialect for {{ .DB }}
var Dialect = {{ .Dialect }}
`

var codeTableColTemplateText = `

// {{ .StructName }} provides column definitions for table '{{ .SchemaName }}.{{ .TableName }}'.
{{- if .PrimaryKey }}
// Primary key: {{ .PrimaryKey.Name }}
{{- end}}
{{- if .Indexes }}
// Indexes:
{{- range .Indexes }}
//   {{ .Name }}:{{if .IsPrimary }} PRIMARY{{end}}{{if .IsUnique }} UNIQUE{{end}} [{{ join .ColumnNames "," }}]
{{- end }}
{{- end }}
var {{ .StructName }} = struct {
	Table *schema.TableInfo

{{- range .Columns }}
	{{columnStructName .}} schema.Column // {{.Name}} {{.Type}}
{{- end }}
}{
	Table: &{{.TableStructName}},

	{{- range .Columns }}
	{{ columnStructName .}}: schema.Column{{.StructString}},
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
{{- $fieldName := columnStructName . }}
	// {{$fieldName}} represents '{{.Name}}' column of '{{.Type}}'
	{{$fieldName}} {{ sqlToGoType . }} ` + "`" + `{{ .Tag }}` + "`" + `
{{- end }}
}

// ScanRow scans one row for {{ .TableName }}.
func(m *{{ .StructName }}) ScanRow(rows xdb.Row) error {
	err := rows.Scan(
{{- range $i, $e := .Columns }}
		&m.{{ columnStructName $e }},
{{- end }}
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

type {{ .StructName }}Slice []*{{ .StructName }}
type {{ .StructName }}Result = xdb.Result[{{ .StructName }}, *{{ .StructName }}]
`

var codeSchemaTemplateText = `// DO NOT EDIT!
// This file is MACHINE GENERATED
// DB: {{ .DB }}

package {{ .Package }}

import (
	"github.com/effective-security/xdb/schema"
	{{range .Imports}}{{/*
		*/}}"{{ . }}"
	{{ end }}
)

// Dialect provides Dialect for {{ .DB }}
var Dialect = {{ .Dialect }}
{{- $dialect := .Dialect }}

{{ range .Tables }}
{{- $tableName := tableInfoStructName . }}
// {{ $tableName }} provides table info for '{{ .Name }}'
var {{ $tableName }} = schema.TableInfo{
	SchemaName : "{{ .SchemaName }}",
	Schema     : "{{ .Schema }}",
	Name       : "{{ .Name }}",
	PrimaryKey : "{{ .PrimaryKey }}", 
	Columns    : []string{ {{- range .Columns }}"{{ . }}", {{ end -}} },
	Indexes    : []string{ {{- range .Indexes }}"{{ . }}", {{ end -}} },
	Dialect    : {{ $dialect }},
}
{{ end }}

// {{ goName .DB }}Tables provides tables map for {{ .DB }}
var {{ goName .DB }}Tables = map[string]*schema.TableInfo{
{{- range .Tables }}
 	"{{ .Name }}": &{{ tableInfoStructName . }},
{{- end }}
}
`
