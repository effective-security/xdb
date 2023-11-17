// Package schema provides CLI commands
package schema

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/effective-security/porto/x/slices"
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
	Out          string   `help:"schema folder name to store files"`
	Package      string   `help:"package name to override from --out path"`
	StructSuffix string   `help:"optional, suffix for struct names"`
	Imports      []string `help:"optional go imports"`
	UseSchema    bool     `help:"optional, use schema name in table name"`
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

	return a.generate(ctx, a.DB, res)
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
	"join":  strings.Join,
	"lower": strings.ToLower,
}

func (a *GenerateCmd) generate(ctx *cli.Cli, dbName string, res schema.Tables) error {

	templateFuncMap["sqlToGoType"] = sqlToGoType(ctx.Provider)

	var headerTemplate = template.Must(template.New("rowCode").Funcs(templateFuncMap).Parse(codeHeaderTemplateText))
	var rowCodeTemplate = template.Must(template.New("rowCode").Funcs(templateFuncMap).Parse(codeModelTemplateText))

	packageName := slices.StringsCoalesce(a.Package, packageName(a.Out))

	imports := a.Imports
	if ctx.Provider == "postgres" {
		imports = append(imports, "github.com/lib/pq")
	}

	var err error
	schemas := map[string]schema.Tables{}
	for _, t := range res {
		schemas[t.Schema] = append(schemas[t.Schema], t)
	}

	var tableInfos []schema.TableInfo
	var tableDefs []tableDefinition

	w := ctx.Writer()
	if a.Out != "" {
		fn := filepath.Join(a.Out, "model.gen.go")
		f, err := os.OpenFile(fn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
		w = f
	}
	err = headerTemplate.Execute(w, &tableDefinition{
		DB:      dbName,
		Package: packageName,
		Imports: imports,
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
				Package:    packageName,
				Imports:    imports,
				Name:       prefix + tName,
				StructName: prefix + tName,
				SchemaName: t.Schema,
				TableName:  t.Name,
				Columns:    t.Columns,
				Indexes:    t.Indexes,
				PrimaryKey: t.PrimaryKey,
			}
			err = rowCodeTemplate.Execute(w, td)
			if err != nil {
				return errors.WithMessagef(err, "failed to generate model for %s.%s", t.Schema, t.Name)
			}
			tableDefs = append(tableDefs, td)
		}
	}

	var schemaCodeTemplate = template.Must(template.New("schemaCode").Funcs(templateFuncMap).Parse(codeSchemaTemplateText))
	var collsCodeTemplate = template.Must(template.New("collsCode").Funcs(templateFuncMap).Parse(codeTableColTemplateText))
	w = ctx.Writer()
	if a.Out != "" {
		fn := filepath.Join(a.Out, "tables.gen.go")
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
		Package: packageName,
		Imports: a.Imports,
		Tables:  tableInfos,
		Defs:    tableDefs,
	}
	err = schemaCodeTemplate.Execute(w, td)
	if err != nil {
		return errors.WithMessagef(err, "failed to generate schema")
	}

	for _, ctd := range tableDefs {
		err = collsCodeTemplate.Execute(w, ctd)
		if err != nil {
			return errors.WithMessagef(err, "failed to generate schema")
		}
	}
	return nil
}
