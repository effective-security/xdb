package xsql

import (
	"context"
	"database/sql"
	"reflect"
	"strings"

	"github.com/valyala/bytebufferpool"
)

// Builder is an interface for SQL statement builders.
type Builder interface {
	/*
		Args returns the list of arguments to be passed to
		database driver for statement execution.

		Do not access a slice returned by this method after Builder is closed.

		An array, a returned slice points to, can be altered by any method that
		adds a clause or an expression with arguments.

		Make sure to make a copy of the returned slice if you need to preserve it.
	*/
	Args() []any

	// Bind adds structure fields to SELECT statement.
	// Structure fields have to be annotated with "db" tag.
	// Reflect-based Bind is slightly slower than `Select("field").To(&record.field)`
	// but provides an easier way to retrieve data.
	//
	// Note: this method does no type checks and returns no errors.
	Bind(data any) Builder

	/*
		Clause appends a raw SQL fragment to the statement.

		Use it to add a raw SQL fragment like ON CONFLICT, ON DUPLICATE KEY, WINDOW, etc.

		An SQL fragment added via Clause method appears after the last clause previously
		added. If called first, Clause method prepends a statement with a raw SQL.
	*/
	Clause(expr string, args ...any) Builder

	// Clone creates a copy of the statement.
	// The copy does not inherit the cache name; call SetName if needed.
	Clone() Builder

	/*
		Close puts buffers and other objects allocated to build an SQL statement
		back to pool for reuse by other Builder instances.

		Builder instance should not be used after Close method call.
		A Builder is not safe for concurrent use.
	*/
	Close()

	/*
		DeleteFrom starts a DELETE statement.

			err := xsql.DeleteFrom("table").Where("id = ?", id).ExecAndClose(ctx, db)
	*/
	DeleteFrom(tableName string) Builder

	/*
		Dest returns a list of value pointers passed via To method calls.
		The order matches the constructed SQL statement.

		Do not access a slice returned by this method after Builder is closed.

		Note that an array, a returned slice points to, can be altered by To method
		calls.

		Make sure to make a copy if you need to preserve a slice returned by this method.
	*/
	Dest() []any

	// Exec executes the statement.
	Exec(ctx context.Context, db Executor) (sql.Result, error)

	// ExecAndClose executes the statement and releases all the objects
	// and buffers allocated by statement builder back to a pool.
	//
	// Do not call any Builder methods after this call.
	ExecAndClose(ctx context.Context, db Executor) (sql.Result, error)

	/*
		Expr appends an expression to the most recently added clause.

		Expressions are separated with commas.
	*/
	Expr(expr string, args ...any) Builder

	/*
		From starts a SELECT statement.

			var cnt int64

			err := xsql.From("table").
				Select("COUNT(*)").To(&cnt)
				Where("value >= ?", 42).
				QueryRowAndClose(ctx, db)
			if err != nil {
				panic(err)
			}
	*/
	From(expr string, args ...any) Builder

	/*
		FullJoin adds a FULL OUTER JOIN clause to SELECT statement
	*/
	FullJoin(table string, on string) Builder

	// CrossJoin adds a CROSS JOIN clause to SELECT statement
	CrossJoin(table string) Builder

	/*
		ForUpdate appends FOR UPDATE to a SELECT statement.

		Optional tokens are appended as-is, for example "NOWAIT" or "SKIP LOCKED".
	*/
	ForUpdate(opts ...string) Builder

	/*
		ForShare appends FOR SHARE to a SELECT statement.

		Optional tokens are appended as-is, for example "NOWAIT" or "OF t".
	*/
	ForShare(opts ...string) Builder

	// GroupBy adds the GROUP BY clause to SELECT statement
	GroupBy(expr string) Builder

	// Having adds the HAVING clause to SELECT statement
	Having(expr string, args ...any) Builder

	In(args ...any) Builder
	NotIn(args ...any) Builder
	InsertInto(tableName string) Builder

	/*
		Invalidate forces a rebuild on next query execution.

		Most likely you don't need to call this method directly.
	*/
	Invalidate()
	Join(table string, on string) Builder
	LeftJoin(table string, on string) Builder

	// Limit adds a limit on number of returned rows
	Limit(limit any) Builder

	/*
		NewRow method helps to construct a bulk INSERT statement.

		The following code

				q := stmt.InsertInto("table")
			    for k, v := range entries {
					q.NewRow().
						Set("key", k).
						Set("value", v)
				}

		produces (assuming there were 2 key/value pairs at entries map):

			INSERT INTO table ( key, value ) VALUES ( ?, ? ), ( ?, ? )
	*/
	NewRow() Row

	// Offset adds an offset on number of skipped rows
	Offset(offset any) Builder

	/*
		OnConflict appends an ON CONFLICT clause (PostgreSQL upsert).

		target is appended after ON CONFLICT, for example "(email) DO NOTHING"
		or "(email) DO UPDATE SET name = EXCLUDED.name". An empty target
		produces a bare ON CONFLICT, typically followed by Clause("DO NOTHING").
	*/
	OnConflict(target string, args ...any) Builder

	OrderBy(expr ...string) Builder

	// Paginate provides an easy way to set both offset and limit
	Paginate(page int, pageSize int) Builder

	// Query executes the statement.
	// For every row of a returned dataset it calls a handler function.
	// If scan targets were set via To method calls, Query method
	// executes rows.Scan right before calling a handler function.
	Query(ctx context.Context, db Executor, handler func(rows *sql.Rows)) error

	// QueryAndClose executes the statement and releases all the resources that
	// can be reused to a pool. Do not call any Builder methods after this call.
	// For every row of a returned dataset QueryAndClose executes a handler function.
	// If scan targets were set via To method calls, QueryAndClose method
	// executes rows.Scan right before calling a handler function.
	QueryAndClose(ctx context.Context, db Executor, handler func(rows *sql.Rows)) error

	// QueryRow executes the statement via Executor methods
	// and scans values to variables bound via To method calls.
	QueryRow(ctx context.Context, db Executor) error

	// QueryRowAndClose executes the statement via Executor methods
	// and scans values to variables bound via To method calls.
	// All the objects allocated by query builder are moved to a pool
	// to be reused.
	//
	// Do not call any Builder methods after this call.
	QueryRowAndClose(ctx context.Context, db Executor) error

	// Returning adds a RETURNING clause to a statement
	Returning(expr string) Builder

	/*
		RightJoin adds a RIGHT OUTER JOIN clause to SELECT statement
	*/
	RightJoin(table string, on string) Builder

	/*
		Select starts a SELECT statement.

			var cnt int64

			err := xsql.Select("COUNT(*)").To(&cnt).
				From("table").
				Where("value >= ?", 42).
				QueryRowAndClose(ctx, db)
			if err != nil {
				panic(err)
			}

		Note that From method can also be used to start a SELECT statement.
	*/
	Select(expr string, args ...any) Builder

	/*
		Set method:

		- Adds a column to the list of columns and a value to VALUES clause of INSERT statement,

		A call to Set method generates both the list of columns and
		values to be inserted by INSERT statement:

			q := xsql.InsertInto("table").Set("field", 42)

		produces

			INSERT INTO table (field) VALUES (42)

		Do not use it to construct ON CONFLICT DO UPDATE SET or similar clauses.
		Use generic Clause and Expr methods instead:

			q.Clause("ON CONFLICT DO UPDATE SET").Expr("column_name = ?", value)
	*/
	Set(field string, value any) Builder

	/*
		SetExpr is an extended version of Set method.

			q.SetExpr("field", "field + 1")
			q.SetExpr("field", "? + ?", 31, 11)
	*/
	SetExpr(field string, expr string, args ...any) Builder

	// String method builds and returns an SQL statement.
	String() string

	/*
		SubQuery appends a sub query expression to a current clause.

		SubQuery method call closes the Builder passed as query parameter.
		Do not reuse it afterwards.
	*/
	SubQuery(prefix string, suffix string, query Builder) Builder

	To(dest ...any) Builder

	/*
		Union adds a UNION clause to the statement.

		all argument controls if UNION ALL or UNION clause
		is to be constructed. Use UNION ALL if possible to
		get faster queries.
	*/
	Union(all bool, query Builder) Builder

	/*
		Update starts an UPDATE statement.

			err := xsql.Update("table").
				Set("field1", "newvalue").
				Where("id = ?", 42).
				ExecAndClose(ctx, db)
			if err != nil {
				panic(err)
			}
	*/
	Update(tableName string) Builder

	/*
		Where adds a filter:

			xsql.From("users").
				Select("id, name").
				Where("email = ?", email).
				Where("is_active = 1")
	*/
	Where(expr string, args ...any) Builder

	// With prepends a statement with an WITH clause.
	// With method calls a Close method of a given query, so
	// make sure not to reuse it afterwards.
	With(queryName string, query Builder) Builder

	// Name returns the name of the statement
	Name() string

	// SetName sets the name of the statement to be cached
	SetName(name string) Builder

	// UseNewLines specifies an option to add new lines for each clause
	UseNewLines(op bool) Builder
}

// Row is an interface for a single row of data.
type Row interface {
	/*
		Set method:

		- Adds a column to the list of columns and a value to VALUES clause of INSERT statement,

		- Adds an item to SET clause of an UPDATE statement.

			q.Set("field", 32)

		For INSERT statements a call to Set method generates
		both the list of columns and values to be inserted:

			q := xsql.InsertInto("table").Set("field", 42)

		produces

			INSERT INTO table (field) VALUES (42)

		Do not use it to construct ON CONFLICT DO UPDATE SET or similar clauses.
		Use generic Clause and Expr methods instead:

			q.Clause("ON CONFLICT DO UPDATE SET").Expr("column_name = ?", value)
	*/
	Set(field string, value any) Row
	/*
		SetExpr is an extended version of Set method.

			q.SetExpr("field", "field + 1")
			q.SetExpr("field", "? + ?", 31, 11)
	*/
	SetExpr(field string, expr string, args ...any) Row
}

/*
New initializes a SQL statement builder instance with an arbitrary verb.

Use xsql.Select(), xsql.InsertInto(), xsql.DeleteFrom() to start
common SQL statements.

Use New for special cases like this:

	q := xsql.New("TRUNCATE")
	for _, table := range tableNames {
		q.Expr(table)
	}
	q.Clause("RESTART IDENTITY")
	err := q.ExecAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
*/
func New(verb string, args ...any) Builder {
	return defaultDialect.Load().(SQLDialect).New(verb, args...)
}

// UseNewLines returns the default dialect with the given newline policy.
// The process-wide default dialect is not mutated; statements created from
// the returned value use op. Call this before adding clauses:
//
//	q := xsql.UseNewLines(false).Select("id").From("table")
func UseNewLines(op bool) SQLDialect {
	return defaultDialect.Load().(SQLDialect).WithNewLines(op)
}

/*
From starts a SELECT statement.

	var cnt int64

	err := xsql.From("table").
		Select("COUNT(*)").To(&cnt)
		Where("value >= ?", 42).
		QueryRowAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
*/
func From(expr string, args ...any) Builder {
	return defaultDialect.Load().(SQLDialect).From(expr, args...)
}

/*
With starts a statement prepended by WITH clause
and closes a subquery passed as an argument.
*/
func With(queryName string, query Builder) Builder {
	return defaultDialect.Load().(SQLDialect).With(queryName, query)
}

/*
Select starts a SELECT statement.

	var cnt int64

	err := xsql.Select("COUNT(*)").To(&cnt).
		From("table").
		Where("value >= ?", 42).
		QueryRowAndClose(ctx, db)
	if err != nil {
		panic(err)
	}

Note that From method can also be used to start a SELECT statement.
*/
func Select(expr string, args ...any) Builder {
	return defaultDialect.Load().(SQLDialect).Select(expr, args...)
}

/*
Update starts an UPDATE statement.

	err := xsql.Update("table").
		Set("field1", "newvalue").
		Where("id = ?", 42).
		ExecAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
*/
func Update(tableName string) Builder {
	return defaultDialect.Load().(SQLDialect).Update(tableName)
}

/*
InsertInto starts an INSERT statement.

	var newId int64
	err := xsql.InsertInto("table").
		Set("field", value).
		Returning("id").To(&newId).
		QueryRowAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
*/
func InsertInto(tableName string) Builder {
	return defaultDialect.Load().(SQLDialect).InsertInto(tableName)
}

/*
DeleteFrom starts a DELETE statement.

	err := xsql.DeleteFrom("table").Where("id = ?", id).ExecAndClose(ctx, db)
*/
func DeleteFrom(tableName string) Builder {
	return defaultDialect.Load().(SQLDialect).DeleteFrom(tableName)
}

type stmtChunk struct {
	pos     chunkPos
	bufLow  int
	bufHigh int
	hasExpr bool
	argLen  int
}
type stmtChunks []stmtChunk

/*
Stmt provides a set of helper methods for SQL statement building and execution.

A Stmt is not safe for concurrent use. Share a Dialect across goroutines
and give each goroutine its own Stmt (or Clone a template per request).

Use one of the following methods to create a SQL statement builder instance:

	xsql.From("table")
	xsql.Select("field")
	xsql.InsertInto("table")
	xsql.Update("table")
	xsql.DeleteFrom("table")

For other SQL statements use New:

	q := xsql.New("TRUNCATE")
	for _, table := range tablesToBeEmptied {
		q.Expr(table)
	}
	err := q.ExecAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
*/
type Stmt struct {
	name        string
	dialect     SQLDialect
	pos         chunkPos
	chunks      stmtChunks
	buf         *bytebufferpool.ByteBuffer
	sql         string
	args        []any
	dest        []any
	useNewLines bool
}

// UseNewLines specifies an option to add new lines for each clause.
// It only affects clauses added after this call. Set it on the dialect
// (WithNewLines) before constructing the statement when possible.
func (q *Stmt) UseNewLines(op bool) Builder {
	q.useNewLines = op
	return q
}

// Name returns the name of the statement
func (q *Stmt) Name() string {
	return q.name
}

// SetName sets the name of the statement
func (q *Stmt) SetName(name string) Builder {
	q.name = name
	return q
}

type newRow struct {
	*Stmt
	first    bool
	notEmpty bool
}

// WriteString appends a string to the statement
func (q *Stmt) WriteString(s string) {
	_, err := q.buf.WriteString(s)
	if err != nil {
		panic(err)
	}
}

/*
Select adds a SELECT clause to a statement and/or appends
an expression that defines columns of a resulting data set.

	q := xsql.Select("DISTINCT field1, field2").From("table")

Select can be called multiple times to add more columns:

	q := xsql.From("table").Select("field1")
	if needField2 {
		q.Select("field2")
	}
	// ...
	q.Close()

Use To method to bind variables to selected columns:

	var (
		num  int
		name string
	)

	res := xsql.From("table").
		Select("num, name").To(&num, &name).
		Where("id = ?", 42).
		QueryRowAndClose(ctx, db)
	if err != nil {
		panic(err)
	}

Note that a SELECT statement can also be started by a From method call.
*/
func (q *Stmt) Select(expr string, args ...any) Builder {
	q.addChunk(posSelect, "SELECT", expr, args, ", ")
	return q
}

// Returning adds a RETURNING clause to a statement
func (q *Stmt) Returning(expr string) Builder {
	q.addChunk(posReturning, "RETURNING", expr, nil, ", ")
	return q
}

/*
To sets a scan target for columns to be selected.

Accepts value pointers to be passed to sql.Rows.Scan by
Query and QueryRow methods.

	var (
		field1 int
		field2 string
	)
	q := xsql.From("table").
		Select("field1").To(&field1).
		Select("field2").To(&field2)
	err := QueryRow(nil, db)
	q.Close()
	if err != nil {
		// ...
	}

To method MUST be called immediately after Select, Returning or other
method that defines data to be returned.
*/
func (q *Stmt) To(dest ...any) Builder {
	if len(dest) > 0 {
		// As Scan bindings make sense for a single clause per statement,
		// the order expressions appear in SQL matches the order expressions
		// are added. So dest value pointers can safely be appended
		// to the list on every To call.
		q.dest = insertAt(q.dest, dest, len(q.dest))
	}
	return q
}

/*
Update adds UPDATE clause to a statement.

	q.Update("table")

tableName argument can be a SQL fragment:

	q.Update("ONLY table AS t")
*/
func (q *Stmt) Update(tableName string) Builder {
	q.addChunk(posUpdate, "UPDATE", tableName, nil, ", ")
	return q
}

/*
InsertInto adds INSERT INTO clause to a statement.

	q.InsertInto("table")

tableName argument can be a SQL fragment:

	q.InsertInto("table AS t")
*/
func (q *Stmt) InsertInto(tableName string) Builder {
	q.addChunk(posInsert, "INSERT INTO", tableName, nil, ", ")
	q.addChunk(posInsertFields-1, "(", "", nil, "")
	q.addChunk(posValues-1, ") VALUES (", "", nil, "")
	q.addChunk(posValues+1, ")", "", nil, "")
	q.pos = posInsertFields
	return q
}

/*
DeleteFrom adds DELETE clause to a statement.

	q.DeleteFrom("table").Where("id = ?", id)
*/
func (q *Stmt) DeleteFrom(tableName string) Builder {
	q.addChunk(posDelete, "DELETE FROM", tableName, nil, ", ")
	return q
}

/*
Set method:

- Adds a column to the list of columns and a value to VALUES clause of INSERT statement,

- Adds an item to SET clause of an UPDATE statement.

	q.Set("field", 32)

For INSERT statements a call to Set method generates
both the list of columns and values to be inserted:

	q := xsql.InsertInto("table").Set("field", 42)

produces

	INSERT INTO table (field) VALUES (42)

Do not use it to construct ON CONFLICT DO UPDATE SET or similar clauses.
Use generic Clause and Expr methods instead:

	q.Clause("ON CONFLICT DO UPDATE SET").Expr("column_name = ?", value)
*/
func (q *Stmt) Set(field string, value any) Builder {
	return q.SetExpr(field, "?", value)
}

/*
SetExpr is an extended version of Set method.

	q.SetExpr("field", "field + 1")
	q.SetExpr("field", "? + ?", 31, 11)
*/
func (q *Stmt) SetExpr(field, expr string, args ...any) Builder {
	p := chunkPos(0)
	for _, chunk := range q.chunks {
		if chunk.pos == posInsert || chunk.pos == posUpdate {
			p = chunk.pos
			break
		}
	}

	switch p {
	case posInsert:
		q.addChunk(posInsertFields, "", field, nil, ", ")
		q.addChunk(posValues, "", expr, args, ", ")
	case posUpdate:
		q.addChunk(posSet, "SET", field+"="+expr, args, ", ")
	}
	return q
}

// From adds a FROM clause to statement.
func (q *Stmt) From(expr string, args ...any) Builder {
	q.addChunk(posFrom, "FROM", expr, args, ", ")
	return q
}

/*
Where adds a filter:

	xsql.From("users").
		Select("id, name").
		Where("email = ?", email).
		Where("is_active = 1")
*/
func (q *Stmt) Where(expr string, args ...any) Builder {
	q.addChunk(posWhere, "WHERE", expr, args, " AND ")
	return q
}

/*
In adds IN expression to the current filter.

In method must be called after a Where method call.

An empty argument list is rendered as `IN (SELECT NULL WHERE 1=0)`
so the filter matches no rows, stays valid SQL (engines reject
IN ()), and does not change the meaning of neighboring AND/OR terms.
*/
func (q *Stmt) In(args ...any) Builder {
	return q.inList("IN", emptyInExpr, args)
}

/*
NotIn adds NOT IN expression to the current filter.

NotIn method must be called after a Where method call.

An empty argument list is rendered as `NOT IN (SELECT NULL WHERE 1=0)`
so the filter matches all values of that column, stays valid SQL
(engines reject NOT IN ()), and does not introduce an ungrouped OR.
*/
func (q *Stmt) NotIn(args ...any) Builder {
	return q.inList("NOT IN", emptyNotInExpr, args)
}

func (q *Stmt) inList(op, emptyExpr string, args []any) Builder {
	if len(args) == 0 {
		q.addChunk(posWhere, "", emptyExpr, nil, " ")
		return q
	}
	buf := getBuffer()
	_, _ = buf.WriteString(op)
	_, _ = buf.WriteString(" (")
	l := len(args) - 1
	for i := range args {
		if i < l {
			_, _ = buf.Write(placeholderComma)
		} else {
			_, _ = buf.Write(placeholder)
		}
	}
	_, _ = buf.WriteString(")")
	chunkStr := bufToString(buf)
	q.addChunk(posWhere, "", chunkStr, args, " ")

	putBuffer(buf)
	return q
}

/*
Join adds an INNER JOIN clause to SELECT statement
*/
func (q *Stmt) Join(table, on string) Builder {
	q.join("JOIN ", table, on)
	return q
}

/*
LeftJoin adds a LEFT OUTER JOIN clause to SELECT statement
*/
func (q *Stmt) LeftJoin(table, on string) Builder {
	q.join("LEFT JOIN ", table, on)
	return q
}

/*
RightJoin adds a RIGHT OUTER JOIN clause to SELECT statement
*/
func (q *Stmt) RightJoin(table, on string) Builder {
	q.join("RIGHT JOIN ", table, on)
	return q
}

/*
FullJoin adds a FULL OUTER JOIN clause to SELECT statement
*/
func (q *Stmt) FullJoin(table, on string) Builder {
	q.join("FULL JOIN ", table, on)
	return q
}

// CrossJoin adds a CROSS JOIN clause to SELECT statement
func (q *Stmt) CrossJoin(table string) Builder {
	q.addChunk(posFrom, "", "CROSS JOIN "+table, nil, " ")
	return q
}

/*
ForUpdate appends FOR UPDATE to a SELECT statement.

Optional tokens are appended as-is:

	q.ForUpdate("NOWAIT")
	q.ForUpdate("OF t", "SKIP LOCKED")
*/
func (q *Stmt) ForUpdate(opts ...string) Builder {
	return q.lockClause("FOR UPDATE", opts)
}

/*
ForShare appends FOR SHARE to a SELECT statement.

Optional tokens are appended as-is:

	q.ForShare("NOWAIT")
*/
func (q *Stmt) ForShare(opts ...string) Builder {
	return q.lockClause("FOR SHARE", opts)
}

func (q *Stmt) lockClause(lock string, opts []string) Builder {
	if len(opts) > 0 {
		lock = lock + " " + strings.Join(opts, " ")
	}
	q.addChunk(posLock, lock, "", nil, " ")
	return q
}

/*
OnConflict appends an ON CONFLICT clause (PostgreSQL upsert).

	q.InsertInto("users").
		Set("email", email).
		Set("name", name).
		OnConflict("(email) DO UPDATE SET name = EXCLUDED.name")

	q.InsertInto("users").
		Set("email", email).
		OnConflict("DO NOTHING")
*/
func (q *Stmt) OnConflict(target string, args ...any) Builder {
	clause := "ON CONFLICT"
	if target != "" {
		clause = "ON CONFLICT " + target
	}
	q.addChunk(posOnConflict, clause, "", args, " ")
	return q
}

// OrderBy adds the ORDER BY clause to SELECT statement
func (q *Stmt) OrderBy(expr ...string) Builder {
	q.addChunk(posOrderBy, "ORDER BY", strings.Join(expr, ", "), nil, ", ")
	return q
}

// GroupBy adds the GROUP BY clause to SELECT statement
func (q *Stmt) GroupBy(expr string) Builder {
	q.addChunk(posGroupBy, "GROUP BY", expr, nil, ", ")
	return q
}

// Having adds the HAVING clause to SELECT statement
func (q *Stmt) Having(expr string, args ...any) Builder {
	q.addChunk(posHaving, "HAVING", expr, args, " AND ")
	return q
}

// Limit adds a limit on number of returned rows
func (q *Stmt) Limit(limit any) Builder {
	q.addChunk(posLimit, "LIMIT ?", "", []any{limit}, "")
	return q
}

// Offset adds an offset on number of skipped rows
func (q *Stmt) Offset(offset any) Builder {
	q.addChunk(posOffset, "OFFSET ?", "", []any{offset}, "")
	return q
}

// Paginate provides an easy way to set both offset and limit
func (q *Stmt) Paginate(page, pageSize int) Builder {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if page > 1 {
		q.Offset((page - 1) * pageSize)
	}
	q.Limit(pageSize)
	return q
}

// With prepends a statement with an WITH clause.
// With method calls a Close method of a given query, so
// make sure not to reuse it afterwards.
func (q *Stmt) With(queryName string, query Builder) Builder {
	q.addChunk(posWith, "WITH", "", nil, "")
	return q.SubQuery(queryName+" AS (", ")", query)
}

/*
Expr appends an expression to the most recently added clause.

Expressions are separated with commas.
*/
func (q *Stmt) Expr(expr string, args ...any) Builder {
	q.addChunk(q.pos, "", expr, args, ", ")
	return q
}

/*
SubQuery appends a sub query expression to a current clause.

SubQuery method call closes the Builder passed as query parameter.
Do not reuse it afterwards.
*/
func (q *Stmt) SubQuery(prefix, suffix string, b Builder) Builder {
	query := b.(*Stmt)
	delimiter := ", "
	if q.pos == posWhere {
		delimiter = " AND "
	}
	index := q.addChunk(q.pos, "", prefix, query.args, delimiter)
	chunk := &q.chunks[index]
	// Make sure subquery is not dialect-specific.
	if query.dialect != NoDialect {
		query.dialect = NoDialect
		query.Invalidate()
	}
	q.WriteString(query.String())
	q.WriteString(suffix)
	chunk.bufHigh = q.buf.Len()
	// Close the subquery
	query.Close()

	return q
}

/*
Union adds a UNION clause to the statement.

all argument controls if UNION ALL or UNION clause
is to be constructed. Use UNION ALL if possible to
get faster queries.
*/
func (q *Stmt) Union(all bool, b Builder) Builder {
	query := b.(*Stmt)
	p := posUnion
	if len(q.chunks) > 0 {
		last := (&q.chunks[len(q.chunks)-1]).pos
		if last >= p {
			p = last + 1
		}
	}
	var index int
	if all {
		index = q.addChunk(p, "UNION ALL ", "", query.args, "")
	} else {
		index = q.addChunk(p, "UNION ", "", query.args, "")
	}
	chunk := &q.chunks[index]
	// Make sure subquery is not dialect-specific.
	if query.dialect != NoDialect {
		query.dialect = NoDialect
		query.Invalidate()
	}
	q.WriteString(query.String())
	chunk.bufHigh = q.buf.Len()
	// Close the subquery
	query.Close()

	return q
}

/*
Clause appends a raw SQL fragment to the statement.

Use it to add a raw SQL fragment like ON CONFLICT, ON DUPLICATE KEY, WINDOW, etc.

An SQL fragment added via Clause method appears after the last clause previously
added. If called first, Clause method prepends a statement with a raw SQL.
*/
func (q *Stmt) Clause(expr string, args ...any) Builder {
	p := posStart
	if len(q.chunks) > 0 {
		p = (&q.chunks[len(q.chunks)-1]).pos + 10
	}
	q.addChunk(p, expr, "", args, ", ")
	return q
}

// String method builds and returns an SQL statement.
func (q *Stmt) String() string {
	if q.sql == "" {
		// Key the rebuild cache by the raw buffer so a later clause on a
		// named statement cannot return a stale cached string.
		// NOTE: can't use bufToString here as it returns a raw pointer.
		bufKey := q.buf.String()
		if sql, ok := q.dialect.GetCachedQuery(bufKey); ok {
			q.sql = sql
		} else {
			var argNo = 1
			buf := strings.Builder{}

			pos := chunkPos(0)
			for n, chunk := range q.chunks {
				// Separate clauses with spaces
				if n > 0 && chunk.pos > pos {
					buf.Write(space)
				}
				s := q.buf.B[chunk.bufLow:chunk.bufHigh]
				if q.dialect.Provider() == "postgres" {
					argNo = writePg(argNo, s, &buf)
				} else {
					buf.Write(s)
				}
				pos = chunk.pos
			}
			q.sql = strings.Trim(buf.String(), "\n\r\t ")
			q.dialect.PutCachedQuery(bufKey, q.sql)
		}
		if q.name != "" {
			q.dialect.PutCachedQuery(q.name, q.sql)
		}
	}
	return q.sql
}

/*
Args returns the list of arguments to be passed to
database driver for statement execution.

Do not access a slice returned by this method after Stmt is closed.

An array, a returned slice points to, can be altered by any method that
adds a clause or an expression with arguments.

Make sure to make a copy of the returned slice if you need to preserve it.
*/
func (q *Stmt) Args() []any {
	return q.args
}

/*
Dest returns a list of value pointers passed via To method calls.
The order matches the constructed SQL statement.

Do not access a slice returned by this method after Stmt is closed.

Note that an array, a returned slice points to, can be altered by To method
calls.

Make sure to make a copy if you need to preserve a slice returned by this method.
*/
func (q *Stmt) Dest() []any {
	return q.dest
}

/*
Invalidate forces a rebuild on next query execution.

Most likely you don't need to call this method directly.
*/
func (q *Stmt) Invalidate() {
	if q.sql != "" {
		q.sql = ""
	}
}

/*
Close puts buffers and other objects allocated to build an SQL statement
back to pool for reuse by other Stmt instances.

Stmt instance should not be used after Close method call.
*/
func (q *Stmt) Close() {
	reuseStmt(q)
}

// Clone creates a copy of the statement.
// The copy is independent: later clauses added to either instance
// do not affect the other. Newline policy, rendered SQL, and scan
// targets are copied. The cache name is not: an extended clone must
// not overwrite the source statement's named-cache entry. Call SetName
// on the clone if it should be cached under its own key.
func (q *Stmt) Clone() Builder {
	stmt := q.dialect.(*Dialect).getStmt()
	if cap(stmt.chunks) < len(q.chunks) {
		stmt.chunks = make(stmtChunks, len(q.chunks), len(q.chunks)+2)
		copy(stmt.chunks, q.chunks)
	} else {
		stmt.chunks = append(stmt.chunks, q.chunks...)
	}
	stmt.args = insertAt(stmt.args, q.args, 0)
	stmt.dest = insertAt(stmt.dest, q.dest, 0)
	_, _ = stmt.buf.Write(q.buf.B)
	stmt.sql = q.sql
	stmt.useNewLines = q.useNewLines
	stmt.pos = q.pos

	return stmt
}

// Bind adds structure fields to SELECT statement.
// Structure fields have to be annotated with "db" tag.
// Reflect-based Bind is slightly slower than `Select("field").To(&record.field)`
// but provides an easier way to retrieve data.
//
// Fields tagged `db:"-"` are skipped, including anonymous embedded
// structs. Column names may be followed by comma-separated options
// (`db:"id,int8,primary"`); only the first token is used as the column name.
//
// Field mappings are cached per struct type.
//
// Note: this method does no type checks and returns no errors.
func (q *Stmt) Bind(data any) Builder {
	val := reflect.ValueOf(data).Elem()
	for _, f := range bindFields(val.Type()) {
		q.Select(f.name).To(val.FieldByIndex(f.index).Addr().Interface())
	}
	return q
}

// join adds a join clause to a SELECT statement
func (q *Stmt) join(joinType, table, on string) (index int) {
	buf := getBuffer()
	_, _ = buf.WriteString(joinType)
	_, _ = buf.WriteString(table)
	_, _ = buf.Write(joinOn)
	_, _ = buf.WriteString(on)
	_ = buf.WriteByte(')')

	chunkStr := bufToString(buf)
	index = q.addChunk(posFrom, "", chunkStr, nil, " ")

	putBuffer(buf)

	return index
}

// addChunk adds a clause or expression to a statement.
func (q *Stmt) addChunk(pos chunkPos, clause, expr string, args []any, sep string) (index int) {
	// Remember the position
	q.pos = pos

	argLen := len(args)
	bufLow := len(q.buf.B)
	index = len(q.chunks)
	argTail := 0

	addNew := true
	addClause := clause != ""

	// Find the position to insert a chunk to
loop:
	for i := index - 1; i >= 0; i-- {
		chunk := &q.chunks[i]
		index = i
		switch {
		// See if an existing chunk can be extended
		case chunk.pos == pos:
			// Do nothing if a clause is already there and no expressions are to be added
			if expr == "" {
				// See if arguments are to be updated
				if argLen > 0 {
					copy(q.args[len(q.args)-argTail-chunk.argLen:], args)
				}
				return i
			}
			// Write a separator
			if chunk.hasExpr {
				q.WriteString(sep)
			} else {
				q.WriteString(" ")
			}
			if chunk.bufHigh == bufLow {
				// Do not add a chunk
				addNew = false
				// Update the existing one
				q.WriteString(expr)
				chunk.argLen += argLen
				chunk.bufHigh = len(q.buf.B)
				chunk.hasExpr = true
			} else {
				// Do not add a clause
				addClause = false
				index = i + 1
			}
			break loop
		// No existing chunks of this type
		case chunk.pos < pos:
			index = i + 1
			break loop
		default:
			argTail += chunk.argLen
		}
	}

	if addNew {
		// Insert a new chunk
		if addClause {
			if q.useNewLines {
				q.WriteString("\n")
			}
			q.WriteString(clause)
			if expr != "" {
				q.WriteString(" ")
			}
		}
		q.WriteString(expr)

		if cap(q.chunks) == len(q.chunks) {
			chunks := make(stmtChunks, len(q.chunks), cap(q.chunks)*2)
			copy(chunks, q.chunks)
			q.chunks = chunks
		}

		chunk := stmtChunk{
			pos:     pos,
			bufLow:  bufLow,
			bufHigh: len(q.buf.B),
			argLen:  argLen,
			hasExpr: expr != "",
		}

		q.chunks = append(q.chunks, chunk)
		if index < len(q.chunks)-1 {
			copy(q.chunks[index+1:], q.chunks[index:])
			q.chunks[index] = chunk
		}
	}

	// Insert query arguments
	if argLen > 0 {
		q.args = insertAt(q.args, args, len(q.args)-argTail)
	}
	q.Invalidate()

	return index
}

/*
NewRow method helps to construct a bulk INSERT statement.

The following code

		q := stmt.InsertInto("table")
	    for k, v := range entries {
			q.NewRow().
				Set("key", k).
				Set("value", v)
		}

produces (assuming there were 2 key/value pairs at entries map):

	INSERT INTO table ( key, value ) VALUES ( ?, ? ), ( ?, ? )
*/
func (q *Stmt) NewRow() Row {
	first := true
	// Check if there are values
loop:
	for i := len(q.chunks) - 1; i >= 0; i-- {
		chunk := q.chunks[i]
		switch {
		// See if an existing chunk can be extended
		case chunk.pos == posValues:
			// Values section is there, prepend
			first = false
			break loop
		case chunk.pos < posValues:
			break loop
		}
	}
	if !first {
		q.addChunk(posValues, "", " ", nil, " ), (")
	}
	return newRow{
		Stmt:  q,
		first: first,
	}
}

/*
Set method:

- Adds a column to the list of columns and a value to VALUES clause of INSERT statement,

A call to Set method generates both the list of columns and
values to be inserted by INSERT statement:

	q := xsql.InsertInto("table").Set("field", 42)

produces

	INSERT INTO table (field) VALUES (42)

Do not use it to construct ON CONFLICT DO UPDATE SET or similar clauses.
Use generic Clause and Expr methods instead:

	q.Clause("ON CONFLICT DO UPDATE SET").Expr("column_name = ?", value)
*/
func (row newRow) Set(field string, value any) Row {
	return row.SetExpr(field, "?", value)
}

/*
SetExpr is an extended version of Set method.

	q.SetExpr("field", "field + 1")
	q.SetExpr("field", "? + ?", 31, 11)
*/
func (row newRow) SetExpr(field, expr string, args ...any) Row {
	q := row.Stmt

	if row.first {
		q.addChunk(posInsertFields, "", field, nil, ", ")
		q.addChunk(posValues, "", expr, args, ", ")
	} else {
		sep := ""
		if row.notEmpty {
			sep = ", "
		}
		q.addChunk(posValues, "", expr, args, sep)
	}

	return newRow{
		Stmt:     row.Stmt,
		first:    row.first,
		notEmpty: true,
	}
}

var (
	space            = []byte{' '}
	placeholder      = []byte{'?'}
	placeholderComma = []byte{'?', ','}
	joinOn           = []byte{' ', 'O', 'N', ' ', '('}

	// emptyInExpr / emptyNotInExpr complete `Where("col").In()` / `NotIn()`
	// when the value list is empty. A subquery membership test is always
	// valid SQL and does not change AND/OR binding of neighboring filters.
	emptyInExpr    = "IN (SELECT NULL WHERE 1=0)"
	emptyNotInExpr = "NOT IN (SELECT NULL WHERE 1=0)"
)

type chunkPos int

const (
	_        chunkPos = iota
	posStart chunkPos = 100 * iota
	posWith
	posInsert
	posInsertFields
	posValues
	posOnConflict
	posDelete
	posUpdate
	posSet
	posSelect
	posInto
	posFrom
	posWhere
	posGroupBy
	posHaving
	posUnion
	posOrderBy
	posLimit
	posOffset
	posLock
	posReturning
	posEnd
)
