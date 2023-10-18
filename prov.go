package xdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/effective-security/porto/pkg/flake"
	"github.com/effective-security/xlog"
	"github.com/pkg/errors"
)

var logger = xlog.NewPackageLogger("github.com/effective-security/xdb", "xdb")

// SQLProvider represents SQL client instance
type SQLProvider struct {
	conn   *sql.DB
	sql    DB
	idGen  flake.IDGenerator
	tx     *sql.Tx
	ticker *time.Ticker
}

// New creates a Provider instance
func New(db *sql.DB, idGen flake.IDGenerator) (*SQLProvider, error) {
	p := &SQLProvider{
		conn:  db,
		sql:   db,
		idGen: idGen,
	}

	p.keepAlive(60 * time.Second)

	return p, nil
}

func (p *SQLProvider) keepAlive(period time.Duration) {
	if p.ticker != nil {
		p.ticker.Stop()
		p.ticker = nil
	}

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
	tx, err := p.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	txProv := &SQLProvider{
		conn:  p.conn,
		sql:   tx,
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

	if p.conn == nil || p.tx != nil {
		return
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
	return p.conn
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
