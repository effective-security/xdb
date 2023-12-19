// Package schema provides helper package to generate schema.
package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/effective-security/xdb/xsql"
)

//go:generate mockgen -source=schema.go -destination=../mocks/mockschema/schema_mock.go -package mockschema

// TableInfo defines a table info
type TableInfo struct {
	Schema     string
	Name       string
	PrimaryKey string
	Columns    []string
	Indexes    []string

	Dialect xsql.SQLDialect `json:"-" yaml:"-"`

	// SchemaName is FQN in schema.name format
	SchemaName string `json:"-" yaml:"-"`

	allColumns string `json:"-" yaml:"-"`
}

// From starts FROM expression
func (t *TableInfo) From() xsql.Builder {
	return t.Dialect.From(t.SchemaName)
}

// DeleteFrom starts DELETE FROM expression
func (t *TableInfo) DeleteFrom() xsql.Builder {
	return t.Dialect.DeleteFrom(t.SchemaName)
}

// InsertInto starts INSERT expression
func (t *TableInfo) InsertInto() xsql.Builder {
	return t.Dialect.InsertInto(t.SchemaName)
}

// Update starts UPDATE expression
func (t *TableInfo) Update() xsql.Builder {
	return t.Dialect.Update(t.SchemaName)
}

// Select starts SELECT FROM  expression
func (t *TableInfo) Select(cols ...string) xsql.Builder {
	var expr string
	if len(cols) > 0 {
		expr = strings.Join(cols, ",")
	} else {
		expr = t.AllColumns()
	}
	return t.Dialect.From(t.SchemaName).Select(expr)
}

// AllColumns returns list of all columns separated by comma
func (t *TableInfo) AllColumns() string {
	if t.allColumns == "" {
		t.allColumns = strings.Join(t.Columns, ", ")
	}
	return t.allColumns
}

// AliasedColumns returns list of columns separated by comma,
// with prefix a.C1, NULL, a.C2 etc.
// Columns identified in nulls, will be replaced with NULL.
func (t *TableInfo) AliasedColumns(prefix string, nulls map[string]bool) string {
	prefixed := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		if nulls[c] {
			prefixed[i] = "NULL"
		} else {
			prefixed[i] = prefix + "." + c
		}
	}
	return strings.Join(prefixed, ", ")
}

// Table definition
type Table struct {
	Schema  string
	Name    string
	IsView  bool
	Columns Columns
	Indexes Indexes

	PrimaryKey *Column

	// FKMap provides the cache of the FK
	FKMap map[string]*ForeignKey `json:"-" yaml:"-"`

	// SchemaName is FQN in schema.name format
	SchemaName string `json:"-" yaml:"-"`
}

// PrimaryKeyName returns the name of primary key
func (t *Table) PrimaryKeyName() string {
	if t != nil && t.PrimaryKey != nil {
		return t.PrimaryKey.Name
	}
	return ""
}

// Tables defines slice of Table
type Tables []*Table

// Column definition
type Column struct {
	Name      string
	Type      string
	UdtType   string
	Nullable  bool
	MaxLength uint32

	// GoName string
	// GoType string

	// SchemaName is FQN in schema.table.name format
	SchemaName string `json:"-" yaml:"-"`
	// Ref provides the FK reference
	Ref *ForeignKey `json:"-" yaml:"-"`
	// Indexes provides the index references, where the column is part of index
	Indexes Indexes `json:"-" yaml:"-"`
}

func (c *Column) StructString() string {
	ml := ""
	if c.MaxLength > 0 {
		ml = fmt.Sprintf(", MaxLength: %d ", c.MaxLength)
	}
	return fmt.Sprintf(`{ Name: "%s", Type: "%s", UdtType: "%s", Nullable: %t %s}`,
		c.Name, c.Type, c.UdtType, c.Nullable, ml,
	)
}

// IsIndex returns true if column is part of index
func (c *Column) IsIndex() bool {
	return len(c.Indexes) > 0
}

// IsPrimary returns true if column is primary key
func (c *Column) IsPrimary() bool {
	for _, idx := range c.Indexes {
		if idx.IsPrimary {
			return true
		}
	}
	return false
}

func (c *Column) Tag() string {
	ops := ""

	if c.UdtType != "" {
		ops += fmt.Sprintf(",%s", c.UdtType)
	} else {
		ops += fmt.Sprintf(",%s", c.Type)
	}
	if c.MaxLength > 0 {
		ops += fmt.Sprintf(",max:%d", c.MaxLength)
	}
	if c.Nullable {
		ops += ",null"
	}

	if len(c.Indexes) > 0 {
		ops += ",index"
		if c.IsPrimary() {
			ops += ",primary"
		}
	}
	if c.Ref != nil {
		ops += ",fk:" + c.Ref.RefColumnSchemaName()
	}
	return fmt.Sprintf("db:\"%s%s\"", c.Name, ops)
}

// Columns defines slice of Column
type Columns []*Column

// Names returns list of column names
func (c Columns) Names() []string {
	var list []string
	for _, col := range c {
		list = append(list, col.Name)
	}
	return list
}

// Index definition
type Index struct {
	Name        string
	IsPrimary   bool
	IsUnique    bool
	ColumnNames []string

	// SchemaName is FQN in schema.table.name format
	SchemaName string `json:"-" yaml:"-"`
}

// Indexes defines slice of Index
type Indexes []*Index

// Names returns list of index names
func (c Indexes) Names() []string {
	var list []string
	for _, col := range c {
		list = append(list, col.Name)
	}
	return list
}

// ForeignKey describes FK
type ForeignKey struct {
	Name string

	Schema string
	Table  string
	Column string

	RefSchema string
	RefTable  string
	RefColumn string

	// SchemaName is FQN in schema.table.name format
	SchemaName string `json:"-" yaml:"-"`
}

// ColumnSchemaName is FQN in schema.db.column format
func (k *ForeignKey) ColumnSchemaName() string {
	if k == nil {
		return ""
	}
	return fmt.Sprintf("%s.%s.%s", k.Schema, k.Table, k.Column)
}

// RefColumnSchemaName is FQN in schema.db.column format
func (k *ForeignKey) RefColumnSchemaName() string {
	if k == nil {
		return ""
	}
	return fmt.Sprintf("%s.%s.%s", k.RefSchema, k.RefTable, k.RefColumn)
}

// ForeignKeys defines slice of ForeingKey
type ForeignKeys []*ForeignKey

// Provider defines schema provider interface
type Provider interface {
	Name() string

	// ListTables returns a list of tables in database.
	// schemaName and tableNames are optional parameters to filter,
	// if not provided, then all items are returned
	ListTables(ctx context.Context, schemaName string, tableNames []string, withDependencies bool) (Tables, error)
	// ListViews returns a list of views in database.
	// schemaName and tableNames are optional parameters to filter,
	// if not provided, then all items are returned
	ListViews(ctx context.Context, schemaName string, tableNames []string) (Tables, error)
	// ListForeignKeys returns a list of FK in database.
	// schemaName and tableNames are optional parameters to filter on source tables,
	// if not provided, then all items are returned
	ListForeignKeys(ctx context.Context, schemaName string, tableNames []string) (ForeignKeys, error)
}
