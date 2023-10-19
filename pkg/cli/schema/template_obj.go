package schema

import (
	"fmt"

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

const yesVal = "YES"

func sqlToGoType(c *schema.Column) string {
	switch c.Type {

	case "int", "bigint":
		if c.Nullable == yesVal {
			return "*int"
		}
		return "int"

	case "decimal", "numeric":
		if c.Nullable == yesVal {
			return "*float64"
		}
		return "float64"

	case "bit", "boolean":
		if c.Nullable == yesVal {
			return "*bool"
		}
		return "bool"

	case "jsonb":
		return "xdb.NULLString"

	case "char", "nchar", "varchar", "nvarchar", "character varying":
		if c.Nullable == yesVal {
			return "xdb.NULLString"
		}
		return "string"

	case "uniqueidentifier":
		if c.Nullable == yesVal {
			return "xdb.NULLString"
		}
		return "string"

	case "time", "date", "datetime", "datetime2", "timestamp", "timestamp with time zone":
		return "xdb.Time"
	default:
		panic(fmt.Sprintf("don't know how to convert type: %s [%s]", c.Type, c.Name))
	}
}

var codeRowTemplateText = `// DO NOT EDIT!
// This file is MACHINE GENERATED
// Table: {{ .SchemaName }}.{{ .TableName }}

package {{ .Package }}

import (
	"github.com/effective-security/porto/x/xdb"
	"github.com/pkg/errors"
	{{range .Imports}}{{/*
		*/}}"{{ . }}"
	{{ end }}
)

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
	// {{$fieldName}} representation of DB field: '{{.Type}} {{.Name}}'
	// Indexed: {{ .IsIndex }}
	{{- if .Ref }}
	// FK: {{ .Ref.RefColumnSchemaName }}
	{{- end }}
	{{$fieldName}} {{ sqlToGoType . }} ` + "`" + `db:"{{.Name}},{{.Type}}"` + "`" + `
{{- end }}
}

// ScanRow scans one row for {{ .TableName }}.
func(r *{{ .StructName }}) ScanRow(rows xdb.Scanner) error {
	err := rows.Scan(
{{- range $i, $e := .Columns }}
		&r.{{ goName $e.Name }},
{{- end }}
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
`
