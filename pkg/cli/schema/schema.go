// Package schema provides CLI commands
package schema

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/effective-security/x/configloader"
	"github.com/effective-security/x/slices"
	"github.com/effective-security/xdb/pkg/cli"
	"github.com/effective-security/xdb/schema"
	"github.com/ettle/strcase"
	"github.com/gertd/go-pluralize"
	"github.com/pkg/errors"
)

var pluralizeClient = pluralize.NewClient()

// Cmd base command for schema
type Cmd struct {
	Generate    GenerateCmd     `cmd:"" help:"generate Go model for database schema"`
	Columns     PrintColumnsCmd `cmd:"" help:"prints database schema"`
	Tables      PrintTablesCmd  `cmd:"" help:"prints database tables and dependencies"`
	Views       PrintViewsCmd   `cmd:"" help:"prints database views and dependencies"`
	ForeignKeys PrintFKCmd      `cmd:"" help:"prints Foreign Keys"`
}

// PrintColumnsCmd prints database schema
type PrintColumnsCmd struct {
	DB           string   `help:"database name: DataHub|DataHub.BrokerData|DataHub.HubspotData" required:""`
	Schema       string   `help:"optional schema name to filter"`
	Table        []string `help:"optional, list of tables, default: all tables"`
	Dependencies bool     `help:"optional, to discover all dependencies"`
}

// Run the command
func (a *PrintColumnsCmd) Run(ctx *cli.Cli) error {
	r, err := ctx.SchemaProvider(a.DB)
	if err != nil {
		return err
	}
	res, err := r.ListTables(ctx.Context(), a.Schema, a.Table, a.Dependencies)
	if err != nil {
		return err
	}

	return ctx.Print(res)
}

// PrintTablesCmd prints database tables with dependencies
type PrintTablesCmd struct {
	DB     string   `help:"database name" required:""`
	Schema string   `help:"optional schema name to filter"`
	Table  []string `help:"optional, list of tables, default: all tables"`
}

// Run the command
func (a *PrintTablesCmd) Run(ctx *cli.Cli) error {
	r, err := ctx.SchemaProvider(a.DB)
	if err != nil {
		return err
	}
	res, err := r.ListTables(ctx.Context(), a.Schema, a.Table, true)
	if err != nil {
		return err
	}
	w := ctx.Writer()

	if ctx.O == "json" || ctx.O == "yaml" {
		return ctx.Print(res)
	}
	for _, t := range res {
		fmt.Fprintf(w, "%s.%s\n", t.Schema, t.Name)
	}

	return nil
}

// PrintViewsCmd prints database tables with dependencies
type PrintViewsCmd struct {
	DB     string   `help:"database name" required:""`
	Schema string   `help:"optional schema name to filter"`
	View   []string `help:"optional, list of views, default: all views"`
}

// Run the command
func (a *PrintViewsCmd) Run(ctx *cli.Cli) error {
	r, err := ctx.SchemaProvider(a.DB)
	if err != nil {
		return err
	}
	res, err := r.ListViews(ctx.Context(), a.Schema, a.View)
	if err != nil {
		return err
	}
	w := ctx.Writer()

	if ctx.O == "json" || ctx.O == "yaml" {
		return ctx.Print(res)
	}
	for _, t := range res {
		fmt.Fprintf(w, "%s.%s\n", t.Schema, t.Name)
	}

	return nil
}

// PrintFKCmd prints database FK
type PrintFKCmd struct {
	DB     string   `help:"database name" required:""`
	Schema string   `help:"optional schema name to filter"`
	Table  []string `help:"optional, list of tables, default: all tables"`
}

// Run the command
func (a *PrintFKCmd) Run(ctx *cli.Cli) error {
	r, err := ctx.SchemaProvider(a.DB)
	if err != nil {
		return err
	}
	res, err := r.ListForeignKeys(ctx.Context(), a.Schema, a.Table)
	if err != nil {
		return err
	}
	return ctx.Print(res)
}

// GenerateCmd generates database schema
type GenerateCmd struct {
	DB           string   `help:"database name" required:""`
	Schema       string   `help:"optional schema name to filter"`
	Table        []string `help:"optional, list of tables, default: all tables"`
	View         []string `help:"optional, list of views"`
	Dependencies bool     `help:"optional, to discover all dependencies"`
	OutModel     string   `help:"folder name to store model files"`
	OutSchema    string   `help:"folder name to store schema files"`
	PkgModel     string   `help:"package name to override from --out-model path"`
	PkgSchema    string   `help:"package name to override from --out-schema path"`
	StructSuffix string   `help:"optional, suffix for struct names"`
	Imports      []string `help:"optional go imports"`
	UseSchema    bool     `help:"optional, use schema name in table name"`
	TypesDef     string   `help:"optional, path to types definition file"`
}

// Run the command
func (a *GenerateCmd) Run(ctx *cli.Cli) error {
	r, err := ctx.SchemaProvider(a.DB)
	if err != nil {
		return err
	}

	res, err := r.ListTables(ctx.Context(), a.Schema, a.Table, a.Dependencies)
	if err != nil {
		return err
	}

	if len(a.View) > 0 {
		res2, err := r.ListViews(ctx.Context(), a.Schema, a.View)
		if err != nil {
			return err
		}
		res = append(res, res2...)
	}

	return a.generate(ctx, r.Name(), a.DB, res)
}

func packageName(folder string) string {
	f := path.Base(folder)
	if f == "" || f == "." || f == "/" {
		return "model"
	}
	return f
}

func goName(s string) string {
	if s[0] == '_' {
		a := []rune(s)
		a[0] = 'X'
		s = string(a)
	}
	return strcase.ToGoPascal(s)
}

func tableStructName(s string) string {
	return goName(pluralizeClient.Singular(s)) + "Table"
}

var templateFuncMap = template.FuncMap{
	"goName":          goName,
	"tableStructName": tableStructName,
	"concat": func(args ...string) string {
		return strings.Join(args, "")
	},
	"join":        strings.Join,
	"lower":       strings.ToLower,
	"sqlToGoType": toGoType,
}

type override struct {
	Tables map[string]string `json:"tables" yaml:"tables"`
	Fields map[string]string `json:"fields" yaml:"fields"`
	Types  map[string]string `json:"types" yaml:"types"`
}

func (a *GenerateCmd) generate(ctx *cli.Cli, provider, dbName string, res schema.Tables) error {
	var headerTemplate = template.Must(template.New("rowCode").Funcs(templateFuncMap).Parse(codeHeaderTemplateText))
	var rowCodeTemplate = template.Must(template.New("rowCode").Funcs(templateFuncMap).Parse(codeModelTemplateText))

	modelPkg := slices.StringsCoalesce(a.PkgModel, packageName(a.OutModel))
	schemaPkg := slices.StringsCoalesce(a.PkgSchema, packageName(a.OutSchema))

	var dialect string
	imports := a.Imports
	if provider == "postgres" {
		imports = append(imports, "github.com/lib/pq")
		dialect = "xsql.Postgres"
	} else if provider == "sqlserver" {
		dialect = "xsql.SQLServer"
	} else {
		dialect = "xsql.NoDialect"
	}

	if a.TypesDef != "" {
		var defs override
		err := configloader.Unmarshal(a.TypesDef, &defs)
		if err != nil {
			return errors.WithMessagef(err, "failed to load types definition")
		}
		for k, v := range defs.Types {
			typesMap[k] = v
		}
		for k, v := range defs.Fields {
			fieldNamesMap[k] = v
		}
		for k, v := range defs.Tables {
			tableNamesMap[k] = v
		}
	}

	schemas := map[string]schema.Tables{}
	for _, t := range res {
		if name, ok := tableNamesMap[t.SchemaName]; ok {
			t.Name = name
		}
		schemas[t.Schema] = append(schemas[t.Schema], t)
	}

	var err error
	var tableInfos []schema.TableInfo
	var tableDefs []tableDefinition

	w := ctx.Writer()
	buf := &bytes.Buffer{}

	if a.OutModel != "" {
		_ = os.MkdirAll(a.OutModel, 0777)
		fn := filepath.Join(a.OutModel, "model.gen.go")
		f, err := os.OpenFile(fn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
		w = f
	}
	err = headerTemplate.Execute(buf, &tableDefinition{
		DB:      dbName,
		Package: modelPkg,
		Imports: imports,
		Dialect: dialect,
	})
	if err != nil {
		return errors.WithMessagef(err, "failed to generate header")
	}

	for schemaName, tables := range schemas {
		sName := strcase.ToGoPascal(schemaName)
		for _, t := range tables {
			tName := strcase.ToGoPascal(pluralizeClient.Singular(t.Name))
			if a.StructSuffix != "" {
				tName += t.Name + strcase.ToGoPascal(a.StructSuffix)
			}

			tableInfos = append(tableInfos, schema.TableInfo{
				Schema:     t.Schema,
				Name:       t.Name,
				SchemaName: t.SchemaName,
				Columns:    t.Columns.Names(),
				Indexes:    t.Indexes.Names(),
				PrimaryKey: t.PrimaryKeyName(),
			})
			prefix := ""
			if a.UseSchema && !slices.ContainsStringEqualFold([]string{"dbo", "public"}, schemaName) {
				prefix = sName
			}

			td := tableDefinition{
				DB:         dbName,
				Package:    modelPkg,
				Imports:    imports,
				Dialect:    dialect,
				Name:       prefix + tName,
				StructName: prefix + tName,
				SchemaName: t.Schema,
				TableName:  t.Name,
				Columns:    t.Columns,
				Indexes:    t.Indexes,
				PrimaryKey: t.PrimaryKey,
			}

			for _, c := range td.Columns {
				if res, ok := fieldNamesMap[c.SchemaName]; ok {
					c.Name = res
				}
			}

			err = rowCodeTemplate.Execute(buf, td)
			if err != nil {
				return errors.WithMessagef(err, "failed to generate model for %s.%s", t.Schema, t.Name)
			}
			tableDefs = append(tableDefs, td)
		}
	}

	code, err := format.Source(buf.Bytes())
	if err != nil {
		return errors.WithMessagef(err, "failed to format")
	}
	w.Write(code)

	var schemaCodeTemplate = template.Must(template.New("schemaCode").Funcs(templateFuncMap).Parse(codeSchemaTemplateText))
	var collsCodeTemplate = template.Must(template.New("collsCode").Funcs(templateFuncMap).Parse(codeTableColTemplateText))

	buf.Reset()
	w = ctx.Writer()
	if a.OutSchema != "" {
		_ = os.MkdirAll(a.OutSchema, 0777)
		fn := filepath.Join(a.OutSchema, "schema.gen.go")
		f, err := os.OpenFile(fn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
		w = f
	}
	td := schemaDefinition{
		DB:      dbName,
		Package: schemaPkg,
		Imports: a.Imports,
		Dialect: dialect,
		Tables:  tableInfos,
		Defs:    tableDefs,
	}
	err = schemaCodeTemplate.Execute(buf, td)
	if err != nil {
		return errors.WithMessagef(err, "failed to generate schema")
	}

	for _, ctd := range tableDefs {
		err = collsCodeTemplate.Execute(buf, ctd)
		if err != nil {
			return errors.WithMessagef(err, "failed to generate schema")
		}
	}
	code, err = format.Source(buf.Bytes())
	if err != nil {
		return errors.WithMessagef(err, "failed to format")
	}
	w.Write(code)

	return nil
}
