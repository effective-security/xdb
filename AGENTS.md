# RULES OF CONDUCT

This module (`github.com/effective-security/xdb`) is a public Go library
for SQL services: a `database/sql` provider with transactions and
migrations, NULL-aware value types (`ID`, `Time`, `NULLString`, ...),
query parameters and pagination, the `xsql` query builder, schema metadata
and the `xdbcli` model generator, flake IDs and PostgreSQL notifications.
Services generate their models with `xdbcli` and build every query with `xsql`,
so a change here changes their code and SQL.

Package boundaries and direct internal imports are listed in
[`Documentation/codemap.md`](Documentation/codemap.md).

## NAVIGATION

1. Read the matching concept row in
   [`Documentation/codemap.md`](Documentation/codemap.md), then that
   package's entry points, invariants and test notes. Read only the
   relevant sections.
2. How clients use the library:
   [`Documentation/design/QUERYBUILDER.md`](Documentation/design/QUERYBUILDER.md)
   (CRUD with generated schema, `QueryParams` and `xsql`) and
   [`Documentation/design/SCHEMA_GENERATOR.md`](Documentation/design/SCHEMA_GENERATOR.md)
   (`xdbcli`, `typesmap.yaml`, generated code). The `xsql` builder API is in
   [`xsql/README.md`](xsql/README.md).
3. Prefer codebase-memory-mcp for code discovery when available, with
   `project="xdb"` ([`Documentation/MCP.md`](Documentation/MCP.md)):
   `search_graph`, `trace_path`, `get_code_snippet`, `query_graph`,
   `get_architecture`. Check the index status before trusting a miss; do not
   reindex unless asked.
4. Read the owning source and its tests before expanding scope. Fall back to
   `rg` for literals, docs and configuration, scoped to the owning package
   first. Add a missing ownership or invariant to the codemap in the same
   change.
5. Read `Documentation/FINDINGS.md`, when you have it, before changing
   surprising behavior: it is the local, gitignored list of known defects.
   [`Documentation/ROADMAP.md`](Documentation/ROADMAP.md) owns planned work.
6. Skip `Documentation/api/` and `Documentation/cli/` (`make docs` generates
   them) and `testdata/e2e/` (`make gen-sql-schema` output, gitignored).
   Edit source comments and regenerate instead.
7. Preserve existing staged and unstaged work. Documentation-only work must
   not recreate databases, regenerate artifacts or rewrite fixtures merely to
   inspect them.

## CODING GUIDELINES

### Library contract

- Exported names, value semantics, rendered SQL and generated code are the
  contract with clients. A change to any of them is a breaking change: say
  so in the PR, add the migration steps to `Documentation/ROADMAP.md`, and
  build at least one client against the change (below).
- Build a client against the working tree without editing it, through a
  workspace file outside both repositories:

  ```sh
  printf 'go 1.27.2\n\nuse (\n\t/path/to/client\n\t/path/to/xdb\n)\n' > /tmp/xdb.work
  GOWORK=/tmp/xdb.work go -C /path/to/client vet ./...
  ```

- Generator changes (`internal/cli/schema`) are contract changes: regenerate
  a client's `model.gen.go` and `schema.gen.go`, run `goimports`, build it
  and diff the output.
- Keep PostgreSQL and SQL Server both working where a package supports both
  (`schema`, `migrate`, `Open`, `xsql` dialects); their tests need the local
  stack.

### Style

- Target Go 1.27 (`go.mod`). Use the standard library where it covers a
  helper: `slices`, `maps`, `cmp.Or`, `min`/`max`, range-over-int. Prefer
  the stdlib `slices` over `github.com/effective-security/x/slices`. Do not
  add calls to `Deprecated` functions; `.golangci.yml` does not report
  SA1019 yet.
- Use `any`, not `interface{}`. Define constants instead of repeating
  literal strings or numbers.
- Split struct and map literals one key per line.
- Match the surrounding code: comment density, naming, receiver names.

### Value types and NULL

- The zero value of a NULL-aware type writes SQL NULL and a NULL scans back
  to the zero value: `ID`, `ID32`, `Int64`, `Int32`, `Float`, `Bool`
  (`false` is NULL), `NULLString`, `UUID`, `Time`, `Metadata` and `KVSet`
  (empty is NULL), `Strings` (nil is NULL). A new type follows the same
  rule, with `Value` on the value receiver and `Scan` on the pointer.
- Values are immutable and safe to copy and share between goroutines. Do
  not put a lazily written cache in a value type or in a struct that
  services share (generated `TableInfo`, dialects): compute it at
  construction or synchronize it, and add a test that races under
  `go test -race`.
- `ID` is `type ID uint64`: comparable with `==`, ordered with `<`, usable
  as a map key. `String()` of the zero ID is empty, `%d` prints `0`.

### SQL and query builder

- Bind every value through a `?` placeholder; never format a value into
  SQL. The Postgres dialect numbers `?` in rendered clause order (SET,
  WHERE, ORDER BY, LIMIT, OFFSET, RETURNING), not call order, and passes a
  literal `$n` through unchanged: do not mix the two in one statement.
- A query cached with `GetOrCreateQuery` keeps only the SQL. The caller
  passes the arguments, including LIMIT and OFFSET values, in rendered
  order; the cache key must change whenever the SQL changes, which is what
  `QueryParams.Name()` encodes.
- Use `OnConflict` for upserts: it has a fixed slot before RETURNING, while
  `Clause` lands after the last clause added.
- `QueryParamsBuilder` panics on programmer errors (position above 63, any
  argument after `SetPage`/`SetCursor`); keep pagination the last call.

### Errors

- Use `github.com/cockroachdb/errors` for creation and wrapping; wrap
  external errors with `WithMessage`/`Wrapf` and keep the stack
  (`WithStack`) on returned driver errors.
- A missing row is `sql.ErrNoRows`; `CheckNotFoundError` turns it into
  `*ErrorNotFound`, and callers test with `IsNotFoundError` or `errors.Is`.
- Library code returns errors. The intentional panics are `Must*`,
  `GetQueryParams` without params, `PageParam` with an unsupported type,
  `QueryParamsBuilder` misuse and the generator on an unknown column type.

### Logging

- Use package-level `xlog` loggers with `logger.KV`; snake_case keys
  (`migrate` still logs some camelCase keys: do not add more). Never log
  connection strings or passwords.

### Tests

- Use `assert`/`require` from testify; `package foo_test` for black-box
  tests, `package foo` only for unexported seams (`internal_test.go`).
- Tests that open PostgreSQL (`127.0.0.1:15433`) or SQL Server
  (`127.0.0.1:11434`) need `make start-localstack start-sql`; the DSNs are
  constants in the tests. A `database/sql` round trip that needs no server
  uses in-memory sqlite (`xsql` executor tests, `TestIDSQL`).
- Cover shared state with a test that actually races, and run
  `make test RACE=true` (or `go test -race ./...`) before proposing a change
  to it.
- Generator tests use `mocks/mockschema` with JSON fixtures in
  `internal/cli/schema/testdata`; `make generate` regenerates the mocks
  after an interface change in `db.go` or `schema/schema.go`.

### Tools

- `make tools` : install golangci-lint, govulncheck, cov-report, mockgen and
  gomarkdoc
- `make build` : build `bin/xdbcli`
- `make generate` : `go generate ./...` (gomock)
- `make fmt` / `make lint` : formatting, vet and golangci-lint
- `make test` (`RACE=true`), `make testshort`, `make covtest`
- `make start-localstack start-sql` / `make drop-sql` : local PostgreSQL and
  SQL Server with `testdb`
- `make gen-sql-schema` : regenerate `testdata/e2e` with the `xdbcli` on
  `PATH` (install it or run `bin/xdbcli` by hand after `make build`)
- `make build docs` : regenerate `Documentation/api/` and
  `Documentation/cli/`

CI (`.github/workflows/unittest.yml`) runs `make vars tools generate
start-localstack start-sql`, then `make build covtest`, on pushes to `main`
and on PRs, and reports coverage against `MIN_TESTCOV` (80). It does not run
lint or the race detector: run both locally. `settag.yml` is a manual
workflow that tags `<.VERSION>.<commit count>`.

### Documentation

- Document every exported symbol: what it is for, not just its name. Keep
  samples in `doc.go`, `README.md`, `xsql/README.md` and
  `Documentation/design/` accurate; compile a changed sample.
- Update [`Documentation/codemap.md`](Documentation/codemap.md) in the same
  change when you add, move or rename a package, file or exported entry
  point, or change an invariant: NULL mapping, process-global state and
  caches, panics, placeholder or clause order, generated names, supported
  drivers.
- When you find a defect you are not fixing, add it to your local
  `Documentation/FINDINGS.md` with the next free ID; committed files do not
  cite finding IDs.

## REPOSITORY MAP

Start here instead of grepping the tree.

- **[`Documentation/codemap.md`](Documentation/codemap.md)** — concept
  index, per-package files, entry points and invariants, dependencies, tests,
  build and CI.
- **[`Documentation/design/QUERYBUILDER.md`](Documentation/design/QUERYBUILDER.md)**
  — client CRUD guide: provider, query functions, `QueryParams`,
  pagination, transactions, recipes and pitfalls.
- **[`Documentation/design/SCHEMA_GENERATOR.md`](Documentation/design/SCHEMA_GENERATOR.md)**
  — `xdbcli` commands, `typesmap.yaml`, type mapping, generated code and the
  regeneration workflow.
- **[`xsql/README.md`](xsql/README.md)** — `xsql` builder reference.
- **[`Documentation/ROADMAP.md`](Documentation/ROADMAP.md)** — planned work.
- **`Documentation/FINDINGS.md`** — local, gitignored list of known defects.
- **[`Documentation/MCP.md`](Documentation/MCP.md)** — codebase-memory-mcp
  usage.
- **[`README.md`](README.md)** — install and quick start.
