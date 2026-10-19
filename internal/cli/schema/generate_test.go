package schema

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/effective-security/xdb/internal/cli"
	dbschema "github.com/effective-security/xdb/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModulePath(t *testing.T) {
	assert.Equal(t, "example.com/svc", modulePath([]byte("// comment\nmodule example.com/svc // x\n\ngo 1.27\n")))
	assert.Equal(t, "example.com/q", modulePath([]byte(`module "example.com/q"`)))
	assert.Empty(t, modulePath([]byte("go 1.27\n")))
}

func TestImportPath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/svc\n"), 0600))

	assert.Equal(t, "example.com/svc/internal/db/schema", importPath(filepath.Join(dir, "internal/db/schema")))
	assert.Equal(t, "example.com/svc", importPath(dir))
	assert.Empty(t, importPath(""))
}

func TestUserImport(t *testing.T) {
	tcases := []struct {
		imp, name, spec string
	}{
		{"github.com/example/svc/api/pb", "pb", `"github.com/example/svc/api/pb"`},
		{"github.com/example/lib/v2", "lib", `"github.com/example/lib/v2"`},
		{"gopkg.in/yaml.v3", "yaml", `"gopkg.in/yaml.v3"`},
		{`apipb "github.com/example/svc/api"`, "apipb", `apipb "github.com/example/svc/api"`},
		{"apipb github.com/example/svc/api", "apipb", `apipb "github.com/example/svc/api"`},
	}
	for _, tc := range tcases {
		imp := userImport(tc.imp)
		assert.Equal(t, tc.name, imp.name, tc.imp)
		assert.Equal(t, tc.spec, imp.spec(), tc.imp)
	}
}

// TestGenerateFiles generates two schemas into a module and checks that each
// package has one file with exact imports and no duplicate declarations.
func TestGenerateFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/svc\n"), 0600))

	column := func(table, name, typ, udt string, nullable bool) *dbschema.Column {
		return &dbschema.Column{
			Name:       name,
			Type:       typ,
			UdtType:    udt,
			Nullable:   nullable,
			SchemaName: table + "." + name,
		}
	}
	tables := dbschema.Tables{
		{
			Schema:     "audit",
			Name:       "event",
			SchemaName: "audit.event",
			Columns: dbschema.Columns{
				column("audit.event", "id", "bigint", "int8", false),
				column("audit.event", "tags", "ARRAY", "_text", true),
			},
		},
		{
			Schema:     "public",
			Name:       "org",
			SchemaName: "public.org",
			Columns: dbschema.Columns{
				column("public.org", "id", "bigint", "int8", false),
				column("public.org", "name", "character varying", "varchar", false),
			},
		},
	}

	cmd := GenerateCmd{
		DB:        "svcdb",
		OutModel:  filepath.Join(dir, "internal/db/model"),
		OutSchema: filepath.Join(dir, "internal/db/schema"),
		UseSchema: true,
		Imports:   []string{"github.com/example/svc/api/pb"},
	}
	require.NoError(t, cmd.generate(&cli.Cli{}, "postgres", "svcdb", tables))

	model := parseGenerated(t, filepath.Join(cmd.OutModel, "model.gen.go"))
	assert.Equal(t, []string{
		"example.com/svc/internal/db/schema",
		"github.com/cockroachdb/errors",
		"github.com/effective-security/xdb",
		"github.com/effective-security/xdb/schema",
		"github.com/effective-security/xdb/xsql",
		"github.com/lib/pq",
	}, model.imports, "pb is not used by any column")
	assert.Contains(t, model.decls, "AuditEvent")
	assert.Contains(t, model.decls, "Org")

	schema := parseGenerated(t, filepath.Join(cmd.OutSchema, "schema.gen.go"))
	assert.Equal(t, []string{
		"github.com/effective-security/xdb/schema",
		"github.com/effective-security/xdb/xsql",
	}, schema.imports)
	assert.Contains(t, schema.decls, "SvcdbTables")

	entries, err := os.ReadDir(cmd.OutModel)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "one model file for both schemas")
}

type generatedFile struct {
	imports []string
	// specs are the imports as `name path`, or `path` without a name
	specs []string
	decls []string
}

// parseGenerated parses a generated file and fails on a duplicate top-level
// declaration.
func parseGenerated(t *testing.T, fn string) generatedFile {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), fn, nil, 0)
	require.NoError(t, err)

	var res generatedFile
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		require.NoError(t, err)
		res.imports = append(res.imports, p)
		spec := p
		if imp.Name != nil {
			spec = imp.Name.Name + " " + p
		}
		res.specs = append(res.specs, spec)
	}
	seen := map[string]bool{}
	add := func(name string) {
		require.False(t, seen[name], "duplicate declaration %s", name)
		seen[name] = true
		res.decls = append(res.decls, name)
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				add(d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					add(s.Name.Name)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						add(n.Name)
					}
				}
			}
		}
	}
	return res
}

func TestTypePackages(t *testing.T) {
	tcases := []struct {
		typ string
		exp []string
	}{
		{"pb.Role_Enum", []string{"pb"}},
		{"*pb.Message", []string{"pb"}},
		{"[]pb.Enum", []string{"pb"}},
		{"map[string]pb.Value", []string{"pb"}},
		{"map[pb.Key][]*x.Value", []string{"pb", "x"}},
		{"*pq.StringArray", []string{"pq"}},
		{"JSON", nil},
		{"[]byte", nil},
		{"not a type(", nil},
	}
	for _, tc := range tcases {
		assert.Equal(t, tc.exp, typePackages(tc.typ), tc.typ)
	}
}

// TestGenerateImportsFromTypes checks the imports of typesmap types, nested
// in pointers, slices and maps, without cached properties.
func TestGenerateImportsFromTypes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/svc\n"), 0600))
	typesDef := filepath.Join(dir, "typesmap.yaml")
	require.NoError(t, os.WriteFile(typesDef, []byte(`types:
  public.widget.xmeta: values.MapAny
  public.widget.xlabels: "[]*pb.Label"
  public.widget.xtags: "*pq.StringArray"
`), 0600))

	column := func(name, typ, udt string) *dbschema.Column {
		return &dbschema.Column{Name: name, Type: typ, UdtType: udt, SchemaName: "public.widget." + name}
	}
	tables := dbschema.Tables{
		{
			Schema:     "public",
			Name:       "widget",
			SchemaName: "public.widget",
			Columns: dbschema.Columns{
				column("id", "bigint", "int8"),
				column("xmeta", "jsonb", "jsonb"),
				column("xlabels", "jsonb", "jsonb"),
				column("xtags", "ARRAY", "_text"),
			},
		},
	}

	cmd := GenerateCmd{
		DB:        "svcdb",
		OutModel:  filepath.Join(dir, "db/model"),
		OutSchema: filepath.Join(dir, "db/schema"),
		TypesDef:  typesDef,
		Imports:   []string{"github.com/example/svc/api/pb"},
	}
	require.NoError(t, cmd.generate(&cli.Cli{}, "postgres", "svcdb", tables))

	model := parseGenerated(t, filepath.Join(cmd.OutModel, "model.gen.go"))
	assert.Equal(t, []string{
		"example.com/svc/db/schema",
		"github.com/cockroachdb/errors",
		"github.com/effective-security/x/values",
		"github.com/effective-security/xdb",
		"github.com/effective-security/xdb/schema",
		"github.com/effective-security/xdb/xsql",
		"github.com/example/svc/api/pb",
		"github.com/lib/pq",
	}, model.imports)
}

func TestGenerateNoTables(t *testing.T) {
	dir := t.TempDir()
	cmd := GenerateCmd{
		DB:        "svcdb",
		OutModel:  filepath.Join(dir, "model"),
		OutSchema: filepath.Join(dir, "schema"),
	}
	require.NoError(t, cmd.generate(&cli.Cli{}, "postgres", "svcdb", nil))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "no files for an empty match")
}

// TestGenerateImportAliases checks that a package named by --imports is
// imported once under each name the column types use.
func TestGenerateImportAliases(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/svc\n"), 0600))
	typesDef := filepath.Join(dir, "typesmap.yaml")
	require.NoError(t, os.WriteFile(typesDef, []byte(`types:
  public.gadget.xalias: lpq.StringArray
`), 0600))

	column := func(name, typ, udt string) *dbschema.Column {
		return &dbschema.Column{Name: name, Type: typ, UdtType: udt, SchemaName: "public.gadget." + name}
	}
	tables := dbschema.Tables{
		{
			Schema:     "public",
			Name:       "gadget",
			SchemaName: "public.gadget",
			Columns: dbschema.Columns{
				column("id", "bigint", "int8"),
				// pq.StringArray by the default mapping
				column("xtags", "ARRAY", "_text"),
				column("xalias", "ARRAY", "_text"),
			},
		},
	}

	cmd := GenerateCmd{
		DB:        "svcdb",
		OutModel:  filepath.Join(dir, "db/model"),
		OutSchema: filepath.Join(dir, "db/schema"),
		PkgSchema: "dbschema",
		TypesDef:  typesDef,
		Imports:   []string{"pq github.com/lib/pq", "lpq github.com/lib/pq"},
	}
	require.NoError(t, cmd.generate(&cli.Cli{}, "postgres", "svcdb", tables))

	model := parseGenerated(t, filepath.Join(cmd.OutModel, "model.gen.go"))
	assert.Equal(t, []string{
		"dbschema example.com/svc/db/schema",
		"github.com/cockroachdb/errors",
		"github.com/effective-security/xdb",
		"xdbschema github.com/effective-security/xdb/schema",
		"github.com/effective-security/xdb/xsql",
		"github.com/lib/pq",
		"lpq github.com/lib/pq",
	}, model.specs)
}
