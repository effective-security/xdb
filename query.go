package xdb

import (
	"context"

	"github.com/effective-security/porto/x/slices"
	"github.com/pkg/errors"
)

// DefaultPageSize is the default page size
const DefaultPageSize = 500

// RowPointer defines a generic interface to scan a single row
type RowPointer[T any] interface {
	*T
	RowScanner
}

// Result describes the result of a list query
type Result[T any, TPointer RowPointer[T]] struct {
	Rows       []TPointer
	NextOffset uint32
	Limit      uint32
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

// Execute runs a query and populates the result with a list of models and the next offset,
// if there are more rows to fetch
func (p *Result[T, RowPointer]) Execute(ctx context.Context, sql DB, query string, args ...any) error {
	var err error
	limit := slices.NvlNumber(p.Limit, DefaultPageSize)

	list, err := ExecuteListQuery[T, RowPointer](ctx, sql, query, args...)
	if err != nil {
		return err
	}
	p.Rows = list
	count := uint32(len(list))
	if count >= limit {
		p.NextOffset = p.NextOffset + count
	} else {
		p.NextOffset = 0
	}

	return nil
}
