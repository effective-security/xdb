// Package cli provides CLI app and global flags
package cli

import (
	"context"
	"database/sql"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/effective-security/porto/x/slices"
	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/pkg/print"
	"github.com/effective-security/xdb/schema"
	"github.com/effective-security/xlog"
	"github.com/effective-security/xpki/x/ctl"
	"github.com/pkg/errors"
)

// Cli provides CLI context to run commands
type Cli struct {
	Version ctl.VersionFlag `name:"version" help:"Print version information and quit" hidden:""`
	Debug   bool            `short:"D" help:"Enable debug mode"`
	O       string          `help:"Print output format: json|yaml|table" default:"table"`

	Provider  string `kong:"required" help:"SQL provider name: sqlserver|postgres"`
	SQLSource string `help:"SQL sources, if not provided, will be used from XDB_DATASOURCE env var"`

	// Stdin is the source to read from, typically set to os.Stdin
	stdin io.Reader
	// Output is the destination for all output from the command, typically set to os.Stdout
	output io.Writer
	// ErrOutput is the destinaton for errors.
	// If not set, errors will be written to os.StdError
	errOutput io.Writer

	ctx    context.Context
	schema schema.Provider
	db     *sql.DB
}

// Close used resources
func (c *Cli) Close() {
	if c.db != nil {
		_ = c.db.Close()
		c.db = nil
	}
}

// Context for requests
func (c *Cli) Context() context.Context {
	if c.ctx == nil {
		c.ctx = context.Background()
	}
	return c.ctx
}

// Reader is the source to read from, typically set to os.Stdin
func (c *Cli) Reader() io.Reader {
	if c.stdin != nil {
		return c.stdin
	}
	return os.Stdin
}

// WithReader allows to specify a custom reader
func (c *Cli) WithReader(reader io.Reader) *Cli {
	c.stdin = reader
	return c
}

// Writer returns a writer for control output
func (c *Cli) Writer() io.Writer {
	if c.output != nil {
		return c.output
	}
	return os.Stdout
}

// WithWriter allows to specify a custom writer
func (c *Cli) WithWriter(out io.Writer) *Cli {
	c.output = out
	return c
}

// ErrWriter returns a writer for control output
func (c *Cli) ErrWriter() io.Writer {
	if c.errOutput != nil {
		return c.errOutput
	}
	return os.Stderr
}

// WithErrWriter allows to specify a custom error writer
func (c *Cli) WithErrWriter(out io.Writer) *Cli {
	c.errOutput = out
	return c
}

// AfterApply hook loads config
func (c *Cli) AfterApply(_ *kong.Kong, _ kong.Vars) error {
	if c.Debug {
		xlog.SetGlobalLogLevel(xlog.DEBUG)
	} else {
		xlog.SetGlobalLogLevel(xlog.ERROR)
	}

	c.SQLSource = slices.StringsCoalesce(c.SQLSource, os.Getenv("XDB_DATASOURCE"))
	if c.SQLSource == "" {
		return errors.Errorf("use --sql-source or set XDB_DATASOURCE")
	}

	return nil
}

// DB returns DB connection
func (c *Cli) DB(dbname string) (*sql.DB, error) {
	if c.db == nil {
		d, _, err := xdb.Open(c.Provider, c.SQLSource, dbname)
		if err != nil {
			return nil, err
		}
		c.db = d
	}
	return c.db, nil
}

// SchemaProvider returns schema.Provider
func (c *Cli) SchemaProvider(dbname string) (schema.Provider, error) {
	if c.schema == nil {
		db, err := c.DB(dbname)
		if err != nil {
			return nil, err
		}

		c.schema = schema.NewProvider(db, c.Provider)
	}

	return c.schema, nil
}

// WithSchemaProvider allows to specify a custom schema provider
func (c *Cli) WithSchemaProvider(p schema.Provider) *Cli {
	c.schema = p
	return c
}

// Print response to out
func (c *Cli) Print(value any) error {
	return print.Object(c.Writer(), c.O, value)
}
