package xsql_test

import (
	"testing"

	"github.com/effective-security/xdb/xsql"
	"github.com/stretchr/testify/require"
)

func TestPostgresUpdateDelete(t *testing.T) {
	q := xsql.Postgres.Update("users").
		Set("name", "Ada").
		SetExpr("login_count", "login_count + 1").
		Where("id = ?", 7)
	require.Equal(t, "UPDATE users \nSET name=$1, login_count=login_count + 1 \nWHERE id = $2", q.String())
	require.Equal(t, []any{"Ada", 7}, q.Args())
	q.Close()

	q = xsql.Postgres.DeleteFrom("users").Where("id = ?", 7)
	require.Equal(t, "DELETE FROM users \nWHERE id = $1", q.String())
	require.Equal(t, []any{7}, q.Args())
	q.Close()
}

func TestPostgresReturning(t *testing.T) {
	var id int64
	q := xsql.Postgres.InsertInto("users").
		Set("email", "a@b.c").
		Returning("id").To(&id)
	require.Equal(t, "INSERT INTO users \n( email \n) VALUES ( $1 \n) \nRETURNING id", q.String())
	require.Equal(t, []any{"a@b.c"}, q.Args())
	require.Equal(t, []any{&id}, q.Dest())
	q.Close()
}

func TestPostgresOnConflict(t *testing.T) {
	q := xsql.Postgres.InsertInto("users").
		Set("email", "a@b.c").
		Set("name", "Ada").
		OnConflict("(email) DO UPDATE SET name = EXCLUDED.name").
		Returning("id")
	require.Equal(t, "INSERT INTO users \n( email, name \n) VALUES ( $1, $2 \n) \nON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name \nRETURNING id", q.String())
	q.Close()

	q = xsql.Postgres.InsertInto("users").
		Set("email", "a@b.c").
		OnConflict("DO NOTHING")
	require.Equal(t, "INSERT INTO users \n( email \n) VALUES ( $1 \n) \nON CONFLICT DO NOTHING", q.String())
	q.Close()

	q = xsql.Postgres.InsertInto("users").
		Set("email", "a@b.c").
		OnConflict("").
		Clause("DO NOTHING")
	require.Equal(t, "INSERT INTO users \n( email \n) VALUES ( $1 \n) \nON CONFLICT \nDO NOTHING", q.String())
	q.Close()

	// RETURNING before OnConflict must still render ON CONFLICT first.
	q = xsql.Postgres.InsertInto("users").
		Set("email", "a@b.c").
		Returning("id").
		OnConflict("DO NOTHING")
	require.Equal(t, "INSERT INTO users \n( email \n) VALUES ( $1 \n) \nON CONFLICT DO NOTHING \nRETURNING id", q.String())
	q.Close()
}

func TestPostgresLocks(t *testing.T) {
	q := xsql.Postgres.Select("id").From("accounts").Where("id = ?", 1).ForUpdate("NOWAIT")
	require.Equal(t, "SELECT id \nFROM accounts \nWHERE id = $1 \nFOR UPDATE NOWAIT", q.String())
	q.Close()

	q = xsql.Postgres.Select("id").From("accounts").Where("id = ?", 1).ForShare("OF accounts")
	require.Equal(t, "SELECT id \nFROM accounts \nWHERE id = $1 \nFOR SHARE OF accounts", q.String())
	q.Close()

	// Clauses added after ForUpdate still appear before the lock.
	q = xsql.Postgres.Select("id").From("accounts").Where("id = ?", 1).ForUpdate("NOWAIT").OrderBy("id").Limit(1)
	require.Equal(t, "SELECT id \nFROM accounts \nWHERE id = $1 \nORDER BY id \nLIMIT $2 \nFOR UPDATE NOWAIT", q.String())
	q.Close()
}

func TestPostgresInNotIn(t *testing.T) {
	q := xsql.Postgres.From("tasks").Select("id").Where("status").In("new", "wip")
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE status IN ($1,$2)", q.String())
	require.Equal(t, []any{"new", "wip"}, q.Args())
	q.Close()

	q = xsql.Postgres.From("tasks").Select("id").Where("status").NotIn("done", "canceled")
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE status NOT IN ($1,$2)", q.String())
	require.Equal(t, []any{"done", "canceled"}, q.Args())
	q.Close()

	q = xsql.Postgres.From("tasks").Select("id").Where("status").In()
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE status IN (SELECT NULL WHERE 1=0)", q.String())
	require.Empty(t, q.Args())
	q.Close()

	q = xsql.Postgres.From("tasks").Select("id").Where("status").NotIn()
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE status NOT IN (SELECT NULL WHERE 1=0)", q.String())
	require.Empty(t, q.Args())
	q.Close()

	q = xsql.Postgres.From("tasks").Select("id").
		Where("tenant_id = ?", 9).
		Where("status").NotIn()
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE tenant_id = $1 AND status NOT IN (SELECT NULL WHERE 1=0)", q.String())
	require.Equal(t, []any{9}, q.Args())
	q.Close()

	q = xsql.Postgres.From("tasks").Select("id").
		Where("tenant_id = ?", 9).
		Where("status").In()
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE tenant_id = $1 AND status IN (SELECT NULL WHERE 1=0)", q.String())
	require.Equal(t, []any{9}, q.Args())
	q.Close()

	q = xsql.SQLServer.From("tasks").Select("id").Where("status").In()
	require.Equal(t, "SELECT id \nFROM tasks \nWHERE status IN (SELECT NULL WHERE 1=0)", q.String())
	q.Close()
}

func TestPostgresCrossJoin(t *testing.T) {
	q := xsql.Postgres.From("t1").Select("t1.id, t2.id").CrossJoin("t2").Where("t1.id = ?", 1)
	require.Equal(t, "SELECT t1.id, t2.id \nFROM t1 CROSS JOIN t2 \nWHERE t1.id = $1", q.String())
	q.Close()
}

func TestPostgresPaginate(t *testing.T) {
	q := xsql.Postgres.From("items").Select("id").Paginate(3, 10)
	require.Equal(t, "SELECT id \nFROM items \nLIMIT $1 \nOFFSET $2", q.String())
	require.Equal(t, []any{10, 20}, q.Args())
	q.Close()

	q = xsql.Postgres.From("items").Select("id").Paginate(1, 10)
	require.Equal(t, "SELECT id \nFROM items \nLIMIT $1", q.String())
	require.Equal(t, []any{10}, q.Args())
	q.Close()
}

func TestPostgresPlaceholderNumbering(t *testing.T) {
	q := xsql.Postgres.From("t").
		Select("a, ?", "x").
		Where("b = ?", 1).
		Where("c IN (?, ?)", 2, 3).
		Having("d > ?", 4).
		Limit(5).
		Offset(6)
	require.Equal(t, "SELECT a, $1 \nFROM t \nWHERE b = $2 AND c IN ($3, $4) \nHAVING d > $5 \nLIMIT $6 \nOFFSET $7", q.String())
	require.Equal(t, []any{"x", 1, 2, 3, 4, 5, 6}, q.Args())
	q.Close()
}

func TestPostgresEscapeQuestion(t *testing.T) {
	q := xsql.Postgres.From("t").Select("id").Where("json \\? ? AND n = ?", "key", 1)
	require.Equal(t, "SELECT id \nFROM t \nWHERE json ? $1 AND n = $2", q.String())
	require.Equal(t, []any{"key", 1}, q.Args())
	q.Close()

	q = xsql.Postgres.From("t").Select("id").Where("json \\? 'key'")
	require.Equal(t, "SELECT id \nFROM t \nWHERE json ? 'key'", q.String())
	require.Empty(t, q.Args())
	q.Close()
}

func TestClonePreservesNameAndNewLines(t *testing.T) {
	q := xsql.Postgres.WithNewLines(false).
		From("t").
		Select("id").
		SetName("clone_src")
	defer q.Close()
	require.Equal(t, "SELECT id FROM t", q.String())

	c := q.Clone()
	defer c.Close()
	require.Empty(t, c.Name())
	require.Equal(t, q.String(), c.String())

	c.Where("id = ?", 1)
	require.Equal(t, "SELECT id FROM t WHERE id = $1", c.String())
	cached, ok := xsql.Postgres.GetCachedQuery("clone_src")
	require.True(t, ok)
	require.Equal(t, "SELECT id FROM t", cached)
}

func TestBindSkipsDashTag(t *testing.T) {
	type Hidden struct {
		Secret string `db:"secret"`
	}
	var row struct {
		ID      int64  `db:"id"`
		Skipped string `db:"-"`
		Also    string `db:"-,"`
		Empty   string `db:","`
		Hidden  `db:"-"`
		Name    string `db:"name,varchar"`
	}
	q := xsql.From("users").Bind(&row)
	defer q.Close()
	require.Equal(t, "SELECT id, name \nFROM users", q.String())
	require.Equal(t, []any{&row.ID, &row.Name}, q.Dest())
}
