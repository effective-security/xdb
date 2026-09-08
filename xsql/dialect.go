package xsql

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// defaultMaxCachedQueries limits how many rendered SQL strings a Dialect
// keeps in memory. Named queries (SetName / GetOrCreateQuery) share this
// budget with automatically cached unnamed statements.
const defaultMaxCachedQueries = 4096

// SQLDialect is an interface for SQL statement builders.
//
// A Dialect value published by this package (NoDialect, Postgres, SQLServer)
// is safe for concurrent use: multiple goroutines may call its constructor
// methods at the same time. The Builder instances they return are not.
type SQLDialect interface {
	// Provider returns the name of the SQL dialect.
	Provider() string

	// UseNewLines sets the default newline policy for statements created
	// from this dialect instance. It mutates the receiver; prefer
	// WithNewLines when the dialect is shared across goroutines.
	UseNewLines(op bool)

	// WithNewLines returns a dialect that shares the receiver's query cache
	// but uses op as the newline policy. The original dialect is not mutated.
	WithNewLines(op bool) SQLDialect

	// GetCachedQuery returns a cached query by name.
	GetCachedQuery(name string) (string, bool)

	// PutCachedQuery stores a query in the cache.
	PutCachedQuery(name, query string)

	// GetOrCreateQuery returns a cached query by name or creates a new one.
	// The function closes the Builder returned by create.
	GetOrCreateQuery(name string, create func(name string) Builder) (query string, key string)

	// ClearCache drops all cached SQL strings for this dialect.
	ClearCache()

	// DeleteFrom starts a DELETE statement.
	DeleteFrom(tableName string) Builder

	// From starts a SELECT statement.
	From(expr string, args ...any) Builder

	// InsertInto starts an INSERT statement.
	InsertInto(tableName string) Builder

	/*
		New starts an SQL statement with an arbitrary verb.

		Use From, Select, InsertInto or DeleteFrom methods to create
		an instance of an SQL statement builder for common statements.
	*/
	New(verb string, args ...any) Builder

	/*
		Select starts a SELECT statement.

		Consider using From method to start a SELECT statement - you may find
		it easier to read and maintain.
	*/
	Select(expr string, args ...any) Builder

	// Update starts an UPDATE statement.
	Update(tableName string) Builder

	/*
		With starts a statement prepended by WITH clause
		and closes a subquery passed as an argument.
	*/
	With(queryName string, query Builder) Builder
}

// Dialect defines how an SQL statement is built and caches rendered SQL.
//
// NoDialect is the default statement builder mode: SQL fragments are not
// rewritten. Postgres mode replaces ? placeholders with $1, $2, ...
//
//	q := xsql.Postgres.From("table").Select("field")
//	// ...
//	q.Close()
//
// or as the process default:
//
//	xsql.SetDialect(xsql.Postgres)
//	q := xsql.From("table").Select("field")
//	q.Close()
//
// Shared Dialect values are safe for concurrent constructors. Toggle
// newlines with WithNewLines rather than UseNewLines when the instance
// is used from more than one goroutine.
type Dialect struct {
	provider    string
	cacheMu     *sync.Mutex
	cache       map[string]string
	maxCache    int64
	useNewLines atomic.Bool
}

func newDialect(provider string, useNewLines bool) *Dialect {
	d := &Dialect{
		provider: provider,
		cacheMu:  new(sync.Mutex),
		cache:    make(map[string]string),
		maxCache: defaultMaxCachedQueries,
	}
	d.useNewLines.Store(useNewLines)
	return d
}

var (
	// NoDialect is a default statement builder mode.
	NoDialect = SQLDialect(newDialect("default", true))
	// Postgres mode automatically replaces ? placeholders with $1, $2...
	Postgres = SQLDialect(newDialect("postgres", true))
	// SQLServer is a statement builder for SQL Server. Placeholders stay as ?.
	SQLServer = SQLDialect(newDialect("sqlserver", true))
)

var defaultDialect atomic.Value // SQLDialect

func init() {
	defaultDialect.Store(NoDialect)
}

// SetDialect selects a Dialect to be used by default.
//
// Dialect can be one of xsql.NoDialect, xsql.Postgres or xsql.SQLServer.
//
//	xsql.SetDialect(xsql.Postgres)
//
// SetDialect is safe to call concurrently with statement constructors.
func SetDialect(newDefaultDialect SQLDialect) {
	defaultDialect.Store(newDefaultDialect)
}

// UseNewLines specifies an option to add new lines for each clause
func (b *Dialect) UseNewLines(op bool) {
	b.useNewLines.Store(op)
}

// WithNewLines returns a dialect that shares this instance's query cache
// but uses op as the newline policy for newly created statements.
func (b *Dialect) WithNewLines(op bool) SQLDialect {
	d := &Dialect{
		provider: b.provider,
		cacheMu:  b.cacheMu,
		cache:    b.cache,
		maxCache: b.maxCache,
	}
	d.useNewLines.Store(op)
	return d
}

// Provider returns the name of the SQL dialect.
func (b *Dialect) Provider() string {
	return b.provider
}

/*
New starts an SQL statement with an arbitrary verb.

Use From, Select, InsertInto or DeleteFrom methods to create
an instance of an SQL statement builder for common statements.
*/
func (b *Dialect) New(verb string, args ...any) Builder {
	q := b.getStmt()
	q.addChunk(posSelect, verb, "", args, ", ")
	return q
}

/*
With starts a statement prepended by WITH clause
and closes a subquery passed as an argument.
*/
func (b *Dialect) With(queryName string, query Builder) Builder {
	q := b.getStmt()
	return q.With(queryName, query)
}

/*
From starts a SELECT statement.
*/
func (b *Dialect) From(expr string, args ...any) Builder {
	q := b.getStmt()
	return q.From(expr, args...)
}

/*
Select starts a SELECT statement.

Consider using From method to start a SELECT statement - you may find
it easier to read and maintain.
*/
func (b *Dialect) Select(expr string, args ...any) Builder {
	q := b.getStmt()
	return q.Select(expr, args...)
}

// Update starts an UPDATE statement.
func (b *Dialect) Update(tableName string) Builder {
	q := b.getStmt()
	return q.Update(tableName)
}

// InsertInto starts an INSERT statement.
func (b *Dialect) InsertInto(tableName string) Builder {
	q := b.getStmt()
	return q.InsertInto(tableName)
}

// DeleteFrom starts a DELETE statement.
func (b *Dialect) DeleteFrom(tableName string) Builder {
	q := b.getStmt()
	return q.DeleteFrom(tableName)
}

// writePg copies s into buf and replaces ? placeholders with $1, $2...
// A question mark escaped as \? is written as a literal ?.
func writePg(argNo int, s []byte, buf *strings.Builder) int {
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 < len(s) && s[i+1] == '?' {
				buf.Write(s[start:i])
				buf.WriteByte('?')
				i++
				start = i + 1
			}
		case '?':
			buf.Write(s[start:i])
			buf.WriteByte('$')
			buf.WriteString(strconv.Itoa(argNo))
			argNo++
			start = i + 1
		}
	}
	if start < len(s) {
		buf.Write(s[start:])
	}
	return argNo
}
