package xsql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSQLCache(t *testing.T) {
	buf := getBuffer()
	defer putBuffer(buf)

	buf.WriteString("test")
	dialect := defaultDialect.Load().(*Dialect)

	_, ok := dialect.getCachedSQL(buf)
	require.False(t, ok)

	dialect.putCachedSQL(buf, "test SQL")
	sql, ok := dialect.getCachedSQL(buf)
	require.True(t, ok)
	require.Equal(t, "test SQL", sql)

	dialect.ClearCache()
}
