package xsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParams(t *testing.T) {
	b := NewQueryParams("ListXXX")

	b.Reset()
	b.Set(0, 1)
	b.Set(1, "a")
	b.Set(2, true)
	b.SetEnum(34, 0x8)
	b.SetEnum(61, 0x4)
	// Limit, Offset
	b.AddArgs(1000, 1)
	b.SetFlags(0x16, 0x4)

	expArgs := []any{1, "a", true, 1000, 1}

	assert.Equal(t, "ListXXX_x2000000400000007_34x8_61x4_fx16_fx4", b.Name())
	assert.Equal(t, expArgs, b.Args())
	assert.True(t, b.IsSet(0))
	assert.True(t, b.IsSet(1))
	assert.True(t, b.IsSet(2))
	assert.False(t, b.IsSet(3))
	assert.True(t, b.IsSet(34))
	assert.True(t, b.IsSet(61))
	assert.Equal(t, []int32{0x16, 0x4}, b.GetFlags())

	e, ok := b.GetEnum(34)
	assert.True(t, ok)
	assert.Equal(t, int32(0x8), e)

	e, ok = b.GetEnum(5)
	assert.False(t, ok)
	assert.Equal(t, int32(0), e)

	assert.Equal(t, "ListXXX_x2000000400000007_34x8_61x4_fx16_fx4", b.Name())
	assert.Equal(t, expArgs, b.Args())

	assert.Panics(t, func() {
		b.Set(64, 1)
	})
	assert.Panics(t, func() {
		b.SetEnum(64, 1)
	})
}

type testQueryParams struct {
	Pos1 int
}

func (t *testQueryParams) QueryParams() QueryParams {
	return NewQueryParams("test")
}

func TestGetQueryParams(t *testing.T) {
	var b QueryParamsBuilder
	assert.NotPanics(t, func() {
		_ = GetQueryParams(&b)
	})
	assert.NotPanics(t, func() {
		_ = GetQueryParams(&testQueryParams{})
	})
	assert.Panics(t, func() {
		_ = GetQueryParams("test")
	})
}
