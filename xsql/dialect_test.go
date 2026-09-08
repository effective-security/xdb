package xsql

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWritePg(t *testing.T) {
	cases := []struct {
		in   string
		from int
		out  string
		next int
	}{
		{in: "id = ?", from: 1, out: "id = $1", next: 2},
		{in: "? + ?", from: 3, out: "$3 + $4", next: 5},
		{in: "json \\? ?", from: 1, out: "json ? $1", next: 2},
		{in: "no placeholders", from: 1, out: "no placeholders", next: 1},
		{in: "?", from: 1, out: "$1", next: 2},
		{in: "\\?", from: 1, out: "?", next: 1},
		{in: "a = ? AND b \\? ? AND c = ?", from: 1, out: "a = $1 AND b ? $2 AND c = $3", next: 4},
	}
	for _, tc := range cases {
		var buf strings.Builder
		next := writePg(tc.from, []byte(tc.in), &buf)
		require.Equal(t, tc.out, buf.String(), tc.in)
		require.Equal(t, tc.next, next, tc.in)
	}
}

func TestDialectProvider(t *testing.T) {
	require.Equal(t, "postgres", Postgres.Provider())
	require.Equal(t, "default", NoDialect.Provider())
	require.Equal(t, "sqlserver", SQLServer.Provider())
}

func TestClearCache(t *testing.T) {
	d := newDialect("postgres", true)
	d.PutCachedQuery("k", "SELECT 1")
	sql, ok := d.GetCachedQuery("k")
	require.True(t, ok)
	require.Equal(t, "SELECT 1", sql)

	view := d.WithNewLines(false).(*Dialect)
	d.ClearCache()
	_, ok = d.GetCachedQuery("k")
	require.False(t, ok)
	_, ok = view.GetCachedQuery("k")
	require.False(t, ok)

	view.PutCachedQuery("k2", "SELECT 2")
	sql, ok = d.GetCachedQuery("k2")
	require.True(t, ok)
	require.Equal(t, "SELECT 2", sql)
}

func TestCacheConcurrentPut(t *testing.T) {
	d := newDialect("postgres", true)
	d.maxCache = 64
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "k"
			d.PutCachedQuery(key, "SQL")
			d.PutCachedQuery(fmt.Sprintf("n%d", i), "SQL")
		}(i)
	}
	wg.Wait()

	sql, ok := d.GetCachedQuery("k")
	require.True(t, ok)
	require.Equal(t, "SQL", sql)
	d.cacheMu.Lock()
	n := len(d.cache)
	d.cacheMu.Unlock()
	require.LessOrEqual(t, n, 64)
	require.GreaterOrEqual(t, n, 2)
}

func TestCacheBound(t *testing.T) {
	d := newDialect("postgres", true)
	d.maxCache = 2
	d.PutCachedQuery("a", "A")
	d.PutCachedQuery("b", "B")
	d.PutCachedQuery("c", "C")

	_, okA := d.GetCachedQuery("a")
	_, okB := d.GetCachedQuery("b")
	_, okC := d.GetCachedQuery("c")
	require.True(t, okA)
	require.True(t, okB)
	require.False(t, okC)

	d.PutCachedQuery("a", "A2")
	sql, ok := d.GetCachedQuery("a")
	require.True(t, ok)
	require.Equal(t, "A2", sql)
}
