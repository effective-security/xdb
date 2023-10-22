package xdb

import (
	"context"
	"database/sql"
	"io"
	"os"
	"strings"
	"time"

	"github.com/effective-security/porto/pkg/flake"
	"github.com/effective-security/porto/x/fileutil"
	"github.com/effective-security/xdb/migrate"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=db.go -destination=./mocks/mockxdb/xdb_mock.go -package mockxdb

// IDGenerator defines an interface to generate unique ID accross the cluster
type IDGenerator interface {
	// NextID generates a next unique ID.
	NextID() ID
	IDTime(id uint64) time.Time
}

// Row defines an interface for DB row
type Row interface {
	// Scan copies the columns from the matched row into the values
	// pointed at by dest. See the documentation on Rows.Scan for details.
	// If more than one row matches the query,
	// Scan uses the first row and discards the rest. If no row matches
	// the query, Scan returns ErrNoRows.
	Scan(dest ...any) error
	// Err provides a way for wrapping packages to check for
	// query errors without calling Scan.
	// Err returns the error, if any, that was encountered while running the query.
	// If this error is not nil, this error will also be returned from Scan.
	Err() error
}

// Rows defines an interface for DB rows
type Rows interface {
	io.Closer
	Row

	// Next prepares the next result row for reading with the Scan method. It
	// returns true on success, or false if there is no next result row or an error
	// happened while preparing it. Err should be consulted to distinguish between
	// the two cases.
	//
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool
	// NextResultSet prepares the next result set for reading. It reports whether
	// there is further result sets, or false if there is no further result set
	// or if there is an error advancing to it. The Err method should be consulted
	// to distinguish between the two cases.
	//
	// After calling NextResultSet, the Next method should always be called before
	// scanning. If there are further result sets they may not have rows in the result
	// set.
	NextResultSet() bool
}

// DB provides interface for Db operations
type DB interface {
	// QueryContext executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRowContext executes a query that is expected to return at most one row.
	// QueryRowContext always returns a non-nil value. Errors are deferred until
	// Row's Scan method is called.
	// If the query selects no rows, the *Row's Scan will return ErrNoRows.
	// Otherwise, the *Row's Scan scans the first selected row and discards
	// the rest.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Tx provides interface for Tx operations
type Tx interface {
	DB

	Commit() error
	Rollback() error
}

// Provider provides complete DB access
type Provider interface {
	IDGenerator
	DB
	Tx

	// DB returns underlying DB connection
	DB() DB
	// Tx returns underlying DB transaction
	Tx() Tx

	// Close connection and release resources
	Close() (err error)

	BeginTx(ctx context.Context, opts *sql.TxOptions) (Provider, error)
}

// Open returns an SQL connection instance
func Open(driverName string, dataSourceName, database string) (*sql.DB, string, error) {
	ds, err := fileutil.LoadConfigWithSchema(dataSourceName)
	if err != nil {
		return nil, "", errors.WithMessagef(err, "failed to load config")
	}

	ds = strings.Trim(ds, "\"")
	ds = strings.TrimSpace(ds)

	if database != "" {
		switch driverName {
		case "sqlserver":
			ds = ds + "&database=" + database
		case "postgres":
			if strings.Contains(ds, "host=") {
				ds = ds + " dbname=" + database
			} else {
				ds = ds + "&dbname=" + database
			}
		default:
			return nil, ds, errors.Errorf("unsuppoprted driver %q", driverName)
		}
	}

	d, err := sql.Open(driverName, ds)
	if err != nil {
		return nil, ds, errors.WithMessagef(err, "unable to open DB")
	}

	d.SetConnMaxIdleTime(0)
	d.SetConnMaxLifetime(0)

	err = d.Ping()
	if err != nil {
		return nil, ds, errors.WithMessagef(err, "unable to ping DB")
	}

	return d, ds, nil
}

// MigrationConfig defines migration configuration
type MigrationConfig struct {
	Source         string
	ForceVersion   int
	MigrateVersion int
}

// NewProvider creates a Provider instance
func NewProvider(provider, dataSourceName, dbName string, idGen flake.IDGenerator, migrateCfg *MigrationConfig) (Provider, error) {
	d, _, err := Open(provider, dataSourceName, dbName)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to open DB")
	}

	if migrateCfg != nil && migrateCfg.Source != "" {
		migrationsDir := migrateCfg.Source
		if isWindows() {
			migrationsDir = strings.ReplaceAll(migrationsDir, "\\", "/")
		}

		err = migrate.Migrate(provider, dbName, migrationsDir, migrateCfg.ForceVersion, migrateCfg.MigrateVersion, d)
		if err != nil {
			return nil, errors.WithMessagef(err, "unable to migrate Orgs DB")
		}
	}
	return New(d, idGen)
}

func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}
