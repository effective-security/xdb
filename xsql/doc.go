// Package xsql is a fast SQL statement builder and executor.
//
// # Overview
//
// xsql helps you assemble SQL at runtime from fragments and bound arguments:
//
//   - Combine raw SQL fragments with matching arguments
//   - Map selected columns to variables via To or Bind
//   - Convert ? placeholders to PostgreSQL $1, $2, ... when using Postgres
//   - Execute the statement through any database/sql-compatible Executor
//
// It is not an ORM. Table and column names are not validated, and there is
// no wrapper for OR (prefer UNION, WITH, or a second query).
//
// # Dialects
//
// NoDialect (the default) leaves placeholders as ?.
// Postgres rewrites them to numbered parameters.
// SQLServer leaves placeholders as ? for the Microsoft driver.
//
//	q := xsql.Postgres.From("users").Select("id").Where("email = ?", email)
//	// SELECT id FROM users WHERE email = $1
//
// SetDialect changes the process-wide default used by package-level
// constructors (From, Select, InsertInto, ...). Shared dialect values
// (Postgres, NoDialect, SQLServer) are safe for concurrent constructors.
//
// # Concurrency
//
// A Dialect is safe for concurrent use. A Builder / Stmt is not.
// Create a new statement per goroutine, or Clone a template:
//
//	template := xsql.Postgres.From("users").Select("id, name")
//	defer template.Close()
//
//	q := template.Clone().Where("id = ?", id)
//	defer q.Close()
//
// UseNewLines on the package returns a dialect view and does not mutate the
// default. Prefer WithNewLines on a shared dialect over UseNewLines, which
// writes the receiver.
//
// # Resource reuse
//
// Statements allocate buffers from a pool. Always Close a statement (or use
// QueryAndClose / QueryRowAndClose / ExecAndClose) so those buffers can be
// reused. Do not use a statement after Close.
//
// # Query cache
//
// Rendered SQL is cached per dialect, keyed by the raw statement buffer
// and, when SetName is used, also by that name. The cache is bounded
// (see defaultMaxCachedQueries). Use SetName or GetOrCreateQuery for hot
// paths that rebuild the same SQL.
package xsql
