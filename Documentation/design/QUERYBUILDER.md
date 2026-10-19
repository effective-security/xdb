# Query Builder

How a service writes its SQL CRUD on xdb: generated table metadata and row
models (see [SCHEMA_GENERATOR.md](SCHEMA_GENERATOR.md)), a `query` package
of SQL functions and request structs, and a provider package that runs
them. The `xsql` builder API itself is in [xsql/README.md](../../xsql/README.md).

The examples use the test schema (`testdata/sql/postgres`: `public.user`,
`public.orgmember`, with `orgmember` renamed to `Member` in `typesmap.yaml`)
and were run against PostgreSQL. Rendered SQL is shown in comments.

## Layout

```text
internal/db/
  db.go         interfaces: <Name>ReadonlyDb (reads), <Name>Db (mutations)
  schema/       schema.gen.go (TableInfo, column defs), typesmap.yaml
  model/        model.gen.go (row structs, ScanRow, <Model>Result) + handwritten Validate/conversions
  pgsql/        Provider: one method per operation, runs SQL from query/
  pgsql/query/  SQL functions, request structs with QueryParams(), shared scope/pagination helpers
```

The generated code a query uses:

| Generated | Use |
| --- | --- |
| `schema.UserTableInfo` | `Select()`, `SelectAliased("u", nil)`, `InsertInto()`, `Update()`, `DeleteFrom()`, `AllColumns()`, `SchemaName` (`public.user`) |
| `schema.User.Email.Name` | column name for SQL fragments |
| `schema.User.Email.Position` | column ordinal: the variant bit for `QueryParamsBuilder.Set` |
| `model.User` | row struct; `ScanRow` scans `AllColumns()` in order |
| `model.UserResult` | `Rows`, `HasNextPage`, `NextOffset`, `Cursor`: the target of `Execute*` |

Generated `TableInfo` values are never written after initialization and are
safe to share between goroutines; they need no `init` warm-up.

## Provider and transactions

```go
p, err := xdb.NewProvider(dsn, "testdb", nil, &xdb.MigrationConfig{Source: "sql/migrations"})
```

`dsn` is a URL with a query string (`postgres://u:p@host:5432?sslmode=disable`)
or a `file://` / `env://` reference to one. A nil ID generator uses
`flake.DefaultIDGenerator`; `p.NextID()` returns a new `xdb.ID`.

A method that writes more than one row runs every statement on the
transaction provider:

```go
tx, err := p.BeginTx(ctx, nil)
if err != nil {
	return errors.WithStack(err)
}
defer tx.Close() // rolls back unless committed
// ... every statement on tx, not on p ...
return tx.Commit()
```

Transactions do not nest. `BeginTx` passes its `TxOptions` (isolation level,
read-only) to the driver. A service provider that wraps `xdb.Provider` returns its own type from
`BeginTx`; call the transaction's methods (`tx.(*Provider).AddMember`), not
the receiver's.

## Query functions

A query function renders its SQL once and returns it with its cache name:

```go
// GetUserByID returns the SQL to read one user by ID.
func GetUserByID() (string, string) {
	const key = "GetUserByID"
	return xsql.Postgres.GetOrCreateQuery(key, func(name string) xsql.Builder {
		return schema.UserTableInfo.Select().Where(schema.User.ID.Name+" = ?", nil)
	})
}
// SELECT id, email, email_verified, name, uuid FROM public.user WHERE id = $1
```

Rules:

1. `GetOrCreateQuery` caches only the SQL and discards the builder's
   arguments: write `nil` for every `?`, including `Limit(nil)` and
   `Offset(nil)`, and pass the values when executing.
2. One cache name per SQL shape. A static query uses a constant; a query
   that varies with its request uses `QueryParams.Name()`. Never let a
   request value change the SQL without changing the name.
3. Never format a value into the SQL; use `?`. Do not mix `?` with a literal
   `$1` in one statement: literals are not renumbered.
4. `create` must not `Close` the builder; `GetOrCreateQuery` does.

## QueryParams: variants and arguments

A request struct records which optional parts are present and the
arguments, in one `xdb.QueryParamsBuilder`:

| Method | Changes the SQL name | Appends an argument | Use |
| --- | --- | --- | --- |
| `Set(pos, v)` | yes (bit `pos`) | `v` | optional filter or SET column |
| `AddArgs(v...)` | no | `v...` | arguments every variant has (the `WHERE id` of an UPDATE) |
| `SetEnum(pos, v)` | yes (value) | no | a value that selects different SQL (a scope) |
| `SetFlags(f...)` | yes | no | other variants; replaces, so add with `SetFlags(append(b.GetFlags(), f)...)` |
| `SetNullColums(cols)` | yes (hash) | no | `IS NULL` filters; the builder keeps a sorted copy |
| `SetPage(limit, offset)` | `_o` when offset > 0 | limit, offset | offset pagination; must be last |
| `SetCursor(limit, pos, key...)` | yes (bit `pos`), `_c` | `key...`, limit | keyset pagination, see [below](#cursor-keyset-pagination); without key values it is `SetPage(limit, 0)`; must be last |

The arguments must follow the **rendered** placeholder order, not the call
order of the builder: SET values, then WHERE values in `Where` call order
(a cursor is a WHERE value), then LIMIT, then OFFSET. A zero limit becomes
`xdb.DefaultPageSize` (500).

```go
// UpdateUserRequest updates the set fields of a user.
type UpdateUserRequest struct {
	ID            uint64
	Name          string
	EmailVerified *bool
}

// QueryParams returns the arguments in the rendered order: SET values, then
// the WHERE id.
func (r *UpdateUserRequest) QueryParams() xdb.QueryParams {
	b := xdb.NewQueryParams("UpdateUser")
	if r.Name != "" {
		b.Set(schema.User.Name.Position, r.Name)
	}
	if r.EmailVerified != nil {
		b.Set(schema.User.EmailVerified.Position, *r.EmailVerified)
	}
	b.AddArgs(r.ID)
	return b
}

// UpdateUser returns the SQL of UpdateUserRequest.
func UpdateUser(args ...any) (string, string) {
	p := xdb.GetQueryParams(args...)
	return xsql.Postgres.GetOrCreateQuery(p.Name(), func(name string) xsql.Builder {
		q := schema.UserTableInfo.Update().
			Where(schema.User.ID.Name+" = ?", nil).
			Returning(schema.UserTableInfo.AllColumns())
		if p.IsSet(schema.User.Name.Position) {
			q.Set(schema.User.Name.Name, nil)
		}
		if p.IsSet(schema.User.EmailVerified.Position) {
			q.Set(schema.User.EmailVerified.Name, nil)
		}
		return q
	})
}
// UpdateUser_x18: UPDATE public.user SET name=$1, email_verified=$2 WHERE id = $3 RETURNING ...
```

`Where` is called before `Set` above, but SET renders first, so the ID is
the last argument.

## Executing

| Call | Returns |
| --- | --- |
| `xdb.QueryRow[model.User](ctx, db, sql, args...)` | `*model.User`; `sql.ErrNoRows` when no row |
| `xdb.QueryRows[model.User](ctx, db, sql, args...)` | `iter.Seq2[*model.User, error]`: rows one at a time, for large or streamed results |
| `xdb.ExecuteQuery(ctx, db, &rs, sql, qp)` | all rows, no next page |
| `xdb.ExecuteQueryWithPagination(ctx, db, &rs, sql, qp)` | page, `HasNextPage`, `NextOffset` |
| `xdb.ExecuteQueryWithCursor(ctx, db, encode, &rs, sql, qp)` | page, `HasNextPage`, `Cursor = encode(lastRow)` |
| `db.ExecContext(ctx, sql, args...)` | `sql.Result` for statements without rows |

`db` is any `xdb.DB`: the provider, a transaction provider, `*sql.DB` or
`*sql.Tx`. The `Execute*` helpers take the `QueryParams` itself as the only
argument, or plain arguments (for the paging helpers the last ones are the
limit and offset, or the limit); `QueryRow` and `ExecContext` take
`qp.Args()...`.

The paging helpers bind `limit+1` in place of the limit argument and drop
the extra row: `HasNextPage` is true only when another row exists, so a
listing that ends on a full page reports no next page, and `NextOffset` or
`Cursor` stay empty. The arguments of the caller and of the `QueryParams`
keep the limit. Because the extra row is the last one of the result, the
page's LIMIT must apply to the final, ordered result; do not page inside a
subquery or CTE that the outer query reorders. A zero limit (no `SetPage`)
reports no next page.

These are package functions that take an `xdb.DB`, not methods: Go 1.27
allows type parameters on methods of concrete types only, never on
interface methods, and `xdb.DB`, `xdb.Provider` and the services' wrappers
are interfaces. `QueryRows` stops at the first error and closes the rows
when the loop ends:

```go
for u, err := range xdb.QueryRows[model.User](ctx, p, q) {
	if err != nil {
		return err
	}
	// ...
}
```

A missing row becomes a typed not-found error:

```go
q, _ := query.GetUserByID()
u, err := xdb.QueryRow[model.User](ctx, p, q, id)
if err != nil {
	return nil, xdb.CheckNotFoundError(err, schema.UserTableInfo.Name, id)
}
// callers: xdb.IsNotFoundError(err)
```

## Recipes

### Insert or upsert, returning the row

```go
// RegisterUser returns the SQL of an upsert by email: args id, email, name.
func RegisterUser() (string, string) {
	const key = "RegisterUser"
	return xsql.Postgres.GetOrCreateQuery(key, func(name string) xsql.Builder {
		q := schema.UserTableInfo.InsertInto().
			OnConflict("(email) DO UPDATE SET name = EXCLUDED.name").
			Returning(schema.UserTableInfo.AllColumns())
		q.NewRow().
			Set(schema.User.ID.Name, nil).
			Set(schema.User.Email.Name, nil).
			Set(schema.User.Name.Name, nil).
			SetExpr(schema.User.EmailVerified.Name, "false")
		return q
	})
}
// INSERT INTO public.user ( id, email, name, email_verified ) VALUES ( $1, $2, $3, false )
// ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name RETURNING id, email, email_verified, name, uuid

u, err := xdb.QueryRow[model.User](ctx, p, q, p.NextID(), email, name)
```

Use `OnConflict`, not `Clause`: `OnConflict` renders before RETURNING,
`Clause` after the last clause added. `SetExpr` writes a literal value
(`Now()`, a constant) without an argument.

### List with optional filters and offset pages

```go
// ListMembersRequest lists memberships, newest first.
type ListMembersRequest struct {
	OrgID  uint64
	Role   string
	Limit  uint32
	Offset uint32
}

// QueryParams records the filters in WHERE order, then the page.
func (r *ListMembersRequest) QueryParams() xdb.QueryParams {
	b := xdb.NewQueryParams("ListMembers")
	if r.OrgID != 0 {
		b.Set(schema.Member.OrgID.Position, r.OrgID)
	}
	if r.Role != "" {
		b.Set(schema.Member.Role.Position, r.Role)
	}
	b.SetPage(r.Limit, r.Offset)
	return b
}

// ListMembers returns the SQL of ListMembersRequest.
func ListMembers(args ...any) (string, string) {
	p := xdb.GetQueryParams(args...)
	return xsql.Postgres.GetOrCreateQuery(p.Name(), func(name string) xsql.Builder {
		q := schema.MemberTableInfo.Select().
			OrderBy(schema.Member.ID.Name + " DESC").
			Limit(nil).
			Offset(nil)
		if p.IsSet(schema.Member.OrgID.Position) {
			q.Where(schema.Member.OrgID.Name+" = ?", nil)
		}
		if p.IsSet(schema.Member.Role.Position) {
			q.Where(schema.Member.Role.Name+" = ?", nil)
		}
		return q
	})
}
// ListMembers_x4: ... WHERE org_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3   [org, limit, offset]

qp := req.QueryParams()
q, _ := query.ListMembers(qp)
var page model.MemberResult
err := xdb.ExecuteQueryWithPagination(ctx, p, &page, q, qp)
// next request: Offset = page.NextOffset while page.HasNextPage
```

The `Where` calls in the SQL function must follow the `Set` calls of the
request, because both orders are the argument order. Page by offset only
over a small or stable set; otherwise page by cursor.

### Cursor (keyset) pagination

An offset page reads and discards every row before it, and shifts when rows
are inserted or deleted meanwhile. A keyset page starts after the sort key
of the last row of the previous page: every page costs one index range
scan, and concurrent writes neither repeat nor skip rows.

Rules:

1. **The sort key is unique and NOT NULL**: end the ORDER BY with the
   primary key as the tie-breaker (`created_at DESC, id DESC`). A nullable
   column needs the same `COALESCE` in the ORDER BY and the predicate.
2. **The predicate compares the whole key**: with every column in one
   direction, a row comparison: `(created_at, id) < (?, ?)` for DESC, `>`
   for ASC. With mixed directions expand it, binding the shared values
   again: `ORDER BY priority ASC, created_at DESC, id DESC` is
   `(priority > ? OR (priority = ? AND (created_at, id) < (?, ?)))` with
   `SetCursor(limit, pos, prio, prio, at, id)`.
3. **An index matches the ORDER BY** after the equality filters:
   `CREATE INDEX ... ON event (org_id, created_at DESC, id DESC)`.
4. **The cursor holds the whole key at full precision**:
   `xdb.NewCursor(last.CreatedAt, last.ID)`. PostgreSQL stores
   microseconds; a cursor built from `CreatedAt.String()` (milliseconds)
   skips the rows in between.
5. **The cursor is opaque and holds no filters**: the API client sends the
   same filters with it. Its key is not signed; it only moves the start of
   the listing, inside the filters the server applies. A cursor that does
   not scan into the key is an invalid argument.

| Step | Code |
| --- | --- |
| API request → request | `req.SetCursor(in.Cursor)`: `xdb.ParseCursor`, then `Scan` into the typed key fields |
| request → QueryParams | filters, then `b.SetCursor(r.Limit, pos, r.AfterCreatedAt, r.AfterID)`; the first page has no key |
| QueryParams → SQL | with a cursor `Where("(created_at, id) < (?, ?)", nil, nil).Limit(nil)`, else `Limit(nil).Offset(nil)` |
| run | `xdb.ExecuteQueryWithCursor(ctx, db, query.EventCursor, &rs, q, qp)` |
| result → API response | `NextCursor: rs.Cursor`, empty on the last page |

```go
// ListEventsRequest lists events newest first: ORDER BY created_at DESC, id DESC.
type ListEventsRequest struct {
	OrgID uint64
	Limit uint32
	// AfterCreatedAt and AfterID are the sort key of the last row of the
	// previous page, zero for the first page
	AfterCreatedAt xdb.Time
	AfterID        xdb.ID
}

// SetCursor decodes the cursor of the previous page; an empty one starts at
// the first page.
func (r *ListEventsRequest) SetCursor(cursor string) error {
	c, err := xdb.ParseCursor(cursor)
	if err != nil || c.IsZero() {
		return err
	}
	return c.Scan(&r.AfterCreatedAt, &r.AfterID)
}

// EventCursor returns the cursor of the page after the row.
func EventCursor(last *model.Event) string {
	// a Time and an ID always encode; NewCursor fails only for a key type
	// whose MarshalText fails
	c, _ := xdb.NewCursor(last.CreatedAt, last.ID)
	return c.String()
}

// QueryParams records the filters, then the cursor key and the limit.
func (r *ListEventsRequest) QueryParams() xdb.QueryParams {
	b := xdb.NewQueryParams("ListEvents")
	if r.OrgID != 0 {
		b.Set(schema.Event.OrgID.Position, r.OrgID)
	}
	if r.AfterID != 0 {
		b.SetCursor(r.Limit, schema.Event.CreatedAt.Position, r.AfterCreatedAt, r.AfterID)
	} else {
		b.SetCursor(r.Limit, schema.Event.CreatedAt.Position) // first page: SetPage(limit, 0)
	}
	return b
}

// ListEvents returns the SQL of ListEventsRequest.
func ListEvents(args ...any) (string, string) {
	p := xdb.GetQueryParams(args...)
	return xsql.Postgres.GetOrCreateQuery(p.Name(), func(name string) xsql.Builder {
		q := schema.EventTableInfo.Select().
			OrderBy(schema.Event.CreatedAt.Name + " DESC, " + schema.Event.ID.Name + " DESC")
		if p.IsSet(schema.Event.OrgID.Position) {
			q.Where(schema.Event.OrgID.Name+" = ?", nil)
		}
		if _, cursor := p.Cursor(); cursor != nil {
			q.Where("(created_at, id) < (?, ?)", nil, nil).Limit(nil)
		} else {
			q.Limit(nil).Offset(nil)
		}
		return q
	})
}
// first page: ... WHERE org_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3     [org, limit, 0]
// next pages: ... WHERE org_id = $1 AND (created_at, id) < ($2, $3) ORDER BY ... LIMIT $4  [org, at, id, limit]
```

```go
// provider
qp := req.QueryParams()
q, _ := query.ListEvents(qp)
var rs model.EventResult
err := xdb.ExecuteQueryWithCursor(ctx, p, query.EventCursor, &rs, q, qp)

// service
req := &query.ListEventsRequest{OrgID: orgID, Limit: in.Limit}
if err := req.SetCursor(in.Cursor); err != nil {
	return nil, invalidArgument("invalid cursor") // the service's own error mapping
}
// ... list, then: NextCursor: rs.Cursor
```

`pos` marks the cursor variant in the query name: use a position no filter
of the request sets. An unfiltered query also passes its limit through
`QueryParams`: the helpers do not take a lone positional argument for the
limit, since it can be a filter value. The first page and an offset page share their SQL.
`query_test.go` (sqlite) and `db_test.go` (`TestPGKeyset`, PostgreSQL) run
this flow over rows 1µs apart. `xdb.EncodeCursor` and `DecodeCursor`, which
take a `values.MapAny`, are deprecated in favor of `NewCursor`/`ParseCursor`.

### Filters without arguments: IS NULL

```go
func (r *ListUsersRequest) QueryParams() xdb.QueryParams {
	b := xdb.NewQueryParams("ListUsers")
	if r.WithoutUUID {
		b.SetNullColums([]string{schema.User.UUID.Name})
	}
	b.SetPage(r.Limit, 0)
	return b
}

func ListUsers(args ...any) (string, string) {
	p := xdb.GetQueryParams(args...)
	return xsql.Postgres.GetOrCreateQuery(p.Name(), func(name string) xsql.Builder {
		q := schema.UserTableInfo.Select().OrderBy(schema.User.ID.Name).Limit(nil).Offset(nil)
		for _, c := range p.GetNullColumns() {
			q.Where(c + " IS NULL")
		}
		return q
	})
}
// ListUsers_x0_n75cd30d251a4c505: ... WHERE uuid IS NULL ORDER BY id LIMIT $1 OFFSET $2
```

`SetEnum` works the same way for a value that picks the SQL (for example a
scope enum that writes `IS NULL` or `IS NOT NULL`), and `SetFlags` for any
other switch.

### Lists of values, joins, locks and deletes

```go
// one array argument: the SQL does not depend on the list length
schema.UserTableInfo.Select().Where(schema.User.ID.Name+" = ANY(?)", nil).OrderBy(schema.User.ID.Name)
xdb.ExecuteQuery(ctx, p, &rs, q, pq.Array(ids))

// join: SelectAliased prefixes the columns, which still scan into model.User
schema.UserTableInfo.SelectAliased("u", nil).
	Join(schema.MemberTableInfo.SchemaName+" m", "m.user_id = u.id").
	Where("m."+schema.Member.OrgID.Name+" = ?", nil)
// SELECT u.id, u.email, ... FROM public.user u JOIN public.orgmember m ON (m.user_id = u.id) WHERE m.org_id = $1

// lock a row inside a transaction
schema.UserTableInfo.Select().Where(schema.User.ID.Name+" = ?", nil).ForUpdate()

// delete
schema.UserTableInfo.DeleteFrom().Where(schema.User.ID.Name+" = ?", nil)
res, err := p.ExecContext(ctx, q, id) // res.RowsAffected()
```

Prefer `= ANY(?)` with `pq.Array` over `In(...)`: `In` renders one
placeholder per value, so every list length is another SQL string.
`SelectAliased(prefix, nullColumns)` writes `NULL` for the listed columns,
for a join that cannot fill them.

## Pitfalls

- **Argument order**: rendered order (SET, WHERE, LIMIT, OFFSET), not
  builder call order; request `Set` calls in the same order as the `Where`
  calls.
- **Builder arguments are dropped** by `GetOrCreateQuery`: `Limit(1)` still
  renders `LIMIT $n`, and the caller binds `1`.
- **NULL on write**: the zero value of `xdb.ID`, `Int64`, `Int32`, `Float`,
  `Bool` (`false`), `NULLString`, `UUID`, `Time`, and an empty `Metadata` or
  `KVSet`, writes NULL. A NOT NULL jsonb column needs `{}` from the caller;
  a nullable boolean cannot store `false` through `xdb.Bool`.
- **NULL on read**: a view column that is NULL for some rows must be
  `COALESCE(...)` in the view or a NULL-aware type in `typesmap.yaml`;
  scanning NULL into `string` fails.
- **ExecuteQuery does not page**: a request with `SetPage` run through
  `ExecuteQuery` is cut at the limit and reports no next page.
- **Column positions above 63** do not fit the `QueryParams` bit mask and
  panic in `Set`; use small constants for such variants.
- **QueryParams are per request**: build one per call; a builder is not
  safe to share between goroutines.
- **Cursor precision**: encode the key with `xdb.NewCursor`, never with
  `Time.String()`, which drops the microseconds PostgreSQL stores.
- **Paged LIMIT in a subquery**: the paging helpers fetch one extra row and
  drop the last row of the result; a LIMIT inside a subquery that the outer
  query reorders would drop another row instead, which no page shows.
