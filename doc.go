// Package xdb provides the building blocks of SQL services on database/sql.
//
//   - Provider: a connection opened by NewProvider, which can run the
//     migrations first, with transactions (BeginTx) and snowflake IDs (NextID).
//   - Value types whose zero value is SQL NULL: ID, ID32, Int64, Int32, Float,
//     Bool, NULLString, UUID, Time, Metadata, KVSet and Strings.
//   - QueryParams: the variant name and positional arguments of a query whose
//     SQL is built once with xsql and cached under that name.
//   - QueryRow, QueryRows, ExecuteQuery, ExecuteQueryWithPagination and
//     ExecuteQueryWithCursor: run SQL into the models that xdbcli generates.
//   - Cursor: the opaque token of keyset pagination, built with NewCursor
//     from the sort key of the last row and read back with ParseCursor.
//
// In a service, with the generated model and schema packages and a query
// package of SQL functions:
//
//	p, err := xdb.NewProvider("postgres://u:p@localhost:5432?sslmode=disable", "mydb", nil, nil)
//	if err != nil {
//		return err
//	}
//	defer p.Close()
//
//	q, _ := query.GetUserByID()
//	u, err := xdb.QueryRow[model.User](ctx, p, q, id)
//	if err != nil {
//		return xdb.CheckNotFoundError(err, schema.UserTableInfo.Name, id)
//	}
//
// Documentation/design/QUERYBUILDER.md describes the query patterns and
// Documentation/design/SCHEMA_GENERATOR.md the generator.
package xdb
