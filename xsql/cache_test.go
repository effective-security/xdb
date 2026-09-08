package xsql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLCache(t *testing.T) {
	dialect := newDialect("default", true)

	buf := getBuffer()
	buf.WriteString("test")

	key := bufToString(buf)
	_, ok := dialect.GetCachedQuery(key)
	require.False(t, ok)

	dialect.PutCachedQuery(key, "test SQL")
	sql, ok := dialect.GetCachedQuery(key)
	require.True(t, ok)
	require.Equal(t, "test SQL", sql)

	putBuffer(buf)

	buf2 := getBuffer()
	buf2.WriteString("test2")
	key = bufToString(buf2)
	_, ok = dialect.GetCachedQuery(key)
	require.False(t, ok)

	dialect.PutCachedQuery(key, "test SQL2")
	sql, ok = dialect.GetCachedQuery(key)
	require.True(t, ok)
	require.Equal(t, "test SQL2", sql)

	putBuffer(buf2)

	exp := "SELECT * \nFROM table"
	q, name := dialect.GetOrCreateQuery("test3", func(string) Builder {
		return dialect.From("table").Select("*")
	})
	assert.Equal(t, exp, q)
	assert.Equal(t, "test3", name)

	dialect.cacheMu.Lock()
	count := len(dialect.cache)
	dialect.cacheMu.Unlock()
	// two explicit puts + GetOrCreateQuery stores both the name and the buffer key
	assert.Equal(t, 4, count)

	// GetOrCreateQuery must close the builder so a later statement can reuse the pool slot.
	qReuse := dialect.From("other").Select("id")
	assert.Equal(t, "SELECT id \nFROM other", qReuse.String())
	qReuse.Close()
}

func TestReusePool(t *testing.T) {
	q := From("table").Select("id").Where("id > ?", 42).Where("id < ?", 1000)
	sql, args := q.String(), q.Args()
	assert.Equal(t, "SELECT id \nFROM table \nWHERE id > ? AND id < ?", sql)
	assert.Equal(t, []any{42, 1000}, args)
	q.Close()

	q2 := From("table").Select("id, ?", "NULL").Where("id > ?", 42).Where("id < ?", 1000)
	sql, args = q2.String(), q2.Args()
	assert.Equal(t, "SELECT id, ? \nFROM table \nWHERE id > ? AND id < ?", sql)
	assert.Equal(t, []any{"NULL", 42, 1000}, args)
	q2.Close()

	q2 = Postgres.From("table").Select("id, ?", "NULL").Where("id > ?", 42).Where("id < ?", 1000)
	sql, args = q2.String(), q2.Args()
	assert.Equal(t, "SELECT id, $1 \nFROM table \nWHERE id > $2 AND id < $3", sql)
	assert.Equal(t, []any{"NULL", 42, 1000}, args)
	q2.Close()

	q3 := Select("id").Where("id > ?", 42).From("table").Where("id < ?", 1000)
	sql, args = q3.String(), q3.Args()
	assert.Equal(t, "SELECT id \nFROM table \nWHERE id > ? AND id < ?", sql)
	assert.Equal(t, []any{42, 1000}, args)
	q3.Close()

	var row struct {
		ID       int64 `db:"id"`
		Quantity int64 `db:"quantity"`
	}
	q4 := With("t",
		From("orders").
			Select("id, quantity").
			Where("ts < ?", time.Now())).
		From("t").
		Bind(&row)

	assert.Equal(t, "WITH t AS (SELECT id, quantity \nFROM orders \nWHERE ts < ?) \nSELECT id, quantity \nFROM t", q4.String())
	q4.Close()
}
