package schema

import (
	"context"
	"testing"

	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/pkg/flake"
	"github.com/effective-security/xdb/xsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const XDB_SQL_DATASOURCE = "sqlserver://127.0.0.1:11434?user id=sa&password=notUsed123_P"
const XDB_PG_DATASOURCE = "postgres://postgres:postgres@127.0.0.1:15433?sslmode=disable"

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
	assert.Equal(t, `db:"org_id,int8,null" json:",omitempty"`, c.Tag())
	assert.Equal(t, `{ Name: "org_id", Position: 0, Type: "bigint", UdtType: "int8", Nullable: true }`, c.StructString())

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
	assert.Equal(t, `db:"id,int8,max:32,index,primary,fk:smb.t2.c2" json:",omitempty"`, c2.Tag())
	assert.Equal(t, `{ Name: "id", Position: 0, Type: "bigint", UdtType: "int8", Nullable: false , MaxLength: 32 }`, c2.StructString())

	cols := Columns{c, c2}
	assert.Equal(t, []string{"org_id", "id"}, cols.Names())
}

func TestListSQLServer(t *testing.T) {
	provider, err := xdb.NewProvider(
		XDB_SQL_DATASOURCE,
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

	require.Equal(t, "sqlserver", provider.Name())
	p := NewProvider(provider.DB(), provider.Name())

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
		XDB_PG_DATASOURCE,
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

	require.Equal(t, "postgres", provider.Name())
	p := NewProvider(provider.DB(), provider.Name())

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

func TestTableInfo(t *testing.T) {
	nulls := map[string]bool{
		"meta": true,
	}
	ti := TableInfo{
		Schema:     "public",
		Name:       "org",
		SchemaName: "public.org",
		Columns:    []string{"id", "meta", "name"},
		PrimaryKey: "id",
		Dialect:    xsql.Postgres,
	}
	assert.Equal(t, "id, meta, name", ti.AllColumns())
	assert.Equal(t, "a.id, NULL, a.name", ti.AliasedColumns("a", nulls))
	assert.Equal(t, "id, NULL, name", ti.AliasedColumns("", nulls))

	assert.Equal(t, `FROM public.org`, ti.From().String())
	assert.Equal(t, "SELECT id, meta, name \nFROM public.org", ti.Select().String())
	assert.Equal(t, "SELECT o.id, NULL, o.name \nFROM public.org o", ti.SelectAliased("o", map[string]bool{"meta": true}).String())
	assert.Equal(t, "SELECT id \nFROM public.org", ti.Select("id").String())
	assert.Equal(t, "UPDATE public.org \nSET id=$1 \nWHERE id = $2", ti.Update().Set("id", nil).Where("id = ?", nil).String())
	assert.Equal(t, "DELETE FROM public.org \nWHERE id = $1", ti.DeleteFrom().Where("id = ?", nil).String())
	assert.Equal(t, "INSERT INTO public.org \n( id \n) VALUES ( $1 \n)", ti.InsertInto().Set("id", nil).String())
}
