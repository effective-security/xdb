package xdb

import (
	"context"
	"encoding/base64"
	"encoding/json"

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

// ExecuteListQuery runs a query and returns a list of models
func ExecuteListQuery[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, query string, args ...any) ([]TPointer, error) {
	rows, err := sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	list := make([]TPointer, 0, DefaultPageSize)

	for rows.Next() {
		var m TPointer = new(T)
		err = m.ScanRow(rows)
		if err != nil {
			return nil, errors.WithStack(err)
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

type SetCursor[T any, TPointer RowPointer[T]] func(lastRow TPointer) string

// ExecuteQueryWithPagination runs a query and populates the result with a list of models and the next offset,
// if there are more rows to fetch.
// args can be a QueryParams or a list of arguments followed by the limit and offset.
func ExecuteQueryWithPagination[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, res Result[T, TPointer], query string, args ...any) error {
	var (
		limit  uint32
		offset uint32
	)
	if len(args) == 1 {
		if qp, ok := args[0].(QueryParams); ok {
			limit, offset = qp.Page()
			args = qp.Args()
		}
	} else if len(args) >= 2 {
		clen := len(args)
		// Limit and Offset are the last two arguments
		limit = PageParam(args[clen-2])
		offset = PageParam(args[clen-1])
	}

	list, err := ExecuteListQuery[T, TPointer](ctx, sql, query, args...)
	if err != nil {
		return err
	}

	count := uint32(len(list))
	hasNextPage := count >= limit
	nextOffset := values.Select(hasNextPage, offset+count, 0)

	res.SetResult(list, hasNextPage, nextOffset)

	return nil
}

// ExecuteQueryWithCursor runs a query and populates the result with a list of models and the next cursor,
// if there are more rows to fetch.
// args can be a QueryParams or a list of arguments followed by the limit and offset.
func ExecuteQueryWithCursor[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, cursor SetCursor[T, TPointer], res ResultWithCursor[T, TPointer], query string, args ...any) error {
	var (
		limit uint32
	)
	if len(args) == 1 {
		if qp, ok := args[0].(QueryParams); ok {
			limit, _ = qp.Cursor()
			args = qp.Args()
		}
	} else if len(args) >= 2 {
		clen := len(args)
		// Cursor and Limit are the last two arguments
		// cursor = PageParam(args[clen-2])
		limit = PageParam(args[clen-1])
	}

	list, err := ExecuteListQuery[T, TPointer](ctx, sql, query, args...)
	if err != nil {
		return err
	}

	count := uint32(len(list))
	hasNextPage := count >= limit

	res.SetResultWithCursor(list, hasNextPage, cursor)

	return nil
}

// ExecuteQuery runs a query and populates the result with a list of models.
// args can be a QueryParams or a list of arguments
func ExecuteQuery[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, res Result[T, TPointer], query string, args ...any) error {
	if len(args) == 1 {
		if qp, ok := args[0].(QueryParams); ok {
			args = qp.Args()
		}
	}

	list, err := ExecuteListQuery[T, TPointer](ctx, sql, query, args...)
	if err != nil {
		return err
	}

	res.SetResult(list, false, 0)
	return nil
}

// EncodeCursor encodes the offset or value into a cursor
func EncodeCursor(val values.MapAny) string {
	return base64.RawURLEncoding.EncodeToString([]byte(val.JSON()))
}

// DecodeCursor decodes the cursor into a map
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
