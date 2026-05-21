package flake_test

import (
	"testing"
	"time"

	"github.com/effective-security/xdb/pkg/flake"
	"github.com/stretchr/testify/assert"
)

func TestFlake(t *testing.T) {
	curID := flake.DefaultIDGenerator.NextID()
	assert.Less(t, curID, flake.MaxValue)

	defer func() {
		flake.NowFunc = time.Now
	}()

	flake.NowFunc = func() time.Time {
		return time.Date(2090, 9, 7, 0, 0, 0, 0, time.UTC)
	}

	largeID := flake.DefaultIDGenerator.NextID()
	assert.Less(t, largeID, flake.MaxValue)

	tm := flake.IDTime(flake.DefaultIDGenerator, largeID)
	assert.Equal(t, tm.Year(), 2090)
}
