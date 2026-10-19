package xdb

import (
	"database/sql"
	"math"
	"strconv"
	"testing"
	"time"

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

	assert.PanicsWithValue(t, "position is out of range", func() {
		b.Set(64, 1)
	})
	assert.PanicsWithValue(t, "position is out of range", func() {
		b.SetEnum(64, 1)
	})
}

func TestParamsNameFollowsChanges(t *testing.T) {
	b := NewQueryParams("ListXXX")
	b.Set(1, "a")
	assert.Equal(t, "ListXXX_x2", b.Name())

	b.Set(2, "b")
	assert.Equal(t, "ListXXX_x6", b.Name())
	b.SetEnum(3, 1)
	assert.Equal(t, "ListXXX_xe_3x1", b.Name())
	b.SetFlags(5)
	assert.Equal(t, "ListXXX_xe_3x1_fx5", b.Name())
	b.SetNullColums([]string{"c"})
	withNulls := b.Name()
	assert.NotEqual(t, "ListXXX_xe_3x1_fx5", withNulls)
	b.SetPage(10, 20)
	assert.NotEqual(t, withNulls, b.Name())
	assert.Contains(t, b.Name(), "_fx5_o_n")
}

func TestParamsNullColumnsAreCopied(t *testing.T) {
	cols := []string{"c", "a", "b"}
	b := NewQueryParams("ListXXX")
	b.SetNullColums(cols)
	name := b.Name()

	assert.Equal(t, []string{"c", "a", "b"}, cols, "the caller's slice is not reordered")
	assert.Equal(t, []string{"a", "b", "c"}, b.GetNullColumns())
	assert.Equal(t, NewQueryParams("ListXXX").SetNullColums([]string{"b", "c", "a"}).Name(), name)

	b.Reset()
	assert.Empty(t, b.GetNullColumns())
	assert.Equal(t, "ListXXX_x0", b.Name())
}

func TestParamsCursor(t *testing.T) {
	at := Time(time.Date(2026, 10, 19, 12, 0, 0, 123456000, time.UTC))

	// multi-column key: created_at, id
	b := NewQueryParams("ListEvents")
	b.Set(2, uint64(7))
	b.SetCursor(10, 5, at, ID(42))
	assert.Equal(t, "ListEvents_x24_c", b.Name())
	assert.Equal(t, []any{uint64(7), at, ID(42), uint32(10)}, b.Args())
	limit, cursor := b.Cursor()
	assert.Equal(t, uint32(10), limit)
	assert.Equal(t, []any{at, ID(42)}, cursor)
	assert.Panics(t, func() {
		b.Set(3, 1)
	})

	// single-column key: the value itself
	b = NewQueryParams("ListXXX").SetCursor(0, 1, ID(42))
	limit, cursor = b.Cursor()
	assert.Equal(t, uint32(DefaultPageSize), limit)
	assert.Equal(t, ID(42), cursor)
	assert.Equal(t, []any{ID(42), uint32(DefaultPageSize)}, b.Args())

	// no key: the first page, as SetPage(limit, 0)
	b = NewQueryParams("ListXXX").SetCursor(10, 1)
	limit, cursor = b.Cursor()
	assert.Equal(t, uint32(10), limit)
	assert.Nil(t, cursor)
	assert.Equal(t, []any{uint32(10), uint32(0)}, b.Args())
	assert.Equal(t, "ListXXX_x0", b.Name())
}

func TestWithDatabase(t *testing.T) {
	tcases := []struct {
		driver, ds, db string
		exp            string
	}{
		{"postgres", "postgres://u:p@h:5432?sslmode=disable", "testdb", "postgres://u:p@h:5432?sslmode=disable&dbname=testdb"},
		{"postgres", "postgres://u:p@h:5432", "testdb", "postgres://u:p@h:5432?dbname=testdb"},
		{"sqlserver", "sqlserver://h:1433?user id=sa", "testdb", "sqlserver://h:1433?user id=sa&database=testdb"},
		{"sqlserver", "sqlserver://h:1433", "my db", "sqlserver://h:1433?database=my+db"},
		{"postgres", "postgres://u:p@h:5432", "", "postgres://u:p@h:5432"},
	}
	for _, tc := range tcases {
		ds, err := withDatabase(tc.driver, tc.ds, tc.db)
		assert.NoError(t, err)
		assert.Equal(t, tc.exp, ds)
	}

	_, err := withDatabase("mysql", "mysql://h", "testdb")
	assert.EqualError(t, err, `unsupported driver "mysql"`)
}

func TestParamsWithNuls(t *testing.T) {
	b := NewQueryParams("ListXXX")
	b.Reset()
	nulls := []string{"a", "b", "c"}
	b.SetNullColums(nulls)

	assert.Equal(t, "ListXXX_x0_nb4963f3f3fad7867", b.Name())
	assert.Empty(t, b.Args())
	assert.Equal(t, nulls, b.GetNullColumns())
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

func TestPeekLimit(t *testing.T) {
	args := []any{"org", uint32(10), uint32(0)}
	peek, ok := peekLimit(args, 1, 10)
	assert.True(t, ok)
	assert.Equal(t, []any{"org", uint32(11), uint32(0)}, peek)
	assert.Equal(t, uint32(10), args[1], "the arguments are copied")

	peek, ok = peekLimit([]any{5, 0}, 0, 5)
	assert.True(t, ok)
	assert.Equal(t, []any{6, 0}, peek)

	peek, ok = peekLimit([]any{sql.Named("limit", 5)}, 0, 5)
	assert.True(t, ok)
	assert.Equal(t, []any{sql.Named("limit", 6)}, peek)

	// an int limit at math.MaxInt32 is math.MaxInt on 32-bit targets, where
	// the increment would wrap to a negative limit
	_, ok = peekLimit([]any{math.MaxInt32}, 0, math.MaxInt32)
	assert.Equal(t, strconv.IntSize == 64, ok)

	for _, tc := range []struct {
		args  []any
		at    int
		limit uint32
	}{
		{[]any{uint32(10)}, -1, 10},                        // no limit argument
		{[]any{uint32(10)}, 1, 10},                         // out of range
		{[]any{"org", uint32(10)}, 0, 10},                  // not a limit
		{[]any{uint32(7)}, 0, 10},                          // another value
		{[]any{uint64(10)}, 0, 10},                         // a type PageParam does not read
		{[]any{uint32(math.MaxUint32)}, 0, math.MaxUint32}, // would wrap
	} {
		_, ok := peekLimit(tc.args, tc.at, tc.limit)
		assert.False(t, ok, "%v at %d", tc.args, tc.at)
	}
}
