package xsql_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/effective-security/xdb/xsql"
	"github.com/stretchr/testify/require"
)

func TestConcurrentBuilders(t *testing.T) {
	const goroutines = 64
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			q := xsql.Postgres.From("users").
				Select("id, name").
				Where("id = ?", id).
				Where("active = ?", true).
				Limit(10)
			sql := q.String()
			args := append([]any(nil), q.Args()...)
			q.Close()
			want := "SELECT id, name \nFROM users \nWHERE id = $1 AND active = $2 \nLIMIT $3"
			if sql != want {
				errCh <- fmt.Errorf("sql: %q", sql)
				return
			}
			if len(args) != 3 || args[0] != id || args[1] != true || args[2] != 10 {
				errCh <- fmt.Errorf("args: %#v", args)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func TestConcurrentClone(t *testing.T) {
	base := xsql.Postgres.From("users").Select("id").Where("active = ?", true)
	defer base.Close()

	const goroutines = 32
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			q := base.Clone().Where("id = ?", id)
			args := append([]any(nil), q.Args()...)
			sql := q.String()
			q.Close()
			if len(args) != 2 || args[0] != true || args[1] != id {
				errCh <- fmt.Errorf("args: %#v", args)
				return
			}
			if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
				errCh <- fmt.Errorf("sql: %q", sql)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	require.Equal(t, []any{true}, base.Args())
	require.Equal(t, "SELECT id \nFROM users \nWHERE active = $1", base.String())
}

func TestConcurrentGetOrCreateQuery(t *testing.T) {
	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errCh := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sql, name := xsql.Postgres.GetOrCreateQuery("concurrent_users_by_id", func(string) xsql.Builder {
				return xsql.Postgres.From("users").Select("id, name").Where("id = ?", 0)
			})
			if name != "concurrent_users_by_id" {
				errCh <- fmt.Errorf("name: %s", name)
				return
			}
			if sql != "SELECT id, name \nFROM users \nWHERE id = $1" {
				errCh <- fmt.Errorf("sql: %s", sql)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	cached, ok := xsql.Postgres.GetCachedQuery("concurrent_users_by_id")
	require.True(t, ok)
	require.Equal(t, "SELECT id, name \nFROM users \nWHERE id = $1", cached)
}

func TestUseNewLinesDoesNotMutateDefault(t *testing.T) {
	q1 := xsql.UseNewLines(false).Select("id").From("t")
	q2 := xsql.Select("id").From("t")
	require.Equal(t, "SELECT id FROM t", q1.String())
	require.Equal(t, "SELECT id \nFROM t", q2.String())
	q1.Close()
	q2.Close()
}

func TestWithNewLinesSharesCache(t *testing.T) {
	name := "shared_cache_named_query"
	d := xsql.Postgres.WithNewLines(false)
	q := d.From("t").Select("id").SetName(name)
	sql := q.String()
	q.Close()

	cached, ok := xsql.Postgres.GetCachedQuery(name)
	require.True(t, ok)
	require.Equal(t, sql, cached)
}
