package schema_test

import (
	"sync"
	"testing"

	"github.com/effective-security/xdb/schema"
	"github.com/effective-security/xdb/xsql"
	"github.com/stretchr/testify/assert"
)

// TestTableInfoConcurrent checks that a generated table, a package-level
// value in clients, is safe to use from many goroutines.
// Run with `go test -race` to exercise the check.
func TestTableInfoConcurrent(t *testing.T) {
	ti := &schema.TableInfo{
		Schema:     "public",
		Name:       "org",
		SchemaName: "public.org",
		Columns:    []string{"id", "name"},
		Dialect:    xsql.Postgres,
	}

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			assert.Equal(t, "id, name", ti.AllColumns())
			q := ti.Select()
			assert.Contains(t, q.String(), "SELECT id, name")
			q.Close()
		})
	}
	wg.Wait()
}
