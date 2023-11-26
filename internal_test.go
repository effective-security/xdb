package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestID(t *testing.T) {
	assert.Panics(t, func() { MustID("abd") })
	assert.Panics(t, func() { MustID("") })

	id := MustID("123456789")
	assert.Equal(t, uint64(123456789), id.UInt64())
	assert.False(t, id.IsZero())
	assert.False(t, id.Invalid())
	assert.True(t, id.Valid())

	id2 := NewID(123412341)
	assert.Empty(t, id2.id.str)
	assert.Equal(t, "123412341", id2.String())
	assert.Equal(t, "123412341", id2.id.str)
	assert.Equal(t, "123412341", id2.String())

	id3 := NewID(123412341)
	assert.NotEqual(t, id2, id3)
	assert.Equal(t, id2.String(), id3.String())
	assert.Equal(t, id2, id3)
}
