# SQL Schema Generator (xdbcli)

`xdbcli` reads a live PostgreSQL or SQL Server schema and prints it, or
generates the Go code that queries use (see [QUERYBUILDER.md](QUERYBUILDER.md)):

- `model.gen.go`: one row struct per table or view, with `ScanRow`, a
  `<Model>Slice` and a `<Model>Result` for the `xdb.Execute*` helpers.
- `schema.gen.go`: one `xdbschema.TableInfo` per table (`<Model>TableInfo`),
  one column-definition variable per table (`<Model>`), the `Dialect` and a
  `<Db>Tables` map.

The generated files are never edited by hand: change the migrations or
`typesmap.yaml` and regenerate. Code: `internal/cli/schema` (commands and
templates), `schema` (introspection), `cmd/xdbcli`.

## Install and connect

```sh
go install github.com/effective-security/xdb/cmd/xdbcli@latest   # or: make build -> bin/xdbcli
```

The connection comes from `--sql-source` or `XDB_DATASOURCE`, as a URL with
a query string, or a `file://` / `env://` reference to one; `--db` selects
the database:

```sh
export XDB_DATASOURCE="postgres://postgres:postgres@127.0.0.1:15433?sslmode=disable"
xdbcli schema tables --db testdb
xdbcli --sql-source=file://etc/dev/sql-conn.txt schema columns --db mydb --schema myschema
```

## Commands

| Command | Prints / writes | Flags besides `--db` |
| --- | --- | --- |
| `schema tables` | tables, `--views` adds views | `--schema`, `--table`, `--views` |
| `schema views` | views | `--schema`, `--view` |
| `schema columns` | columns per table: type, NULL, max length, FK reference | `--schema`, `--table`, `--dependencies`, `--views` |
| `schema foreign-keys` | foreign keys | `--schema`, `--table` |
| `schema generate` | `model.gen.go`, `schema.gen.go` (stdout without `--out-*`) | below |

Global flags: `-D/--debug`, `--o=table|json|yaml`, `--sql-source`. List
flags take repeated values or a comma-separated list (`--view a,b`).

`schema generate` flags:

| Flag | Effect |
| --- | --- |
| `--schema` | schemas to read; generate each schema into its own `--out-model`/`--out-schema` folders (see [Several schemas](#several-schemas)) |
| `--table` | tables to generate; default all tables of the schemas |
| `--view` | views to add; views are never included otherwise |
| `--dependencies` | also every table reachable through foreign keys, in any schema |
| `--out-model`, `--out-schema` | output folders; the package name is the folder name |
| `--pkg-model`, `--pkg-schema` | override the package names |
| `--types-def` | path to `typesmap.yaml` (YAML or JSON) |
| `--imports` | packages of the types used in `typesmap.yaml` other than `xdb`, `values` and `pq`, which are known (for example the `pb` package of enums, or `time` for `time.Time`), as a path or `name path`; written to the model file only when a column type refers to the package, also inside `*T`, `[]T` or `map[K]V`. A package named `pq` or `values` here replaces the known one; a package is imported once per name |
| `--struct-suffix` | suffix for model struct and column-definition names |
| `--use-schema` | prefix model and column-definition names with the schema, except for `public` and `dbo`; `<Model>TableInfo` names and the tables map keep the bare table name |

## Workflow in a service

The schema lives in forward migrations (`N_name.up.sql` with a matching
`.down.sql`). To regenerate after a change, recreate a local database from
the migrations, then generate from it:

```make
gen-sql-schema:
	echo '```' > internal/db/README.md
	xdbcli --sql-source=file://etc/dev/sql-conn.txt \
		schema columns --dependencies --schema myschema --db mydb >> internal/db/README.md
	echo '```' >> internal/db/README.md
	rm -f internal/db/model/*.gen.go internal/db/schema/*.gen.go
	xdbcli --sql-source=file://etc/dev/sql-conn.txt \
		schema generate --db mydb --schema myschema \
		--types-def internal/db/schema/typesmap.yaml \
		--imports github.com/example/service/api/pb \
		--dependencies \
		--out-model internal/db/model \
		--out-schema internal/db/schema \
		--view vw_membership_info,vw_other
```

1. Recreate the database (drop, create, run the service's migrations or
   `--dry-run`). golang-migrate records applied versions: editing an applied
   migration has no effect until the database is recreated.
2. Run the target, then `go build ./...` and review the diff of the
   generated files and of `internal/db/README.md`.
3. The files compile as written: the generator imports exactly the packages
   each file uses, including the sibling `schema` package of the model,
   whose import path it reads from the nearest `go.mod` above
   `--out-schema`, and the `--imports` packages the column types refer to.
   Outside a module the sibling import is left out, and a typesmap type of
   a package missing from `--imports` is not imported; goimports fixes
   both. Otherwise running goimports changes nothing and is optional.
4. When the filters match no table, the command writes no file.

### Several schemas

Generate each schema into its own pair of folders, with one
`xdbcli schema generate --schema <name>` run per target (for example
`internal/db/model` + `internal/db/schema` and `internal/db/audit/model` +
`internal/db/audit/schema`). Names are not qualified by the schema:
`<Model>TableInfo`, the column definitions and the keys of `<Db>Tables` come
from the table name, so two schemas with a table of the same name in one
target do not compile, even with `--use-schema`. Tables of several schemas
share one target only when their names differ, for example a table of
another schema that `--dependencies` adds through a foreign key.

## typesmap.yaml

```yaml
tables:            # schema.table -> Go name of the model and column defs
  myschema.apikey: APIKey
fields:            # schema.table.column -> Go field name
  myschema.event.ref_id: ReferenceID
types:             # column -> Go type, as written in the generated code
  metadata: xdb.Metadata                   # bare column name: every table
  _count: uint32                           # NOT NULL int4 columns named *_count
  myschema.org.status: pb.ItemStatus_Enum  # one column (needs --imports for pb)
  myschema.issuer.config: JSON             # a handwritten type of the model package
  myschema.vw_membership_info.name: string # a view column known to be NOT NULL
with_cached_props: # schema.table: adds cachedProps values.MapAny and Cached()
  - myschema.org
drop_columns:      # bare column names left out of every table
  - legacy_column
```

A column's type is the first match of:

1. `types` by bare column name (so a qualified key cannot override a bare
   one),
2. `types` by `schema.table.column`,
3. `types._count` for a NOT NULL `int4` column named `*_count`,
4. arrays: `_int8` is `xdb.IDArray` for a column named `*_ids`, otherwise
   `pq.Int64Array`; `_text` and `_varchar` are `pq.StringArray`; any other
   array panics,
5. the defaults below, trying the `NULL` variant first for a nullable
   column. A column named `id`, `*_id`, `*Id` or `*ID` looks up `id <type>`
   first.

An unmapped type panics with `don't know how to convert type`: add a
`types` entry.

| SQL type | NOT NULL | NULL |
| --- | --- | --- |
| `bigint`/`int8` named like an ID | `xdb.ID` | `xdb.ID` |
| `int`/`int4` named like an ID | `xdb.ID32` | `xdb.ID32` |
| `bigint`, `int8` | `int64` | `xdb.Int64` |
| `int`, `int4` / `int2`, `smallint` / `tinyint` | `int32` / `int16` / `int8` | `xdb.Int32` |
| `decimal`, `numeric`, `float8` / `real`, `float4` | `float64` / `float32` | `xdb.Float` |
| `bool`, `boolean`, `bit` | `bool` | `xdb.Bool` |
| `varchar`, `char`, `bpchar`, `text`, `nchar`, `nvarchar`, `cidr`, `inet`, `ltree` | `string` | `xdb.NULLString` |
| `uuid`, `uniqueidentifier` | `xdb.UUID` | `xdb.UUID` |
| `date`, `time`, `timestamp`, `timestamptz`, `datetime`, `datetime2` | `xdb.Time` | `xdb.Time` |
| `jsonb` | `xdb.NULLString` | `xdb.NULLString` |
| `bytea` | `[]byte` | `[]byte` |

The NULL-aware types write their zero value as NULL; see
[QUERYBUILDER.md](QUERYBUILDER.md#pitfalls).

## Generated code

For table `myschema.org` (struct names are the singular PascalCase of the
table name unless `tables` overrides them):

```go
// model package
type Org struct {
	ID   xdb.ID `db:"id,int8,index,primary" json:",omitempty"`
	Name string `db:"name,varchar,max:64,index" json:",omitempty"`
	// ...
}
func (m *Org) ScanRow(rows xdb.Row) error // scans the columns in TableInfo order
type OrgSlice []*Org
type OrgResult struct {
	Rows        OrgSlice
	NextOffset  uint32
	HasNextPage bool
	Cursor      string
}
// SetResult, SetResultWithCursor, GetTableInfo

// schema package
var Dialect = xsql.Postgres
var OrgTableInfo = xdbschema.TableInfo{SchemaName: "myschema.org", Name: "org", PrimaryKey: "id", Columns: []string{"id", "name" /* ... */}, Dialect: xsql.Postgres}
var Org = struct {
	Table *xdbschema.TableInfo
	ID    xdbschema.Column // id bigint
	Name  xdbschema.Column // name character varying
}{ /* Name, Position, Type, UdtType, Nullable, MaxLength */ }
var MydbTables = map[string]*xdbschema.TableInfo{"org": &OrgTableInfo /* ... */}
```

The `db` tag lists the column name, its type, `max:N`, `null`, `index` and
`primary`. The tables are in the order of the introspection, sorted by
`schema.table`, then the views.

## Views

Views are generated only through `--view`. PostgreSQL reports every view
column as nullable, so text columns become `xdb.NULLString` and integers
`xdb.Int64`/`xdb.Int32`. When the view guarantees a value (`COALESCE(x, '')`
or a NOT NULL source), map the column to the plain type in `types`;
scanning NULL into a plain `string` fails.

## Changing the generator

- Templates: `internal/cli/schema/template.go`; type mapping:
  `internal/cli/schema/types.go`; naming and file output: `schema.go`.
- Unit tests run the commands on mocked introspection
  (`internal/cli/schema/schema_test.go`, JSON fixtures in `testdata/`);
  type mapping tests are in `template_test.go`; `generate_test.go` checks
  the imports and declarations of files generated into a module.
- End to end: `make start-localstack start-sql build`, then generate into
  `testdata/e2e` (`make gen-sql-schema` uses the `xdbcli` on `PATH`) and
  build it. Generated code is a client contract: regenerate a service and
  build it before merging (see AGENTS.md, Library contract).
