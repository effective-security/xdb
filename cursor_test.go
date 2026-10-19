package xdb_test

import (
	"encoding/base64"
	"errors"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/effective-security/xdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnum int32

func TestCursorRoundTrip(t *testing.T) {
	// microseconds, as PostgreSQL stores a timestamptz
	at := xdb.Time(time.Date(2026, 10, 19, 12, 0, 0, 123456000, time.UTC))
	tm := time.Date(2026, 1, 2, 3, 4, 5, 999999999, time.FixedZone("x", 3600))
	addr := netip.MustParseAddr("10.0.0.1")

	c := newCursor(t, at, xdb.ID(1234567890123456789), uint64(18446744073709551615),
		int64(-5), "a,b\"c", true, 1.5, xdb.NULLString("n"), testEnum(3), tm, addr)
	require.Equal(t, 11, c.Len())

	s := c.String()
	assert.NotContains(t, s, "=", "the token is URL-safe and unpadded")

	parsed, err := xdb.ParseCursor(s)
	require.NoError(t, err)

	var (
		gotAt   xdb.Time
		gotID   xdb.ID
		gotU64  uint64
		gotI64  int64
		gotStr  string
		gotBool bool
		gotF    float64
		gotNS   xdb.NULLString
		gotEnum testEnum
		gotTm   time.Time
		gotAddr netip.Addr
	)
	require.NoError(t, parsed.Scan(&gotAt, &gotID, &gotU64, &gotI64, &gotStr, &gotBool, &gotF, &gotNS, &gotEnum, &gotTm, &gotAddr))
	assert.Equal(t, at, gotAt)
	assert.Equal(t, 123456000, time.Time(gotAt).Nanosecond(), "full precision")
	assert.Equal(t, xdb.ID(1234567890123456789), gotID)
	assert.Equal(t, uint64(18446744073709551615), gotU64)
	assert.Equal(t, int64(-5), gotI64)
	assert.Equal(t, "a,b\"c", gotStr)
	assert.True(t, gotBool)
	assert.Equal(t, 1.5, gotF)
	assert.Equal(t, xdb.NULLString("n"), gotNS)
	assert.Equal(t, testEnum(3), gotEnum)
	assert.True(t, tm.Equal(gotTm))
	assert.Equal(t, time.UTC, gotTm.Location())
	assert.Equal(t, addr, gotAddr)
}

func TestCursorZero(t *testing.T) {
	var c xdb.Cursor
	assert.True(t, c.IsZero())
	assert.Empty(t, c.String())
	assert.True(t, newCursor(t).IsZero())

	c, err := xdb.ParseCursor("")
	require.NoError(t, err)
	assert.True(t, c.IsZero())
	assert.NoError(t, c.Scan())
}

func TestCursorErrors(t *testing.T) {
	encode := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

	for _, s := range []string{"not base64!", encode("{}"), encode("[]"), encode("[1,2]"), encode("x")} {
		_, err := xdb.ParseCursor(s)
		assert.Error(t, err, s)
	}

	c := newCursor(t, "abc", xdb.ID(1))
	var id xdb.ID
	var n int64
	assert.EqualError(t, c.Scan(&id), "invalid cursor: 2 values, expected 1")
	err := c.Scan(&n, &id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor value 0")
	assert.Error(t, c.Scan(id, &id), "not a pointer")
	assert.Error(t, c.Scan(&struct{}{}, &id), "unsupported type")

	var at xdb.Time
	assert.Error(t, newCursor(t, "yesterday").Scan(&at))
}

// textKey is an integer-backed key with custom text methods on its pointer
type textKey int64

func (k *textKey) MarshalText() ([]byte, error) {
	return []byte("id:" + strconv.FormatInt(int64(*k), 10)), nil
}

func (k *textKey) UnmarshalText(b []byte) error {
	n, err := strconv.ParseInt(strings.TrimPrefix(string(b), "id:"), 10, 64)
	*k = textKey(n)
	return err
}

func TestCursorTextMethods(t *testing.T) {
	key := textKey(42)
	n := int64(7)
	at := xdb.Time(time.Date(2026, 10, 19, 12, 0, 0, 1000, time.UTC))

	// text methods win over the integer kind, by value or by pointer
	for _, c := range []xdb.Cursor{newCursor(t, key, &n, &at), newCursor(t, &key, n, at)} {
		parsed, err := xdb.ParseCursor(c.String())
		require.NoError(t, err)

		var gotKey textKey
		var gotN int64
		var gotAt xdb.Time
		require.NoError(t, parsed.Scan(&gotKey, &gotN, &gotAt))
		assert.Equal(t, textKey(42), gotKey)
		assert.Equal(t, int64(7), gotN)
		assert.Equal(t, at, gotAt)
	}

	var nilKey *textKey
	assert.Equal(t, 1, newCursor(t, nilKey).Len(), "a nil pointer is an empty value")
}

func TestCursorNilDestinations(t *testing.T) {
	c := newCursor(t, xdb.Time(time.Now()))
	assert.NotPanics(t, func() {
		assert.Error(t, c.Scan((*xdb.Time)(nil)))
		assert.Error(t, c.Scan((*time.Time)(nil)))
		assert.Error(t, c.Scan((*textKey)(nil)))
		assert.Error(t, c.Scan(nil))
	})
}

func newCursor(t *testing.T, key ...any) xdb.Cursor {
	t.Helper()
	c, err := xdb.NewCursor(key...)
	require.NoError(t, err)
	return c
}

// failingKey has a text format its MarshalText cannot always produce
type failingKey int64

func (k failingKey) MarshalText() ([]byte, error) {
	if k < 0 {
		return nil, errors.New("negative key")
	}
	return []byte(strconv.FormatInt(int64(k), 10)), nil
}

func (k *failingKey) UnmarshalText(b []byte) error {
	n, err := strconv.ParseInt(string(b), 10, 64)
	*k = failingKey(n)
	return err
}

func TestNewCursorErrors(t *testing.T) {
	// no fallback token that Scan could not read back
	_, err := xdb.NewCursor(xdb.ID(1), failingKey(-1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor value 1: negative key")

	_, err = xdb.NewCursor(struct{ A int }{A: 1})
	assert.EqualError(t, err, "invalid cursor value 0: unsupported type: struct { A int }")

	c := newCursor(t, failingKey(5))
	var got failingKey
	require.NoError(t, c.Scan(&got))
	assert.Equal(t, failingKey(5), got)
}
