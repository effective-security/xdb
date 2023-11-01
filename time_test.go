package xdb_test

import (
	"testing"
	"time"

	"github.com/effective-security/xdb"
	"github.com/stretchr/testify/assert"
)

func TestTimeTruncate(t *testing.T) {
	now := xdb.Now()
	assert.Equal(t, now.UTC(), xdb.ParseTime(now.String()).UTC())

	now = xdb.Now().Add(time.Second)
	assert.Equal(t, now.UTC(), xdb.ParseTime(now.String()).UTC())

	now = xdb.FromNow(time.Second)
	assert.Equal(t, now.UTC(), xdb.ParseTime(now.String()).UTC())
}
