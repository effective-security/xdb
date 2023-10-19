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

	"github.com/FlowbankSA/gapsys/pkg/db"
	{{range .Imports}}{{/*
		*/}}"{{ . }}"
	{{ end }}
)

{{ range .Tables }}
// {{ .Name }}Table provides table info for '{{ .Name }}'
var {{ .Name }}Table = db.TableInfo{
	SchemaName : "{{ .SchemaName }}",
	Schema     : "{{ .Schema }}",
	Name       : "{{ .Name }}",
	PrimaryKey : "{{ .PrimaryKey }}", 
	Columns    : []string{ {{- range .Columns }}"{{ . }}", {{ end -}} },
	Indexes    : []string{ {{- range .Indexes }}"{{ . }}", {{ end -}} },
}
{{ end }}

// {{ .DB }}Tables provides tables map for {{ .DB }}
var {{ .DB }}Tables = map[string]*db.TableInfo{
{{ range .Tables }}
 	"{{ .Name }}": &{{ .Name }}Table,
{{- end }}
}
`
