package xdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/effective-security/xdb/pkg/flake"
	"github.com/effective-security/xlog"
	"github.com/pkg/errors"
)

var logger = xlog.NewPackageLogger("github.com/effective-security/xdb", "xdb")

// SQLProvider represents SQL client instance
type SQLProvider struct {
	name   string
	conn   *sql.DB
	db     DB
	idGen  flake.IDGenerator
	tx     Tx
	ticker *time.Ticker
}

// New creates a Provider instance
func New(name string, db *sql.DB, idGen flake.IDGenerator) (*SQLProvider, error) {
	if idGen == nil {
		idGen = flake.DefaultIDGenerator
	}
	p := &SQLProvider{
		name:  name,
		conn:  db,
		db:    db,
		idGen: idGen,
	}

	p.keepAlive(60 * time.Second)

	return p, nil
}

// Name returns provider name
func (p *SQLProvider) Name() string {
	return p.name
}

func (p *SQLProvider) keepAlive(period time.Duration) {
	p.ticker = time.NewTicker(period)
	ch := p.ticker.C

	// Go function
	go func() {
		// Using for loop
		for range ch {
			err := p.conn.Ping()
			if err != nil {
				logger.KV(xlog.ERROR, "reason", "ping", "err", err.Error())
				continue
			}
		}
		logger.KV(xlog.TRACE, "status", "stopped")
	}()
}

// BeginTx starts a transaction.
//
// The provided context is used until the transaction is committed or rolled back.
// If the context is canceled, the sql package will roll back
// the transaction. Tx.Commit will return an error if the context provided to
// BeginTx is canceled.
//
// The provided TxOptions is optional and may be nil if defaults should be used.
// If a non-default isolation level is used that the driver doesn't support,
// an error will be returned.
func (p *SQLProvider) BeginTx(ctx context.Context, _ *sql.TxOptions) (Provider, error) {
	if p.tx != nil {
		return nil, errors.New("transaction already started")
	}
	tx, err := p.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	txProv := &SQLProvider{
		name:  p.name,
		conn:  p.conn,
		db:    tx,
		idGen: p.idGen,
		tx:    tx,
	}
	return txProv, nil
}

// Close connection and release resources
func (p *SQLProvider) Close() (err error) {
	if p.ticker != nil {
		p.ticker.Stop()
		p.ticker = nil
	}
	if p.conn == nil {
		return nil
	}
	if p.tx != nil {
		return p.Rollback()
	}

	if err = p.conn.Close(); err != nil {
		logger.KV(xlog.ERROR, "err", err)
	} else {
		p.conn = nil
	}
	logger.KV(xlog.TRACE, "status", "closed")
	return
}

// DB returns underlying DB connection
func (p *SQLProvider) DB() DB {
	return p.db
}

// Tx returns underlying DB transaction
func (p *SQLProvider) Tx() Tx {
	return p.tx
}

// NextID returns unique ID
func (p *SQLProvider) NextID() ID {
	return NewID(p.idGen.NextID())
}

// IDTime returns time when ID was generated
func (p *SQLProvider) IDTime(id uint64) time.Time {
	return flake.IDTime(p.idGen, id)
}

// QueryContext executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.
func (p *SQLProvider) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row.
// QueryRowContext always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func (p *SQLProvider) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func (p *SQLProvider) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

func (p *SQLProvider) Commit() error {
	if p.tx == nil {
		return errors.New("no transaction started")
	}
	return p.tx.Commit()
}

func (p *SQLProvider) Rollback() error {
	if p.tx == nil {
		return errors.New("no transaction started")
	}
	// Rollback returns sql.ErrTxDone if the transaction was already
	if err := p.tx.Rollback(); err != nil && err != sql.ErrTxDone {
		return errors.WithStack(err)
	}
	return nil
}
