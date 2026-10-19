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
  schema/       schema.gen.go (TableInfo, column defs), typesmap.yaml, schema.go (init warm-up)
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

`TableInfo.AllColumns` caches its string without a lock. Warm it while the
process is single-threaded, next to the generated file:

```go
// schema/schema.go
func init() {
	for _, t := range TestdbTables {
		_ = t.AllColumns()
	}
}
```

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

Transactions do not nest, and `BeginTx` currently ignores `TxOptions`.
A service provider that wraps `xdb.Provider` returns its own type from
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
| `SetNullColums(cols)` | yes (hash) | no | `IS NULL` filters; the slice is kept and sorted, pass a fresh one |
| `SetPage(limit, offset)` | `_o` when offset > 0 | limit, offset | offset pagination; must be last |
| `SetCursor(limit, pos, v)` | yes, `_c` | `v`, limit | keyset pagination; must be last |

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
| `xdb.ExecuteQuery(ctx, db, &rs, sql, qp)` | all rows, no next page |
| `xdb.ExecuteQueryWithPagination(ctx, db, &rs, sql, qp)` | page, `HasNextPage`, `NextOffset` |
| `xdb.ExecuteQueryWithCursor(ctx, db, encode, &rs, sql, qp)` | page, `HasNextPage`, `Cursor = encode(lastRow)` |
| `db.ExecContext(ctx, sql, args...)` | `sql.Result` for statements without rows |

`db` is any `xdb.DB`: the provider, a transaction provider, `*sql.DB` or
`*sql.Tx`. The `Execute*` helpers take the `QueryParams` itself as the only
argument, or plain arguments (for the paging helpers the last ones are the
limit and offset, or the limit); `QueryRow` and `ExecContext` take
`qp.Args()...`.

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

### List with optional filters and offset or cursor pages

```go
// ListMembersRequest lists memberships, by offset or by an ID cursor.
type ListMembersRequest struct {
	OrgID    uint64
	Role     string
	Limit    uint32
	Offset   uint32
	BeforeID uint64 // cursor: the last ID of the previous page
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
	if r.BeforeID != 0 {
		b.SetCursor(r.Limit, schema.Member.ID.Position, r.BeforeID)
	} else {
		b.SetPage(r.Limit, r.Offset)
	}
	return b
}

// ListMembers returns the SQL of ListMembersRequest, newest first.
func ListMembers(args ...any) (string, string) {
	p := xdb.GetQueryParams(args...)
	return xsql.Postgres.GetOrCreateQuery(p.Name(), func(name string) xsql.Builder {
		q := schema.MemberTableInfo.Select().OrderBy(schema.Member.ID.Name + " DESC")
		if p.IsSet(schema.Member.OrgID.Position) {
			q.Where(schema.Member.OrgID.Name+" = ?", nil)
		}
		if p.IsSet(schema.Member.Role.Position) {
			q.Where(schema.Member.Role.Name+" = ?", nil)
		}
		if _, cursor := p.Cursor(); cursor != nil {
			q.Where(schema.Member.ID.Name+" < ?", nil).Limit(nil)
		} else {
			q.Limit(nil).Offset(nil)
		}
		return q
	})
}
// ListMembers_x4:   ... WHERE org_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3   [org, limit, offset]
// ListMembers_x6_c: ... WHERE org_id = $1 AND id < $2 ORDER BY id DESC LIMIT $3 [org, lastID, limit]
```

```go
qp := req.QueryParams()
q, _ := query.ListMembers(qp)

var page model.MemberResult
err := xdb.ExecuteQueryWithPagination(ctx, p, &page, q, qp)
// next request: Offset = page.NextOffset while page.HasNextPage

encode := func(last *model.Member) string { return last.ID.String() }
err = xdb.ExecuteQueryWithCursor(ctx, p, encode, &page, q, qp)
// next request: the cursor decoded from page.Cursor
```

The `Where` calls in the SQL function must follow the `Set` calls of the
request, because both orders are the argument order. Page by offset only
over a stable set: when the caller deletes or updates the rows it lists,
page by the last ID instead. A full page reports `HasNextPage`, so the page
after an exact multiple of the limit is empty. `xdb.EncodeCursor` and
`DecodeCursor` turn a `values.MapAny` into an opaque string; write an ID
into a cursor as a string, since JSON numbers do not hold 64 bits.

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
- **QueryParams are per request**: `Name()` is computed once, after which
  added parameters do not change it; do not share a builder between
  goroutines or reuse it after `Name()`.
