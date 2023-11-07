package schema

import (
	"github.com/effective-security/xdb"
)

type schemaDefinition struct {
	DB      string
	Package string
	Imports []string

	Tables []xdb.TableInfo
}

var codeSchemaTemplateText = `// DO NOT EDIT!
// This file is MACHINE GENERATED
// DB: {{ .DB }}

package {{ .Package }}

import (

	"github.com/effective-security/xdb"
	{{range .Imports}}{{/*
		*/}}"{{ . }}"
	{{ end }}
)

{{ range .Tables }}
{{- $tableName := tableStructName .Name }}
// {{ $tableName }} provides table info for '{{ .Name }}'
var {{ $tableName }} = xdb.TableInfo{
	SchemaName : "{{ .SchemaName }}",
	Schema     : "{{ .Schema }}",
	Name       : "{{ .Name }}",
	PrimaryKey : "{{ .PrimaryKey }}", 
	Columns    : []string{ {{- range .Columns }}"{{ . }}", {{ end -}} },
	Indexes    : []string{ {{- range .Indexes }}"{{ . }}", {{ end -}} },
}
{{ end }}

// {{ goName .DB }}Tables provides tables map for {{ .DB }}
var {{ goName .DB }}Tables = map[string]*xdb.TableInfo{
{{ range .Tables }}
 	"{{ .Name }}": &{{ tableStructName .Name }},
{{- end }}
}
`
