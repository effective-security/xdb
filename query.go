package xdb

import (
	"context"

	"github.com/pkg/errors"
)

// DefaultPageSize is the default page size
const DefaultPageSize = 500

// RowScanner defines an interface to scan a single row
type RowScanner interface {
	ScanRow(rows Row) error
}

// RowPointer defines a generic interface to scan a single row
type RowPointer[T any] interface {
	*T
	RowScanner
}

// Result describes the result of a list query
type Result[T any, TPointer RowPointer[T]] struct {
	Rows       []TPointer
	NextOffset uint32
}

// RunListQuery runs a query and returns a list of models
func RunListQuery[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, take uint32, query string, args ...any) ([]TPointer, error) {
	rows, err := sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	list := make([]TPointer, 0, take)

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

// RunListQueryWithOffset runs a query and returns a list of models, with the next offset,
// if there are more rows to fetch
func RunListQueryWithOffset[T any, TPointer RowPointer[T]](ctx context.Context, sql DB, offset, take uint32, query string, args ...any) ([]TPointer, uint32, error) {
	list, err := RunListQuery[T, TPointer](ctx, sql, take, query, args...)
	if err != nil {
		return nil, 0, err
	}
	nextOffset := uint32(0)
	count := uint32(len(list))
	if count == take {
		nextOffset = offset + count
	}
	return list, nextOffset, nil
}

// RunQueryResult runs a query and populates the result with a list of models and the next offset,
// if there are more rows to fetch
func (p *Result[T, RowPointer]) RunQueryResult(ctx context.Context, sql DB, offset, take uint32, query string, args ...any) error {
	var err error
	p.Rows, p.NextOffset, err = RunListQueryWithOffset[T, RowPointer](ctx, sql, offset, take, query, args...)
	if err != nil {
		return err
	}
	return nil
}
