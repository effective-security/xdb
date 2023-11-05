// Package schema provides helper package to generate schema.
package schema

import (
	"context"
	"fmt"
)

//go:generate mockgen -source=schema.go -destination=../mocks/mockschema/schema_mock.go -package mockschema

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
	Name    string
	Type    string
	UdtType string
	// GoName      string
	// GoType    string
	Nullable  string
	MaxLength *int

	// SchemaName is FQN in schema.table.name format
	SchemaName string `json:"-" yaml:"-"`
	// Ref provides the FK reference
	Ref *ForeignKey `json:"-" yaml:"-"`
	// Indexes provides the index references, where the column is part of index
	Indexes Indexes `json:"-" yaml:"-"`
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
	if c.MaxLength != nil {
		ops += fmt.Sprintf(",max:%d", *c.MaxLength)
	}
	if c.Nullable == "YES" {
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
