package xdb_test

import (
	"database/sql"
	"testing"
	"testing/synctest"
	"time"

	"github.com/effective-security/xdb"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// TestProviderCloseStopsKeepAlive fails when a goroutine of the provider is
// still blocked after Close: synctest.Test waits for every goroutine of the
// bubble to exit.
func TestProviderCloseStopsKeepAlive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)

		p, err := xdb.New("sqlite3", db, nil)
		require.NoError(t, err)

		// two keep-alive pings on the fake clock
		time.Sleep(2*time.Minute + time.Second)
		synctest.Wait()

		require.NoError(t, p.Close())
		require.NoError(t, p.Close(), "Close is idempotent")
	})
}
