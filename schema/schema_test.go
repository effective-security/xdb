package schema

import (
	"context"
	"os"
	"testing"

	"github.com/effective-security/porto/pkg/flake"
	"github.com/effective-security/xdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel(t *testing.T) {
	var fk *ForeignKey
	assert.Empty(t, fk.ColumnSchemaName())
	assert.Empty(t, fk.RefColumnSchemaName())

	fk = &ForeignKey{
		Name:      "FK_1",
		Table:     "t1",
		Column:    "c1",
		Schema:    "dbo",
		RefTable:  "t2",
		RefColumn: "c2",
		RefSchema: "smb",
	}
	assert.Equal(t, "dbo.t1.c1", fk.ColumnSchemaName())
	assert.Equal(t, "smb.t2.c2", fk.RefColumnSchemaName())

	idx := &Index{
		Name:        "idx",
		ColumnNames: []string{"id", "name"},
		IsPrimary:   true,
		IsUnique:    true,
	}
	idxs := Indexes{idx}
	assert.Equal(t, []string{"idx"}, idxs.Names())

	c := &Column{
		Name:     "org_id",
		Type:     "bigint",
		UdtType:  "int8",
		Nullable: true,
	}
	assert.False(t, c.IsIndex())
	assert.False(t, c.IsPrimary())
	assert.Equal(t, `db:"org_id,int8,null"`, c.Tag())
	assert.Equal(t, `{ Name: "org_id", Type: "bigint", UdtType: "int8", Nullable: true }`, c.StructString())

	c2 := &Column{
		Name:      "id",
		Type:      "bigint",
		UdtType:   "int8",
		Nullable:  false,
		MaxLength: 32,
		Ref:       fk,
		Indexes:   idxs,
	}
	assert.True(t, c2.IsIndex())
	assert.True(t, c2.IsPrimary())
	assert.Equal(t, `db:"id,int8,max:32,index,primary,fk:smb.t2.c2"`, c2.Tag())
	assert.Equal(t, `{ Name: "id", Type: "bigint", UdtType: "int8", Nullable: false , MaxLength: 32 }`, c2.StructString())

	cols := Columns{c, c2}
	assert.Equal(t, []string{"org_id", "id"}, cols.Names())
}

func TestListSQLServer(t *testing.T) {
	provider, err := xdb.NewProvider(
		"sqlserver",
		os.Getenv("XDB_SQL_DATASOURCE"),
		"testdb",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "../testdata/sql/sqlserver/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	defer func() {
		err := provider.Close()
		assert.NoError(t, err)
	}()

	p := NewProvider(provider.DB(), "sqlserver")

	tt, err := p.ListTables(context.Background(), "dbo", []string{"Fake"}, true)
	require.NoError(t, err)
	assert.Empty(t, tt)

	fk, err := p.ListForeignKeys(context.Background(), "dbo", []string{"orgmember"})
	require.NoError(t, err)
	assert.Equal(t, 4, len(fk))

	tt, err = p.ListTables(context.Background(), "dbo", []string{"org"}, true)
	require.NoError(t, err)
	assert.Equal(t, 1, len(tt))

	tt, err = p.ListTables(context.Background(), "dbo", []string{"orgmember"}, true)
	require.NoError(t, err)
	assert.Equal(t, 3, len(tt))

	var tr *Table
	for _, t := range tt {
		if t.Name == "org" {
			tr = t
			break
		}
	}
	require.NotNil(t, tr)
	assert.Equal(t, 15, len(tr.Columns))
	assert.Equal(t, 5, len(tr.Indexes))
	require.NotNil(t, tr.PrimaryKey)
	assert.Equal(t, "id", tr.PrimaryKeyName())

	tt, err = p.ListViews(context.Background(), "dbo", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, len(tt))
}

func TestListPostgres(t *testing.T) {
	provider, err := xdb.NewProvider(
		"postgres",
		os.Getenv("XDB_PG_DATASOURCE"),
		"testdb",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "../testdata/sql/postgres/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	defer func() {
		err := provider.Close()
		assert.NoError(t, err)
	}()

	p := NewProvider(provider.DB(), "postgres")

	tt, err := p.ListTables(context.Background(), "public", []string{"Fake"}, true)
	require.NoError(t, err)
	assert.Empty(t, tt)

	fk, err := p.ListForeignKeys(context.Background(), "public", []string{"orgmember"})
	require.NoError(t, err)
	assert.NotEmpty(t, fk)

	tt, err = p.ListTables(context.Background(), "public", []string{"org"}, true)
	require.NoError(t, err)
	require.Equal(t, 1, len(tt))

	tt, err = p.ListTables(context.Background(), "public", []string{"orgmember"}, true)
	require.NoError(t, err)
	assert.Equal(t, 3, len(tt))

	var tr *Table
	for _, t := range tt {
		if t.Name == "org" {
			tr = t
			break
		}
	}
	require.NotNil(t, tr)
	assert.Equal(t, 15, len(tr.Columns))
	assert.Equal(t, 5, len(tr.Indexes))
	require.NotNil(t, tr.PrimaryKey)
	assert.Equal(t, "id", tr.PrimaryKeyName())

	tt, err = p.ListViews(context.Background(), "public", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, len(tt))
}
