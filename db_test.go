package xdb_test

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"testing"

	"github.com/effective-security/porto/pkg/flake"
	"github.com/effective-security/xdb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pgDataSource = "host=localhost port=5432 user=postgres password=postgres sslmode=disable"
const msDataSource = "sqlserver://localhost?user id=sa&password=notUsed123_P"

// User provides basic user information
type user struct {
	ID            xdb.ID `db:"id"`
	Email         string `db:"email"`
	EmailVerified bool   `db:"email_verified"`
	Name          string `db:"name"`
}

func (m *user) ScanRow(row xdb.Row) error {
	err := row.Scan(
		&m.ID,
		&m.Email,
		&m.EmailVerified,
		&m.Name,
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func TestPG(t *testing.T) {
	ctx := context.Background()
	provider, err := xdb.NewProvider(
		"postgres",
		pgDataSource,
		"testdb",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "testdata/sql/pgsql/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	defer func() {
		err := provider.Close()
		assert.NoError(t, err)
	}()

	id := provider.NextID()
	assert.False(t, id.IsZero())
	assert.False(t, provider.(*xdb.SQLProvider).IDTime(id.UInt64()).IsZero())

	t.Run("ListTables", func(t *testing.T) {
		expectedTables := []string{"org", "orgmember", "schema_migrations", "user"}
		require.NotNil(t, provider)
		require.NotNil(t, provider.DB())
		ctx := ctx
		res, err := provider.DB().QueryContext(ctx, `
	SELECT
		tablename
	FROM
		pg_catalog.pg_tables
	;`)
		require.NoError(t, err)
		defer func() {
			err = res.Close()
			assert.NoError(t, err)
		}()

		var tables []string
		var table string
		for res.Next() {
			err = res.Scan(&table)
			require.NoError(t, err)
			if !strings.HasPrefix(table, "sql_") && !strings.HasPrefix(table, "pg_") {
				tables = append(tables, table)
			}
		}
		assert.NoError(t, res.Err())
		sort.Strings(tables)
		assert.Equal(t, expectedTables, tables)
	})

	t.Run("RunQueryResult", func(t *testing.T) {
		var rs xdb.Result[user, *user]
		err = rs.RunQueryResult(ctx,
			provider.DB(),
			0,
			2,
			`SELECT id, email,email_verified, name FROM public.user LIMIT $1 OFFSET $2`, 2, 0)
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		err = rs.RunQueryResult(ctx,
			provider.DB(),
			2,
			2,
			`SELECT id, email,email_verified, name FROM public.user LIMIT $1 OFFSET $2`, 2, 2)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rs.Rows))
		assert.Equal(t, uint32(0), rs.NextOffset)
	})

	t.Run("Tx", func(t *testing.T) {
		ptx, err := provider.BeginTx(ctx, nil)
		require.NoError(t, err)
		var rs xdb.Result[user, *user]
		err = rs.RunQueryResult(ctx,
			ptx.DB(),
			0,
			2,
			`SELECT id, email,email_verified, name FROM public.user LIMIT $1 OFFSET $2`, 2, 0)
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		assert.NoError(t, ptx.Tx().Commit())
		assert.NoError(t, ptx.Close())
	})
}

const mssqlTableNamesWithSchema = `
	SELECT
		schema_name(t.schema_id),
		t.name
	FROM
		sys.tables t
	INNER JOIN
		sys.schemas s
	ON	s.schema_id = t.schema_id
	LEFT JOIN
		sys.extended_properties ep
	ON	ep.major_id = t.[object_id]
	WHERE
		t.is_ms_shipped = 0 AND
		(ep.class_desc IS NULL OR (ep.class_desc <> 'OBJECT_OR_COLUMN' AND
			ep.[name] <> 'microsoft_database_tools_support'))
	ORDER BY
		schema_name(t.schema_id),
		t.name
`

func TestMS(t *testing.T) {
	ctx := context.Background()
	provider, err := xdb.NewProvider(
		"sqlserver",
		msDataSource,
		"testdb",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "testdata/sql/sqlserver/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	defer func() {
		err := provider.Close()
		assert.NoError(t, err)
	}()

	t.Run("ListTables", func(t *testing.T) {
		expectedTables := []string{"org", "orgmember", "schema_migrations", "user"}
		require.NotNil(t, provider)
		require.NotNil(t, provider.DB())

		res, err := provider.DB().QueryContext(ctx, mssqlTableNamesWithSchema)
		require.NoError(t, err)
		defer func() {
			err = res.Close()
			assert.NoError(t, err)
		}()

		var tables []string
		var schema string
		var table string
		for res.Next() {
			err = res.Scan(&schema, &table)
			require.NoError(t, err)
			tables = append(tables, table)
		}
		assert.NoError(t, res.Err())
		sort.Strings(tables)
		assert.Equal(t, expectedTables, tables)
	})

	t.Run("RunQueryResult", func(t *testing.T) {
		var rs xdb.Result[user, *user]
		err = rs.RunQueryResult(ctx,
			provider.DB(),
			0,
			2,
			`SELECT id, email,email_verified, name FROM [dbo].[user] 
			ORDER BY id 
			OFFSET @offset ROWS 
			FETCH NEXT @take ROWS ONLY`,
			sql.Named("offset", 0),
			sql.Named("take", 2))
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		err = rs.RunQueryResult(ctx,
			provider.DB(),
			2,
			2,
			`SELECT id, email,email_verified, name FROM [dbo].[user] 
			ORDER BY id 
			OFFSET @offset ROWS 
			FETCH NEXT @take ROWS ONLY`,
			sql.Named("offset", 2),
			sql.Named("take", 2))
		require.NoError(t, err)
		assert.Equal(t, 1, len(rs.Rows))
		assert.Equal(t, uint32(0), rs.NextOffset)
	})

	t.Run("Tx", func(t *testing.T) {
		ptx, err := provider.BeginTx(ctx, nil)
		require.NoError(t, err)
		var rs xdb.Result[user, *user]
		err = rs.RunQueryResult(ctx,
			provider.DB(),
			0,
			2,
			`SELECT id, email,email_verified, name FROM [dbo].[user] 
			ORDER BY id 
			OFFSET @offset ROWS 
			FETCH NEXT @take ROWS ONLY`,
			sql.Named("offset", 0),
			sql.Named("take", 2))
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		assert.NoError(t, ptx.Tx().Commit())
		assert.NoError(t, ptx.Close())
	})

}
