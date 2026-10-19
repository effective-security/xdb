package xdb

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"iter"
	"math"
	"slices"

	"github.com/cockroachdb/errors"
	"github.com/effective-security/x/values"
)

// DefaultPageSize is the default page size
const DefaultPageSize = 500

// RowPointer defines a generic interface to scan a single row
type RowPointer[T any] interface {
	*T
	RowScanner
}

// QueryRow runs a query and returns a single model
func QueryRow[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, query string, args ...any) (TPointer, error) {
	row := sql.QueryRowContext(ctx, query, args...)
	var m TPointer = new(T)
	err := m.ScanRow(row)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return m, nil
}

// QueryRows runs a query and yields its rows one at a time, each scanned into
// a new model, without collecting them in a slice:
//
//	for m, err := range xdb.QueryRows[model.Event](ctx, db, query, args...) {
//		if err != nil {
//			return err
//		}
//		...
//	}
//
// The rows are closed when the loop ends, also on break. A query, scan or
// iteration error is yielded once, with a nil model, and ends the sequence.
func QueryRows[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, query string, args ...any) iter.Seq2[TPointer, error] {
	return func(yield func(TPointer, error) bool) {
		rows, err := sql.QueryContext(ctx, query, args...)
		if err != nil {
			yield(nil, errors.WithStack(err))
			return
		}
		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			var m TPointer = new(T)
			if err := m.ScanRow(rows); err != nil {
				yield(nil, errors.WithStack(err))
				return
			}
			if !yield(m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errors.WithStack(err))
		}
	}
}

// ExecuteListQuery runs a query and returns a list of models
func ExecuteListQuery[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, query string, args ...any) ([]TPointer, error) {
	return listQuery[T, TPointer](ctx, sql, 0, query, args...)
}

// listQuery collects the rows of a query; size is the expected number of
// rows, such as the page limit, or 0 when unknown.
func listQuery[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, size uint32, query string, args ...any) ([]TPointer, error) {
	list := make([]TPointer, 0, min(size, DefaultPageSize+1))
	for m, err := range QueryRows[T, TPointer](ctx, sql, query, args...) {
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

// Result describes the result of a list query
type Result[T any, TPointer RowPointer[T]] interface {
	SetResult(rows []TPointer, hasNextPage bool, nextOffset uint32)
}

// ResultWithCursor describes the result of a list query with a cursor
type ResultWithCursor[T any, TPointer RowPointer[T]] interface {
	SetResultWithCursor(rows []TPointer, hasNextPage bool, cursor func(lastRow TPointer) string)
}

// SetCursor returns the cursor of the page that follows lastRow, usually
// NewCursor with the sort key of the row:
//
//	func(last *model.Event) string {
//		// a Time and an ID always encode
//		c, _ := xdb.NewCursor(last.CreatedAt, last.ID)
//		return c.String()
//	}
type SetCursor[T any, TPointer RowPointer[T]] func(lastRow TPointer) string

// ExecuteQueryWithPagination runs a query and populates the result with a list of models and the next offset,
// if there are more rows to fetch.
// args can be a QueryParams or a list of arguments followed by the limit and offset.
// The query runs with limit+1 bound in place of the limit, and the extra row,
// which is not returned, tells whether a next page exists; the LIMIT must
// therefore apply to the final, ordered result. A zero limit reports no next page.
func ExecuteQueryWithPagination[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, res Result[T, TPointer], query string, args ...any) error {
	limit, offset, args, limitAt := offsetPage(args)

	list, hasNextPage, err := executePage[T, TPointer](ctx, sql, limit, limitAt, query, args)
	if err != nil {
		return err
	}

	nextOffset := values.Select(hasNextPage, offset+uint32(len(list)), 0)
	res.SetResult(list, hasNextPage, nextOffset)

	return nil
}

// ExecuteQueryWithCursor runs a query and populates the result with a list of models and the next cursor,
// if there are more rows to fetch.
// args can be a QueryParams or a list of arguments followed by the limit,
// with at least one argument before it; a query whose only argument is the
// limit passes QueryParams, such as NewQueryParams(name).SetCursor(limit, pos).
// The query runs with limit+1 bound in place of the limit, and the extra row,
// which is not returned, tells whether a next page exists; the LIMIT must
// therefore apply to the final, ordered result. A zero limit reports no next page.
func ExecuteQueryWithCursor[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, cursor SetCursor[T, TPointer], res ResultWithCursor[T, TPointer], query string, args ...any) error {
	limit, args, limitAt := cursorPage(args)

	list, hasNextPage, err := executePage[T, TPointer](ctx, sql, limit, limitAt, query, args)
	if err != nil {
		return err
	}

	res.SetResultWithCursor(list, hasNextPage, cursor)

	return nil
}

// executePage runs a page query and returns at most limit rows, and whether
// more rows follow. It binds limit+1 in place of the limit argument at
// limitAt and drops the extra row; when that argument does not hold the
// limit, it runs the query as is and a full page counts as having more rows.
func executePage[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, limit uint32, limitAt int, query string, args []any) ([]TPointer, bool, error) {
	if limit == 0 {
		list, err := listQuery[T, TPointer](ctx, sql, 0, query, args...)
		return list, false, err
	}

	peek, ok := peekLimit(args, limitAt, limit)
	if !ok {
		list, err := listQuery[T, TPointer](ctx, sql, limit, query, args...)
		if err != nil {
			return nil, false, err
		}
		return list, uint32(len(list)) >= limit, nil
	}

	list, err := listQuery[T, TPointer](ctx, sql, limit+1, query, peek...)
	if err != nil {
		return nil, false, err
	}
	if uint32(len(list)) <= limit {
		return list, false, nil
	}
	// release the extra row
	clear(list[limit:])
	return list[:limit], true, nil
}

// peekLimit returns a copy of args with the limit argument at i increased by
// one, or false when the argument at i does not hold the limit.
func peekLimit(args []any, i int, limit uint32) ([]any, bool) {
	if i < 0 || i >= len(args) || limit == math.MaxUint32 {
		return nil, false
	}
	next, ok := nextLimit(args[i], limit)
	if !ok {
		return nil, false
	}
	peek := slices.Clone(args)
	peek[i] = next
	return peek, true
}

// nextLimit returns limit+1 in the type of the limit argument v, of a type
// PageParam reads.
func nextLimit(v any, limit uint32) (any, bool) {
	switch v := v.(type) {
	case uint32:
		return v + 1, v == limit
	case int:
		// on 32-bit targets an int limit can reach math.MaxInt
		return v + 1, v < math.MaxInt && int64(v) == int64(limit)
	case sql.NamedArg:
		next, ok := nextLimit(v.Value, limit)
		v.Value = next
		return v, ok
	}
	return nil, false
}

// ExecuteQuery runs a query and populates the result with a list of models.
// args can be a QueryParams or a list of arguments
func ExecuteQuery[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, res Result[T, TPointer], query string, args ...any) error {
	if qp, ok := queryParams(args); ok {
		args = qp.Args()
	}

	list, err := ExecuteListQuery[T, TPointer](ctx, sql, query, args...)
	if err != nil {
		return err
	}

	res.SetResult(list, false, 0)
	return nil
}

// queryParams returns the QueryParams passed as the only argument.
func queryParams(args []any) (QueryParams, bool) {
	if len(args) != 1 {
		return nil, false
	}
	qp, ok := args[0].(QueryParams)
	return qp, ok
}

// offsetPage returns the limit, offset and arguments of a query paged by
// offset, and the index of the limit argument: from QueryParams, or the last
// two of the positional arguments.
func offsetPage(args []any) (limit, offset uint32, qargs []any, limitAt int) {
	if qp, ok := queryParams(args); ok {
		limit, offset = qp.Page()
		qargs = qp.Args()
		return limit, offset, qargs, paramsLimitAt(qp, qargs)
	}
	if n := len(args); n >= 2 {
		return PageParam(args[n-2]), PageParam(args[n-1]), args, n - 2
	}
	return 0, 0, args, -1
}

// cursorPage returns the limit and arguments of a query paged by cursor, and
// the index of the limit argument: from QueryParams, or the last of two or
// more positional arguments.
func cursorPage(args []any) (limit uint32, qargs []any, limitAt int) {
	if qp, ok := queryParams(args); ok {
		limit, _ = qp.Cursor()
		qargs = qp.Args()
		return limit, qargs, paramsLimitAt(qp, qargs)
	}
	// a lone argument is not taken as the limit: its type cannot tell a
	// limit from a filter value; a limit-only query passes QueryParams
	if n := len(args); n >= 2 {
		return PageParam(args[n-1]), args, n - 1
	}
	return 0, args, -1
}

// paramsLimitAt returns the index of the limit argument of QueryParams:
// SetCursor ends the arguments with the key values and the limit, SetPage
// with the limit and the offset.
func paramsLimitAt(qp QueryParams, args []any) int {
	if _, cursor := qp.Cursor(); cursor != nil {
		return len(args) - 1
	}
	return len(args) - 2
}

// EncodeCursor encodes the offset or value into a cursor
//
// Deprecated: use NewCursor with the sort key of the last row, which keeps
// the key values' types and precision.
func EncodeCursor(val values.MapAny) string {
	return base64.RawURLEncoding.EncodeToString([]byte(val.JSON()))
}

// DecodeCursor decodes the cursor into a map
//
// Deprecated: use ParseCursor and Cursor.Scan.
func DecodeCursor(cursor string) (values.MapAny, error) {
	js, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode cursor")
	}
	var m values.MapAny
	err = json.Unmarshal(js, &m)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal cursor")
	}
	return m, nil
}
