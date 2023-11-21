package xsql

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// SQLDialect is an interface for SQL statement builders.
type SQLDialect interface {
	// Provider returns the name of the SQL dialect.
	Provider() string

	ClearCache()
	GetCachedQuery(name string) (string, bool)
	PutCachedQuery(name, query string)

	// DeleteFrom starts a DELETE statement.
	DeleteFrom(tableName string) Builder

	/*
		From starts a SELECT statement.
	*/
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

// Dialect defines the method SQL statement is to be built.
//
// NoDialect is a default statement builder mode.
// No SQL fragments will be altered.
// PostgreSQL mode can be set for a statement:
//
//	q := xsql.PostgreSQL.From("table").Select("field")
//		...
//	q.Close()
//
// or as default mode:
//
//	    xsql.SetDialect(xsql.PostgreSQL)
//		   ...
//	    q := xsql.From("table").Select("field")
//	    q.Close()
//
// When PostgreSQL mode is activated, ? placeholders are
// replaced with numbered positional arguments like $1, $2...
type Dialect struct {
	provider  string
	cacheOnce sync.Once
	cacheLock sync.RWMutex
	cache     sqlCache
}

var (
	// NoDialect is a default statement builder mode.
	NoDialect = SQLDialect(&Dialect{provider: "default"})
	// Postgres mode is to be used to automatically replace ? placeholders with $1, $2...
	Postgres = SQLDialect(&Dialect{provider: "postgres"})

	SQLServer = SQLDialect(&Dialect{provider: "sqlserver"})
)

var defaultDialect atomic.Value // *SQLDialect

func init() {
	// Initialize to a blackhole sink to avoid errors
	defaultDialect.Store(NoDialect)
}

/*
SetDialect selects a Dialect to be used by default.

Dialect can be one of xsql.NoDialect or xsql.PostgreSQL

	xsql.SetDialect(xsql.PostgreSQL)
*/
func SetDialect(newDefaultDialect SQLDialect) {
	defaultDialect.Store(newDefaultDialect)
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
	q := getStmt(b)
	q.addChunk(posSelect, verb, "", args, ", ")
	return q
}

/*
With starts a statement prepended by WITH clause
and closes a subquery passed as an argument.
*/
func (b *Dialect) With(queryName string, query Builder) Builder {
	q := getStmt(b)
	return q.With(queryName, query)
}

/*
From starts a SELECT statement.
*/
func (b *Dialect) From(expr string, args ...any) Builder {
	q := getStmt(b)
	return q.From(expr, args...)
}

/*
Select starts a SELECT statement.

Consider using From method to start a SELECT statement - you may find
it easier to read and maintain.
*/
func (b *Dialect) Select(expr string, args ...any) Builder {
	q := getStmt(b)
	return q.Select(expr, args...)
}

// Update starts an UPDATE statement.
func (b *Dialect) Update(tableName string) Builder {
	q := getStmt(b)
	return q.Update(tableName)
}

// InsertInto starts an INSERT statement.
func (b *Dialect) InsertInto(tableName string) Builder {
	q := getStmt(b)
	return q.InsertInto(tableName)
}

// DeleteFrom starts a DELETE statement.
func (b *Dialect) DeleteFrom(tableName string) Builder {
	q := getStmt(b)
	return q.DeleteFrom(tableName)
}

// writePg function copies s into buf and replaces ? placeholders with $1, $2...
func writePg(argNo int, s []byte, buf *strings.Builder) (int, error) {
	var err error
	start := 0
	// Iterate by runes
	for pos, r := range bufToString(&s) {
		if start > pos {
			continue
		}
		switch r {
		case '\\':
			if pos < len(s)-1 && s[pos+1] == '?' {
				_, err = buf.Write(s[start:pos])
				if err == nil {
					err = buf.WriteByte('?')
				}
				start = pos + 2
			}
		case '?':
			_, err = buf.Write(s[start:pos])
			start = pos + 1
			if err == nil {
				err = buf.WriteByte('$')
				if err == nil {
					buf.WriteString(strconv.Itoa(argNo))
					argNo++
				}
			}
		}
		if err != nil {
			break
		}
	}
	if err == nil && start < len(s) {
		_, err = buf.Write(s[start:])
	}
	return argNo, err
}
