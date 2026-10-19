// Package schema provides CLI commands
package schema

import (
	"bytes"
	"cmp"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"maps"
	"os"
	"path"
	"path/filepath"
	stdslices "slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/cockroachdb/errors"
	"github.com/effective-security/x/configloader"
	"github.com/effective-security/x/slices"
	"github.com/effective-security/xdb/internal/cli"
	"github.com/effective-security/xdb/schema"
	"github.com/ettle/strcase"
	"github.com/gertd/go-pluralize"
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
	DB           string   `help:"database name" required:""`
	Schema       []string `help:"optional schema name to filter"`
	Table        []string `help:"optional, list of tables, default: all tables"`
	Dependencies bool     `help:"optional, to discover all dependencies"`
	Views        bool     `help:"optional, to include views"`
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

	_ = ctx.Print(res)

	if a.Views {
		res, err = r.ListViews(ctx.Context(), a.Schema, a.Table)
		if err != nil {
			return err
		}
		if len(res) > 0 {
			_ = ctx.Print(res)
		}
	}
	return nil
}

// PrintTablesCmd prints database tables with dependencies
type PrintTablesCmd struct {
	DB     string   `help:"database name" required:""`
	Schema []string `help:"optional schema name to filter"`
	Table  []string `help:"optional, list of tables, default: all tables"`
	Views  bool     `help:"optional, to include views"`
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

	if a.Views {
		vres, err := r.ListViews(ctx.Context(), a.Schema, a.Table)
		if err != nil {
			return err
		}
		res = append(res, vres...)
	}

	w := ctx.Writer()

	if ctx.O == "json" || ctx.O == "yaml" {
		return ctx.Print(res)
	}
	for _, t := range res {
		_, _ = fmt.Fprintf(w, "%s.%s\n", t.Schema, t.Name)
	}

	return nil
}

// PrintViewsCmd prints database tables with dependencies
type PrintViewsCmd struct {
	DB     string   `help:"database name" required:""`
	Schema []string `help:"optional schema name to filter"`
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
		_, _ = fmt.Fprintf(w, "%s.%s\n", t.Schema, t.Name)
	}

	return nil
}

// PrintFKCmd prints database FK
type PrintFKCmd struct {
	DB     string   `help:"database name" required:""`
	Schema []string `help:"optional schema name to filter"`
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
	Schema       []string `help:"optional schema name to filter"`
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

func tableStructName(t *schema.Table) string {
	name := t.Name
	if res, ok := tableNamesMap[t.SchemaName]; ok {
		return res + "Table"
	}

	return goName(pluralizeClient.Singular(name)) + "Table"
}

func tableInfoStructName(t *schema.TableInfo) string {
	name := t.Name
	if res, ok := tableNamesMap[t.SchemaName]; ok {
		return res + "TableInfo"
	}

	return goName(pluralizeClient.Singular(name)) + "TableInfo"
}

func columnStructName(c *schema.Column) string {
	name := c.Name
	if res, ok := fieldNamesMap[c.SchemaName]; ok {
		return res
	}

	return goName(name)
}

var templateFuncMap = template.FuncMap{
	"goName":              goName,
	"tableStructName":     tableStructName,
	"tableInfoStructName": tableInfoStructName,
	"columnStructName":    columnStructName,
	"concat": func(args ...string) string {
		return strings.Join(args, "")
	},
	"join":        strings.Join,
	"lower":       strings.ToLower,
	"sqlToGoType": toGoType,
}

type override struct {
	Tables      map[string]string `json:"tables" yaml:"tables"`
	Fields      map[string]string `json:"fields" yaml:"fields"`
	Types       map[string]string `json:"types" yaml:"types"`
	WithCache   []string          `json:"with_cached_props" yaml:"with_cached_props"`
	DropColumns []string          `json:"drop_columns" yaml:"drop_columns"`
}

func filterColumnNames(columns []string, dropColumnsMap map[string]bool) []string {
	var result []string
	for _, c := range columns {
		if !dropColumnsMap[c] {
			result = append(result, c)
		}
	}
	return result
}

func filterColumns(columns schema.Columns, dropColumnsMap map[string]bool) schema.Columns {
	var result schema.Columns
	for _, c := range columns {
		if !dropColumnsMap[c.Name] {
			result = append(result, c)
		}
	}
	return result
}

// goImport is a package the generated code imports, by the name the code
// uses and its path
type goImport struct {
	name string
	path string
}

// imports of the generated code
var (
	importErrors    = goImport{name: "errors", path: "github.com/cockroachdb/errors"}
	importPq        = goImport{name: "pq", path: "github.com/lib/pq"}
	importValues    = goImport{name: "values", path: "github.com/effective-security/x/values"}
	importXdb       = goImport{name: "xdb", path: "github.com/effective-security/xdb"}
	importXdbSchema = goImport{name: "xdbschema", path: "github.com/effective-security/xdb/schema"}
	importXsql      = goImport{name: "xsql", path: "github.com/effective-security/xdb/xsql"}
)

// spec returns the import spec, with the name only when it differs from the
// default name of the path.
func (i goImport) spec() string {
	if i.name == defaultPackageName(i.path) {
		return strconv.Quote(i.path)
	}
	return i.name + " " + strconv.Quote(i.path)
}

// generate writes one model file and one schema file for all the tables,
// in the order of res, whatever schemas they belong to.
func (a *GenerateCmd) generate(ctx *cli.Cli, provider, dbName string, res schema.Tables) error {
	var codeHeaderTemplate = template.Must(template.New("header").Funcs(templateFuncMap).Parse(codeHeaderTemplateText))
	var codeModelTemplate = template.Must(template.New("codeModelTemplate").Funcs(templateFuncMap).Parse(codeModelTemplateText))
	var schemaHeaderCodeTemplate = template.Must(template.New("schemaHeaderCodeTemplate").Funcs(templateFuncMap).Parse(codeSchemaHeaderTemplateText))
	var schemaCodeTemplate = template.Must(template.New("schemaCodeTemplate").Funcs(templateFuncMap).Parse(codeTableSchemaTemplateText))

	modelPkg := cmp.Or(a.PkgModel, packageName(a.OutModel))
	schemaPkg := cmp.Or(a.PkgSchema, packageName(a.OutSchema))

	var dialect string
	switch provider {
	case "postgres":
		dialect = "xsql.Postgres"
	case "sqlserver":
		dialect = "xsql.SQLServer"
	default:
		dialect = "xsql.NoDialect"
	}

	dropColumnsMap := map[string]bool{}
	if a.TypesDef != "" {
		var defs override
		err := configloader.Unmarshal(a.TypesDef, &defs)
		if err != nil {
			return errors.WithMessagef(err, "failed to load types definition")
		}
		maps.Copy(typesMap, defs.Types)
		maps.Copy(fieldNamesMap, defs.Fields)
		maps.Copy(tableNamesMap, defs.Tables)
		for _, v := range defs.WithCache {
			modelWithCacheMap[v] = true
		}
		for _, v := range defs.DropColumns {
			dropColumnsMap[v] = true
		}
	}

	// nothing matched the filters: write no files, which would not compile
	if len(res) == 0 {
		return nil
	}

	var tableInfos []*schema.TableInfo
	var tableDefs []*tableDefinition
	// packages that the column types refer to, such as pb in []*pb.Role
	usedPackages := map[string]bool{}
	usesCache := false

	for _, t := range res {
		structName := strcase.ToGoPascal(pluralizeClient.Singular(t.Name))
		if a.StructSuffix != "" {
			structName += strcase.ToGoPascal(a.StructSuffix)
		}

		tableInfos = append(tableInfos, &schema.TableInfo{
			Schema:     t.Schema,
			Name:       t.Name,
			SchemaName: t.SchemaName,
			Columns:    filterColumnNames(t.Columns.Names(), dropColumnsMap),
			Indexes:    t.Indexes.Names(),
			PrimaryKey: t.PrimaryKeyName(),
		})
		prefix := ""
		if a.UseSchema && !slices.ContainsStringEqualFold([]string{"dbo", "public"}, t.Schema) {
			prefix = strcase.ToGoPascal(t.Schema)
		}

		td := &tableDefinition{
			DB:              dbName,
			Package:         modelPkg,
			Dialect:         dialect,
			Name:            prefix + structName,
			StructName:      prefix + structName,
			SchemaName:      t.Schema,
			TableName:       t.Name,
			TableStructName: tableStructName(t),
			Columns:         filterColumns(t.Columns, dropColumnsMap),
			Indexes:         t.Indexes,
			PrimaryKey:      t.PrimaryKey,
			WithCache:       modelWithCacheMap[t.SchemaName],
		}
		if modelPkg != schemaPkg {
			td.SchemaPackage = schemaPkg + "."
		}
		if name, ok := tableNamesMap[t.SchemaName]; ok {
			td.StructName = name
		}
		for _, c := range td.Columns {
			for _, pkg := range typePackages(toGoType(c)) {
				usedPackages[pkg] = true
			}
		}
		usesCache = usesCache || td.WithCache
		tableDefs = append(tableDefs, td)
	}

	// model file
	modelImports := []goImport{importErrors, importXdb, importXdbSchema}
	modelDef := &tableDefinition{
		DB:      dbName,
		Package: modelPkg,
	}
	if modelPkg != schemaPkg {
		modelDef.Dialect = dialect
		modelImports = append(modelImports, importXsql)
		if p := importPath(a.OutSchema); p != "" {
			modelImports = append(modelImports, goImport{name: schemaPkg, path: p})
		}
	}
	// an --imports package takes the place of a known one of the same name
	named := map[string]bool{}
	for _, imp := range a.Imports {
		ui := userImport(imp)
		if usedPackages[ui.name] {
			modelImports = append(modelImports, ui)
			named[ui.name] = true
		}
	}
	if usedPackages["pq"] && !named["pq"] {
		modelImports = append(modelImports, importPq)
	}
	if (usesCache || usedPackages["values"]) && !named["values"] {
		modelImports = append(modelImports, importValues)
	}
	modelDef.Imports = importSpecs(modelImports)

	buf := &bytes.Buffer{}
	err := codeHeaderTemplate.Execute(buf, modelDef)
	if err != nil {
		return errors.WithMessagef(err, "failed to generate header")
	}
	for _, td := range tableDefs {
		err = codeModelTemplate.Execute(buf, td)
		if err != nil {
			return errors.WithMessagef(err, "failed to generate model for %s.%s", td.SchemaName, td.TableName)
		}
	}
	err = a.writeCode(ctx, a.OutModel, "model.gen.go", buf.Bytes())
	if err != nil {
		return errors.WithMessagef(err, "failed to write models")
	}

	// schema file
	buf.Reset()
	sd := schemaDefinition{
		DB:      dbName,
		Package: schemaPkg,
		Imports: importSpecs([]goImport{importXdbSchema, importXsql}),
		Dialect: dialect,
		Tables:  tableInfos,
		Defs:    tableDefs,
	}
	err = schemaHeaderCodeTemplate.Execute(buf, sd)
	if err != nil {
		return errors.WithMessagef(err, "failed to generate schema")
	}
	for _, ctd := range tableDefs {
		err = schemaCodeTemplate.Execute(buf, ctd)
		if err != nil {
			return errors.WithMessagef(err, "failed to generate schema")
		}
	}
	err = a.writeCode(ctx, a.OutSchema, "schema.gen.go", buf.Bytes())
	if err != nil {
		return errors.WithMessagef(err, "failed to write schema")
	}
	return nil
}

// writeCode formats the generated code and writes it to dir/fileName,
// or to the command output when dir is empty.
func (a *GenerateCmd) writeCode(ctx *cli.Cli, dir, fileName string, data []byte) error {
	code, err := format.Source(data)
	if err != nil {
		return errors.WithMessagef(err, "failed to format code: %s", string(data))
	}
	if dir == "" {
		_, err = ctx.Writer().Write(code)
		return errors.WithStack(err)
	}
	if err = os.MkdirAll(dir, 0777); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(os.WriteFile(filepath.Join(dir, fileName), code, 0666))
}

// userImport returns the package of an --imports value: a path, whose
// default name is the package name, or "name path".
func userImport(imp string) goImport {
	if name, p, ok := strings.Cut(strings.TrimSpace(imp), " "); ok {
		return goImport{name: name, path: strings.Trim(strings.TrimSpace(p), `"`)}
	}
	return goImport{name: defaultPackageName(imp), path: imp}
}

// defaultPackageName returns the package name of an import path by
// convention: its last element, without a version suffix (lib/v2, yaml.v3).
func defaultPackageName(importPath string) string {
	name := path.Base(importPath)
	if strings.HasPrefix(name, "v") && path.Dir(importPath) != "." {
		if _, err := strconv.Atoi(name[1:]); err == nil {
			name = path.Base(path.Dir(importPath))
		}
	}
	name, _, _ = strings.Cut(name, ".")
	return name
}

// typePackages returns the names of the packages that a Go type expression
// refers to: pb for pb.Role_Enum, *pb.Message, []pb.Enum or
// map[string]pb.Value. A type that does not parse refers to none.
func typePackages(typ string) []string {
	expr, err := parser.ParseExpr(typ)
	if err != nil {
		return nil
	}
	var names []string
	ast.Inspect(expr, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			names = append(names, id.Name)
		}
		return false
	})
	return names
}

// importSpecs returns the import specs sorted by path, without duplicates:
// a package imported twice under one name is one import, under two names
// two.
func importSpecs(imports []goImport) []string {
	stdslices.SortFunc(imports, func(a, b goImport) int {
		return cmp.Or(strings.Compare(a.path, b.path), strings.Compare(a.name, b.name))
	})
	imports = stdslices.Compact(imports)
	specs := make([]string, len(imports))
	for i, imp := range imports {
		specs[i] = imp.spec()
	}
	return specs
}

// importPath returns the import path of the package in dir, from the module
// of the nearest go.mod above it, or "" when dir is empty or not in a module.
func importPath(dir string) string {
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for d := abs; ; {
		if data, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			mod := modulePath(data)
			rel, err := filepath.Rel(d, abs)
			if mod == "" || err != nil {
				return ""
			}
			if rel == "." {
				return mod
			}
			return mod + "/" + filepath.ToSlash(rel)
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// modulePath returns the module path declared in a go.mod file.
func modulePath(gomod []byte) string {
	for line := range strings.Lines(string(gomod)) {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "module" {
			return strings.Trim(f[1], "\"`")
		}
	}
	return ""
}
