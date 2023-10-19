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

func TestFKSchemaname(t *testing.T) {
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
}
