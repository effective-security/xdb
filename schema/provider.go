package schema

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/effective-security/x/slices"
	"github.com/effective-security/xdb"
	"github.com/pkg/errors"
)

// Dialect interface
type Dialect interface {
	QueryTables(ctx context.Context) (*sql.Rows, error)
	QueryViews(ctx context.Context) (*sql.Rows, error)
	QueryColumns(ctx context.Context, schema, table string) (*sql.Rows, error)
	QueryIndexes(ctx context.Context, schema, table string) (*sql.Rows, error)
	QueryForeignKeys(ctx context.Context) (*sql.Rows, error)
}

// SQLServerProvider implementation
type SQLServerProvider struct {
	db      xdb.DB
	dialect Dialect
	name    string

	tables  map[string]*Table      // map of Table FQN => table
	columns map[string]*Column     // map of Column FQN => column
	indexes map[string]*Index      // map of Column FQN => index
	fkeys   map[string]*ForeignKey // map of Column FQN => FK
}

// NewProvider return MS SQL reader
func NewProvider(db xdb.DB, provider string) Provider {
	var dialect Dialect
	switch provider {
	case "mssql", "sqlserver":
		dialect = &sqlserver{db: db}
	case "postgres":
		dialect = &postgres{db: db}
	}

	p := &SQLServerProvider{
		db:      db,
		name:    provider,
		columns: make(map[string]*Column),
		tables:  make(map[string]*Table),
		fkeys:   make(map[string]*ForeignKey),
		indexes: make(map[string]*Index),
		dialect: dialect,
	}

	return p
}

// Name returns provider name
func (r *SQLServerProvider) Name() string {
	return r.name
}

// ListTables returns a list of tables in database.
// schema and tables are optional parameters to filter,
// if not provided, then all items are returned
func (r *SQLServerProvider) ListTables(ctx context.Context, schema string, tables []string, withDependencies bool) (Tables, error) {
	rows, err := r.dialect.QueryTables(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query tables")
	}

	tt := Tables{}
	for rows.Next() {
		t := new(Table)
		if err := rows.Scan(&t.Schema, &t.Name); err != nil {
			return nil, errors.WithMessagef(err, "failed to scan")
		}

		if schema != "" && !strings.EqualFold(t.Schema, schema) {
			continue
		}

		if len(tables) > 0 && !slices.ContainsStringEqualFold(tables, t.Name) {
			continue
		}

		t.SchemaName = fmt.Sprintf("%s.%s", t.Schema, t.Name)

		cc, err := r.readColumnsSchema(ctx, t.Schema, t.Name)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to read columns: %s", t.SchemaName)
		}

		t.Columns = cc

		ii, _, err := r.readIndexesSchema(ctx, t.Schema, t.Name)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to read indexes: %s", t.SchemaName)
		}
		t.Indexes = ii

		for _, idx := range ii {
			for _, cn := range idx.ColumnNames {
				colShemaName := fmt.Sprintf("%s.%s", t.SchemaName, cn)
				col := r.columns[colShemaName]
				col.Indexes = append(col.Indexes, idx)
				if idx.IsPrimary && len(idx.ColumnNames) == 1 {
					t.PrimaryKey = col
				}
			}
		}

		r.tables[t.SchemaName] = t
		tt = append(tt, t)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if withDependencies {
		tt, err = r.discover(ctx)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(tt, func(i int, j int) bool {
		return tt[i].SchemaName < tt[j].SchemaName
	})

	return tt, nil
}

// ListViews returns a list of views in database.
// schemaName and tableNames are optional parameters to filter,
// if not provided, then all items are returned
func (r *SQLServerProvider) ListViews(ctx context.Context, schema string, tables []string) (Tables, error) {
	rows, err := r.dialect.QueryViews(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query tables")
	}

	tablesMap := map[string]*Table{} // map of Table FQN => table

	for rows.Next() {
		var schemaName string
		var tableName string
		c := &Column{}
		var nullable string
		var maxLen *int
		var ordinal int
		if err := rows.Scan(&schemaName, &tableName, &c.Name, &c.Type, &c.UdtType, &nullable, &maxLen, &ordinal); err != nil {
			return nil, errors.WithStack(err)
		}
		if schema != "" && !strings.EqualFold(schema, schemaName) {
			continue
		}

		if len(tables) > 0 && !slices.ContainsStringEqualFold(tables, tableName) {
			continue
		}
		c.Nullable = slices.ContainsStringEqualFold(nullableVals, nullable)
		c.MaxLength = maxLength(maxLen)
		c.Name = columnName(c.Name)
		c.SchemaName = fmt.Sprintf("%s.%s.%s", schemaName, tableName, c.Name)
		c.Position = uint32(ordinal)
		r.columns[c.SchemaName] = c

		tSchemaName := fmt.Sprintf("%s.%s", schemaName, tableName)
		t := tablesMap[tSchemaName]
		if t == nil {
			t = &Table{
				Name:       tableName,
				Schema:     schemaName,
				SchemaName: tSchemaName,
				IsView:     true,
			}
			tablesMap[tSchemaName] = t
		}
		t.Columns = append(t.Columns, c)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	tt := Tables{}
	for _, c := range tablesMap {
		sort.Slice(c.Columns, func(i int, j int) bool {
			return c.Columns[i].Position < c.Columns[j].Position
		})

		tt = append(tt, c)
	}

	sort.Slice(tt, func(i int, j int) bool {
		return tt[i].SchemaName < tt[j].SchemaName
	})
	return tt, nil
}

var nullableVals = []string{"YES", "TRUE", "NULL"}

func (r *SQLServerProvider) readColumnsSchema(ctx context.Context, schema, table string) (Columns, error) {
	rows, err := r.dialect.QueryColumns(ctx, schema, table)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cc := Columns{}
	for rows.Next() {
		c := &Column{}
		var nullable string
		var maxLen *int
		var ordinal int
		if err := rows.Scan(&c.Name, &c.Type, &c.UdtType, &nullable, &maxLen, &ordinal); err != nil {
			return nil, errors.WithStack(err)
		}
		c.Position = uint32(ordinal)
		c.Nullable = slices.ContainsStringEqualFold(nullableVals, nullable)
		c.MaxLength = maxLength(maxLen)
		c.Name = columnName(c.Name)
		c.SchemaName = fmt.Sprintf("%s.%s.%s", schema, table, c.Name)
		r.columns[c.SchemaName] = c

		cc = append(cc, c)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	sort.Slice(cc, func(i int, j int) bool {
		return cc[i].Position < cc[j].Position
	})

	return cc, nil
}

func (r *SQLServerProvider) readIndexesSchema(ctx context.Context, schema, table string) (Indexes, *Index, error) {
	rows, err := r.dialect.QueryIndexes(ctx, schema, table)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	var pk *Index
	cc := Indexes{}
	for rows.Next() {
		c := &Index{}
		var columnNames string
		if err := rows.Scan(&c.Name, &c.IsPrimary, &c.IsUnique, &columnNames); err != nil {
			return nil, nil, errors.WithStack(err)
		}

		c.Name = columnName(c.Name)
		for _, cn := range strings.Split(columnNames, ",") {
			cn = columnName(cn)
			c.ColumnNames = append(c.ColumnNames, cn)
		}
		c.SchemaName = fmt.Sprintf("%s.%s.%s", schema, table, c.Name)
		r.indexes[c.SchemaName] = c

		cc = append(cc, c)

		if c.IsPrimary {
			pk = c
		}
	}

	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}

	return cc, pk, nil
}

// ListForeignKeys returns a list of FK in database.
// schema and tables are optional parameters to filter on source tables,
// if not provided, then all items are returned
func (r *SQLServerProvider) ListForeignKeys(ctx context.Context, schema string, tables []string) (ForeignKeys, error) {
	rows, err := r.dialect.QueryForeignKeys(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query foreign keys")
	}

	keys := ForeignKeys{}
	for rows.Next() {
		k := new(ForeignKey)
		if err := rows.Scan(
			&k.Name,
			&k.Schema,
			&k.Table,
			&k.Column,
			&k.RefSchema,
			&k.RefTable,
			&k.RefColumn,
		); err != nil {
			return nil, errors.WithMessagef(err, "failed to scan foreign keys")
		}

		if schema != "" && !strings.EqualFold(k.Schema, schema) {
			continue
		}
		if len(tables) > 0 && !slices.ContainsStringEqualFold(tables, k.Table) {
			continue
		}

		k.Column = columnName(k.Column)
		k.RefColumn = columnName(k.RefColumn)
		k.SchemaName = fmt.Sprintf("%s.%s.%s", k.Schema, k.Table, k.Name)
		r.fkeys[k.ColumnSchemaName()] = k

		keys = append(keys, k)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return keys, nil
}

// discover will DFS on the graph and update internal cache with all dependencies
func (r *SQLServerProvider) discover(ctx context.Context) (Tables, error) {
	_, err := r.ListForeignKeys(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	// the cache now consists of Tables, Columns and FKeys
	for n, c := range r.columns {
		fk := r.fkeys[n]
		if fk == nil {
			continue
		}
		c.Ref = fk

		err = r.discoverTable(ctx, fk.RefSchema, c.Ref.RefTable)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to discover: %s.%s", fk.RefSchema, c.Ref.RefTable)
		}
	}

	// return all discovered tables
	res := Tables{}
	for _, t := range r.tables {
		res = append(res, t)
	}

	return res, nil
}

func (r *SQLServerProvider) discoverTable(ctx context.Context, schema, table string) error {
	tref := fmt.Sprintf("%s.%s", schema, table)
	if r.tables[tref] != nil {
		return nil
	}

	t := &Table{
		Name:       table,
		Schema:     schema,
		SchemaName: fmt.Sprintf("%s.%s", schema, table),
	}
	cc, err := r.readColumnsSchema(ctx, t.Schema, t.Name)
	if err != nil {
		return errors.WithMessagef(err, "failed to read columns: %s", t.SchemaName)
	}

	t.Columns = cc
	r.tables[t.SchemaName] = t

	// traverse columns
	for _, c := range cc {
		fk := r.fkeys[c.SchemaName]
		if fk == nil {
			continue
		}
		c.Ref = fk

		err = r.discoverTable(ctx, fk.RefSchema, c.Ref.RefTable)
		if err != nil {
			return err
		}
	}

	return nil
}

func columnName(s string) string {
	return s
	// if s[0] == '_' {
	// 	a := []rune(s)
	// 	a[0] = 'X'
	// 	s = string(a)
	// }
	// return strcase.ToGoPascal(s)
}

func maxLength(v *int) uint32 {
	if v == nil {
		return 0
	}
	return uint32(*v)
}
