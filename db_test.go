package xdb_test

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"testing"

	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/pkg/flake"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const XDB_SQL_DATASOURCE = "sqlserver://127.0.0.1:11434?user id=sa&password=notUsed123_P"
const XDB_PG_DATASOURCE = "postgres://postgres:postgres@127.0.0.1:15433?sslmode=disable"

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

type UserResult struct {
	Rows        []*user
	NextOffset  uint32
	HasNextPage bool
	Cursor      string
}

func (p *UserResult) SetResult(rows []*user, hasNextPage bool, nextOffset uint32) {
	p.Rows = rows
	p.NextOffset = nextOffset
	p.HasNextPage = hasNextPage
}

func (p *UserResult) SetResultWithCursor(rows []*user, hasNextPage bool, cursor func(lastRow *user) string) {
	p.Rows = rows
	p.HasNextPage = hasNextPage
	if hasNextPage && len(rows) > 0 {
		p.Cursor = cursor(rows[len(rows)-1])
	} else {
		p.Cursor = ""
	}
}

func TestProv(t *testing.T) {
	s, err := xdb.ParseConnectionString("postgres://u1:p2@127.0.0.1:55432?sslmode=disable&dbname=testdb")
	require.NoError(t, err)
	assert.Equal(t, "postgres", s.Driver)
	assert.Equal(t, "127.0.0.1:55432", s.Host)
	assert.Equal(t, "u1", s.User)
	assert.Equal(t, "p2", s.Password)
	assert.Equal(t, "testdb", s.Database)
}

func TestPG(t *testing.T) {
	ctx := context.Background()
	provider, err := xdb.NewProvider(
		XDB_PG_DATASOURCE,
		"testdb",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "testdata/sql/postgres/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	defer func() {
		err := provider.Close()
		assert.NoError(t, err)
	}()

	assert.EqualError(t, provider.Commit(), "no transaction started")
	assert.EqualError(t, provider.Rollback(), "no transaction started")

	id := provider.NextID()
	assert.False(t, id.IsZero())
	assert.False(t, provider.(*xdb.SQLProvider).IDTime(id.UInt64()).IsZero())

	t.Run("ListTables", func(t *testing.T) {
		expectedTables := []string{"org", "orgmember", "schema_migrations", "user"}
		require.NotNil(t, provider)
		require.NotNil(t, provider.DB())
		ctx := ctx
		res, err := provider.QueryContext(ctx, `
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

	t.Run("ExecuteQueryWithPagination", func(t *testing.T) {
		qp := xdb.NewQueryParams("ListUsers")
		qp.SetPage(2, 0)

		var rs UserResult
		err := xdb.ExecuteQueryWithPagination(ctx, provider.DB(), &rs,
			`SELECT id, email,email_verified, name FROM public.user LIMIT $1 OFFSET $2`, qp)
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)
		assert.True(t, rs.HasNextPage)

		err = xdb.ExecuteQueryWithPagination[user, *user](ctx, provider.DB(), &rs,
			`SELECT id, email,email_verified, name FROM public.user LIMIT $1 OFFSET $2`, 2, rs.NextOffset)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rs.Rows))
		assert.Equal(t, uint32(0), rs.NextOffset)
		assert.False(t, rs.HasNextPage)
	})

	t.Run("ExecuteQuery", func(t *testing.T) {
		qp := xdb.NewQueryParams("ListUsers")

		var rs UserResult
		err := xdb.ExecuteQuery(ctx, provider.DB(), &rs,
			`SELECT id, email,email_verified, name FROM public.user LIMIT 3`, qp)
		require.NoError(t, err)
		assert.Equal(t, 3, len(rs.Rows))
		assert.Equal(t, uint32(0), rs.NextOffset)
	})

	t.Run("Tx", func(t *testing.T) {
		ptx, err := provider.BeginTx(ctx, nil)
		require.NoError(t, err)

		_, err = ptx.BeginTx(ctx, nil)
		assert.EqualError(t, err, "transaction already started")

		var rs UserResult
		err = xdb.ExecuteQueryWithPagination(ctx, ptx.DB(), &rs,
			`SELECT id, email,email_verified, name FROM public.user LIMIT $1 OFFSET $2`, 2, 0)
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		assert.NoError(t, ptx.Tx().Commit())
		assert.EqualError(t, provider.Commit(), "no transaction started")
		assert.EqualError(t, provider.Rollback(), "no transaction started")

		assert.NoError(t, ptx.Close())
		assert.NoError(t, ptx.Close())
	})

	t.Run("TxRollback", func(t *testing.T) {
		ptx, err := provider.BeginTx(ctx, nil)
		require.NoError(t, err)

		row := ptx.QueryRowContext(ctx, `SELECT id FROM public.orgmember WHERE id=$1`, 666666)
		assert.NoError(t, row.Err())
		var id uint64
		assert.Error(t, row.Scan(&id))
		assert.NoError(t, row.Err())

		res, err := ptx.ExecContext(ctx, `DELETE FROM public.orgmember WHERE id=$1`, 12345)
		require.NoError(t, err)
		rows, err := res.RowsAffected()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), rows)

		assert.NoError(t, ptx.Close())
		assert.EqualError(t, provider.Commit(), "no transaction started")
		assert.EqualError(t, provider.Rollback(), "no transaction started")
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
		XDB_SQL_DATASOURCE,
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

	assert.EqualError(t, provider.Commit(), "no transaction started")
	assert.EqualError(t, provider.Rollback(), "no transaction started")

	t.Run("ListTables", func(t *testing.T) {
		expectedTables := []string{"org", "orgmember", "schema_migrations", "user"}
		require.NotNil(t, provider)
		require.NotNil(t, provider.DB())

		res, err := provider.QueryContext(ctx, mssqlTableNamesWithSchema)
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

	t.Run("ExecuteQueryWithPagination", func(t *testing.T) {
		q := `SELECT id, email,email_verified, name FROM [dbo].[user] 
ORDER BY id 
OFFSET @offset ROWS 
FETCH NEXT @take ROWS ONLY`

		var rs UserResult
		err := xdb.ExecuteQueryWithPagination(ctx, provider.DB(), &rs, q, sql.Named("take", 2), sql.Named("offset", 0))
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		err = xdb.ExecuteQueryWithPagination(ctx, provider.DB(), &rs, q, sql.Named("take", 2), sql.Named("offset", rs.NextOffset))
		require.NoError(t, err)
		assert.Equal(t, 1, len(rs.Rows))
		assert.Equal(t, uint32(0), rs.NextOffset)
	})

	t.Run("Tx", func(t *testing.T) {
		ptx, err := provider.BeginTx(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, ptx.Tx())
		assert.NotNil(t, ptx.DB())

		q := `SELECT id, email,email_verified, name FROM [dbo].[user] 
		ORDER BY id 
		OFFSET @offset ROWS 
		FETCH NEXT @take ROWS ONLY`

		var rs UserResult
		err = xdb.ExecuteQueryWithPagination(ctx, provider.DB(), &rs, q, sql.Named("take", 2), sql.Named("offset", 0))
		require.NoError(t, err)
		assert.Equal(t, uint32(len(rs.Rows)), rs.NextOffset)

		assert.NoError(t, ptx.Commit())

		assert.EqualError(t, provider.Commit(), "no transaction started")
		assert.EqualError(t, provider.Rollback(), "no transaction started")

		assert.NoError(t, ptx.Close())
		assert.NoError(t, ptx.Close())
	})

	t.Run("TxRollback", func(t *testing.T) {
		ptx, err := provider.BeginTx(ctx, nil)
		require.NoError(t, err)

		row := ptx.QueryRowContext(ctx, `SELECT org_id FROM [dbo].[orgmember] WHERE org_id=$1;`, 666)
		assert.NoError(t, row.Err())
		var id uint64
		assert.NoError(t, row.Scan(&id))

		res, err := ptx.ExecContext(ctx, `DELETE FROM [dbo].[orgmember] WHERE org_id=$1;`, 666)
		require.NoError(t, err)
		rows, err := res.RowsAffected()
		assert.NoError(t, err)
		// TODO: why 2?
		assert.Equal(t, int64(2), rows)

		assert.NoError(t, ptx.Close())
		assert.EqualError(t, provider.Commit(), "no transaction started")
		assert.EqualError(t, provider.Rollback(), "no transaction started")
		assert.NoError(t, ptx.Close())
	})
}
