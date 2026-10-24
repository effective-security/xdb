# RELEASE NOTES

Changes per `.VERSION`, with the action a client takes. Tags are
`<.VERSION>.<commit count>` (`settag.yml`), so every tag of a version carries
the changes listed under it up to that commit.

## v1.0

### Breaking: `xdb.ID` is a value

`xdb.ID` is `type ID uint64`, an immutable value: comparable with
`==`, ordered, a valid map key and safe to share between goroutines.
Clients replace `xdb.ID{}` with `xdb.ID(0)`; `Set` and `Reset` on a copy
no longer change the original.

**JSON wire format:** a zero ID in a field tagged `omitempty` is now
omitted, where the struct-backed ID always wrote `0`. Generated models tag
every column `json:",omitempty"`, so a model marshaled to JSON leaves out
its zero IDs (an unset `ProjectID`, the `OrgID` of a platform row).
Decoding a missing field gives the zero ID, so JSON round trips keep their
values.

**Action:** replace `xdb.ID{}` with `xdb.ID(0)` (or `0`). Workarounds for
the former string cache can go: comparing IDs with `UInt64()` in tests,
formatting shared IDs with `xdb.IDString(id.UInt64())`, avoiding
`String()`/`Pb()` on rows shared between goroutines. Where models are
marshaled to JSON directly (API responses, events, caches, golden files),
expect zero IDs to be absent; a consumer that needs the field present
marshals its own type, such as the protobuf message, instead of the model.

### New: keyset cursors and row iterators

- `xdb.Cursor`, `NewCursor(key...)`, `ParseCursor(s)` and
  `Cursor.Scan(dest...)` encode the sort key of the last row of a page into
  an opaque URL-safe token, at full precision (decimal integers, RFC 3339
  times with nanoseconds), and decode it into typed values; a key type with
  `MarshalText`/`UnmarshalText` uses them, before its kind. `NewCursor`
  returns an error when a key does not encode (a failing `MarshalText`, an
  unsupported type) rather than a token `Scan` cannot read back; Time,
  integer and string keys never fail.
- A query whose only argument is the limit passes `QueryParams`
  (`NewQueryParams(name).SetCursor(limit, pos)` for an unfiltered first
  page): the paging helpers do not take a lone positional argument for the
  limit, since it can be a filter value.
- `QueryParamsBuilder.SetCursor(limit, pos, key...)` takes a multi-column
  key (`created_at, id`); without key values it is `SetPage(limit, 0)`, the
  first page. Single-value calls compile unchanged.
- `xdb.QueryRows[T](ctx, db, query, args...)` returns `iter.Seq2[*T, error]`
  for streaming rows; the list helpers use it.
- `EncodeCursor` and `DecodeCursor` (`values.MapAny`) are deprecated.

**Action:** none required. To add or move cursor pagination, follow
[Cursor (keyset) pagination](design/QUERYBUILDER.md#cursor-keyset-pagination).
Replacing a cursor helper such as
`xdb.EncodeCursor(values.MapAny{"id": id.String()})` with
`xdb.NewCursor(id)` changes the token format: cursors that API
clients hold from before stop parsing, and those listings restart from the
first page. Do not encode times with `Time.String()`, which keeps
milliseconds only and skips rows.

### Behavior changes

| Change | Action |
| --- | --- |
| `ExecuteListQuery`, `ExecuteQuery` and the paging helpers return the `rows.Err()` error; a failure in the middle of a result used to return the rows read so far without an error | none; expect an error where results were cut short |
| `ExecuteQueryWithPagination` and `ExecuteQueryWithCursor` detect the next page exactly: they bind `limit+1` in place of the limit argument and drop the extra row, so a listing that ends on a full page reports `HasNextPage = false` with no `NextOffset` or `Cursor`; a full page used to report a next page, which could be empty | fix tests that expect `HasNextPage` (or a cursor) after the last full page; check that no paged query applies its LIMIT inside a subquery or CTE that the outer query reorders |
| `ExecuteQueryWithPagination` and `ExecuteQueryWithCursor` report no next page for a zero limit (QueryParams without `SetPage`); they reported one for every result | none; a loop on `HasNextPage` over such a query now ends |
| `SQLProvider.BeginTx` passes its `*sql.TxOptions` to the driver; it used `nil` | check calls with non-nil options: a `ReadOnly` transaction now rejects writes, an isolation level now applies |
| `QueryParamsBuilder.Name` is recomputed after every change; it was frozen at the first call | none |
| `SetNullColums` keeps a sorted copy; it used to sort the caller's slice in place, at the first `Name` call | none; a shared (package-level) slice is no longer reordered |
| `QueryParamsBuilder.Reset` also clears the null columns | none |
| `TableInfo.AllColumns` no longer caches its result in the shared table; it was a data race | optional: remove the `init` loops that warm `AllColumns` |
| `Provider.Close` stops the keep-alive goroutine; `Open` and `NewProvider` close the DB when the ping or the migrations fail | none |
| `Open` adds the database name with `?` when the URL has no query string | none |
| `NewProvider` takes `pkg/flake.IDGenerator`; it took `x/flake.IDGenerator`, the same method set | none |
| `flake.IsTestRun` is `testing.Testing()`; it matched "test" or "debug" in the program name | none; a production binary with such a name used a machine ID derived from the program name, the same on every host, and now uses the host address |

### Schema generator (xdbcli)

| Change | Action |
| --- | --- |
| Generated files import exactly what they use: errors, xdb, xdbschema, the model's sibling `schema` package (path from the nearest `go.mod`), and `pq`, `values` and the `--imports` packages that the column types refer to, also inside `*T`, `[]T` and `map[K]V` | optional: drop `goimports` from the `gen-sql-schema` target once every package of a `typesmap.yaml` type other than `xdb`, `values` and `pq` is in `--imports` (for example `--imports time` for `time.Time`). Regenerating changes no file of a single-schema service: this was checked against two client services, byte for byte |
| `--imports` packages go to the model file only, when a column type refers to them; `name path` sets an import name, and an `--imports` package replaces the known `pq` or `values` of the same name; a package is imported once per name | none |
| One run writes one `model.gen.go` and one `schema.gen.go`, and nothing when no table matches; with several schemas it used to write per-schema files that declared `Dialect` and the tables map twice and did not compile | generate each schema into its own `--out-model`/`--out-schema` folders: names are not qualified by the schema |
| Index columns in `schema.ListTables`, `xdbcli schema columns` and generated index comments follow the index key order in the database instead of an unordered `array_agg` (PostgreSQL) or table column ordinal (SQL Server); SQL Server indexes are sorted by index name | none; regenerating may reorder multi-column index comments to match their index definition |
| `xdbcli --version` reports the build version (`make build`) or the module version (`go install ...@version`) | none |
