package xdb_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/schema"
	"github.com/effective-security/xlog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableInfo(t *testing.T) {
	nulls := map[string]bool{
		"meta": true,
	}
	ti := schema.TableInfo{
		Schema:  "public",
		Columns: []string{"id", "meta", "name"},
	}
	assert.Equal(t, "id,meta,name", ti.AllColumns())
	assert.Equal(t, "a.id,NULL,a.name", ti.AliasedColumns("a", nulls))
}

func TestNullTime(t *testing.T) {
	v := xdb.NullTime(nil)
	require.NotNil(t, v)
	assert.False(t, v.Valid)

	i := time.Now()
	v = xdb.NullTime(&i)
	require.NotNil(t, v)
	assert.True(t, v.Valid)
	assert.Equal(t, i, v.Time)
}

func TestString(t *testing.T) {
	v := xdb.String(nil)
	assert.Empty(t, v)

	s := "1234"
	v = xdb.String(&s)
	assert.Equal(t, s, v)
}

func TestParseID(t *testing.T) {
	_, err := xdb.ParseUint("")
	require.Error(t, err)

	_, err = xdb.ParseUint("@123")
	require.Error(t, err)

	v, err := xdb.ParseUint("1234567")
	require.NoError(t, err)
	assert.Equal(t, uint64(1234567), v)

	_, err = xdb.ParseID("")
	assert.EqualError(t, err, "bad_request: invalid ID")
	_, err = xdb.ParseID("@123")
	assert.EqualError(t, err, "bad_request: invalid ID")

	id := xdb.TryParseID("")
	assert.Equal(t, uint64(0), id.UInt64())

	id = xdb.TryParseID("@123")
	assert.Equal(t, uint64(0), id.UInt64())

	id = xdb.TryParseID("1234567")
	assert.Equal(t, uint64(1234567), id.UInt64())
	id, err = xdb.ParseID("1234567")
	require.NoError(t, err)
	assert.Equal(t, uint64(1234567), id.UInt64())
}

func TestIDString(t *testing.T) {
	assert.Equal(t, "0", xdb.IDString(0))
	assert.Equal(t, "999", xdb.IDString(999))
}

func TestIsNotFoundError(t *testing.T) {
	assert.True(t, xdb.IsNotFoundError(sql.ErrNoRows))
	assert.True(t, xdb.IsNotFoundError(errors.WithMessage(errors.New("sql: no rows in result set"), "failed")))
}

type validator struct {
	valid bool
}

func (t validator) Validate() error {
	if !t.valid {
		return errors.New("invalid")
	}
	return nil
}

func TestValidate(t *testing.T) {
	assert.Error(t, xdb.Validate(validator{false}))
	assert.NoError(t, xdb.Validate(validator{true}))
	assert.NoError(t, xdb.Validate(nil))
}

func TestTimePtr(t *testing.T) {
	var zero xdb.Time
	assert.Nil(t, xdb.TimePtr(zero))
	assert.NotNil(t, xdb.TimePtr(xdb.Time(time.Now())))
}

func TestStrings(t *testing.T) {
	tcases := []struct {
		val []string
		exp string
	}{
		{val: []string{"one", "two"}, exp: "[\"one\",\"two\"]"},
		{val: []string{}, exp: "[]"},
		{val: nil, exp: ""},
	}

	for _, tc := range tcases {
		val := xdb.Strings(tc.val)
		dr, err := val.Value()
		require.NoError(t, err)

		var drv string
		if v, ok := dr.(string); ok {
			drv = v
		}
		assert.Equal(t, tc.exp, drv)

		var val2 xdb.Strings
		err = val2.Scan(dr)
		require.NoError(t, err)
		assert.EqualValues(t, val, val2)
	}
}

func TestMetadata(t *testing.T) {
	tcases := []struct {
		val xdb.Metadata
		exp string
	}{
		{val: xdb.Metadata{"one": "two"}, exp: "{\"one\":\"two\"}"},
		{val: xdb.Metadata{}, exp: ""},
		{val: nil, exp: ""},
	}

	for _, tc := range tcases {
		dr, err := tc.val.Value()
		require.NoError(t, err)

		var drv string
		if v, ok := dr.(string); ok {
			drv = v
		}
		assert.Equal(t, tc.exp, drv)

		var val2 xdb.Metadata
		err = val2.Scan(dr)
		require.NoError(t, err)
		assert.Equal(t, len(tc.val), len(val2))
	}
}

func TestDbTime(t *testing.T) {
	nb, err := time.Parse(time.RFC3339, "2022-04-01T16:11:15.123Z")
	require.NoError(t, err)

	nbl := nb.Local()

	tcases := []struct {
		val    xdb.Time
		exp    time.Time
		isZero bool
		str    string
	}{
		{val: xdb.Time{}, exp: time.Time{}, isZero: true, str: ""},
		{val: xdb.Time(nb), exp: nb, isZero: false, str: "2022-04-01T16:11:15.123Z"},
		{val: xdb.Time(nbl), exp: nb, isZero: false, str: "2022-04-01T16:11:15.123Z"},
	}

	for _, tc := range tcases {
		dr, err := tc.val.Value()
		require.NoError(t, err)

		var drv time.Time
		if v, ok := dr.(time.Time); ok {
			drv = v
		}
		assert.Equal(t, tc.exp, drv)

		if tc.isZero {
			assert.True(t, tc.val.IsZero())
			assert.Nil(t, tc.val.Ptr())
		} else {
			assert.False(t, tc.val.IsZero())
			assert.NotNil(t, tc.val.Ptr())
		}
		assert.Equal(t, tc.str, tc.val.String())
		assert.Equal(t, tc.val.IsZero(), tc.val.IsNil())

		var val2 xdb.Time
		err = val2.Scan(dr)
		require.NoError(t, err)
		assert.EqualValues(t, tc.val.UTC(), val2)
	}

	now := time.Now()
	xnow := xdb.Now()
	xafter := xdb.FromNow(time.Hour)
	assert.Equal(t, xnow.UTC().Unix(), now.Unix())

	now = now.Add(time.Hour)
	now2 := xnow.Add(time.Hour)
	assert.Equal(t, now.Unix(), now2.UTC().Unix())
	assert.Equal(t, xafter.UTC().Unix(), now2.UTC().Unix())

	ms := xnow.UnixMilli()
	assert.Equal(t, xnow.UTC().Truncate(time.Millisecond), xdb.FromUnixMilli(ms).UTC())
}

func TestDbTimeParse(t *testing.T) {
	withNano := xdb.ParseTime("2022-11-21T08:39:23.439786Z")
	assert.False(t, withNano.IsZero())
	assert.Equal(t, "2022-11-21T08:39:23.439Z", withNano.String())
}

func TestDbTimeEncode(t *testing.T) {
	nb, err := time.Parse(time.RFC3339, "2022-04-01T16:11:15Z")
	require.NoError(t, err)
	xct := xdb.Time(nb)

	assert.Equal(t, `"2022-04-01T16:11:15Z"`, xlog.EscapedString(xct))
	assert.Equal(t, `""`, xlog.EscapedString(xdb.Time{}))

	b, err := json.Marshal(xct)
	require.NoError(t, err)
	var xnow2 xdb.Time
	require.NoError(t, json.Unmarshal(b, &xnow2))
	assert.Equal(t, xct, xnow2)

	b, err = json.Marshal(xdb.Time{})
	assert.NoError(t, err)
	assert.Equal(t, `""`, string(b))

	foo := struct {
		CreatedAt xdb.Time `json:"created_at,omitempty"`
		UpdatedAt xdb.Time `json:"updated_at,omitempty"`
	}{
		CreatedAt: xct,
	}
	b, err = json.Marshal(foo)
	require.NoError(t, err)
	assert.Equal(t, `{"created_at":"2022-04-01T16:11:15Z","updated_at":""}`, string(b))

	require.NoError(t, json.Unmarshal(b, &foo))
}

func TestNULLString(t *testing.T) {
	tcases := []struct {
		val xdb.NULLString
		exp string
	}{
		{val: "one", exp: "one"},
		{val: "", exp: ""},
	}

	for _, tc := range tcases {
		val := tc.val
		dr, err := val.Value()
		require.NoError(t, err)

		var drv string
		if v, ok := dr.(string); ok {
			drv = v
		}
		assert.Equal(t, tc.exp, drv)

		var val2 xdb.NULLString
		err = val2.Scan(dr)
		require.NoError(t, err)
		assert.EqualValues(t, val, val2)
	}
}

func TestID32Value(t *testing.T) {
	tcases := []struct {
		in  xdb.ID32
		exp any
	}{
		{in: xdb.ID32(1), exp: int64(1)},
		{in: xdb.ID32(0), exp: nil},
	}

	for _, tc := range tcases {
		dr, err := tc.in.Value()
		require.NoError(t, err)
		assert.Equal(t, tc.exp, dr)
	}
}

func TestID32Scan(t *testing.T) {
	tcases := []struct {
		exp xdb.ID32
		val any
	}{
		{val: int64(1), exp: xdb.ID32(1)},
		{val: int64(0), exp: xdb.ID32(0)},
		{val: nil, exp: xdb.ID32(0)},
	}

	for _, tc := range tcases {
		var val2 xdb.ID32
		err := val2.Scan(tc.val)
		require.NoError(t, err)
		assert.EqualValues(t, tc.exp, val2)
	}
}

func TestInt64Value(t *testing.T) {
	tcases := []struct {
		in  xdb.Int64
		exp any
	}{
		{in: xdb.Int64(1), exp: int64(1)},
		{in: xdb.Int64(0), exp: nil},
	}

	for _, tc := range tcases {
		dr, err := tc.in.Value()
		require.NoError(t, err)
		assert.Equal(t, tc.exp, dr)
	}
}

func TestInt64Scan(t *testing.T) {
	tcases := []struct {
		exp xdb.Int64
		val any
	}{
		{val: int64(1), exp: xdb.Int64(1)},
		{val: int64(0), exp: xdb.Int64(0)},
		{val: nil, exp: xdb.Int64(0)},
	}

	for _, tc := range tcases {
		var val2 xdb.Int64
		err := val2.Scan(tc.val)
		require.NoError(t, err)
		assert.EqualValues(t, tc.exp, val2)
	}
}

func TestInt32Value(t *testing.T) {
	tcases := []struct {
		in  xdb.Int32
		exp any
	}{
		{in: xdb.Int32(1), exp: int64(1)},
		{in: xdb.Int32(0), exp: nil},
	}

	for _, tc := range tcases {
		dr, err := tc.in.Value()
		require.NoError(t, err)
		assert.Equal(t, tc.exp, dr)
	}
}

func TestInt32Scan(t *testing.T) {
	tcases := []struct {
		exp xdb.Int32
		val any
	}{
		{val: int64(1), exp: xdb.Int32(1)},
		{val: int64(0), exp: xdb.Int32(0)},
		{val: nil, exp: xdb.Int32(0)},
	}

	for _, tc := range tcases {
		var val2 xdb.Int32
		err := val2.Scan(tc.val)
		require.NoError(t, err)
		assert.EqualValues(t, tc.exp, val2)
	}
}

func TestFloatValue(t *testing.T) {
	tcases := []struct {
		in  xdb.Float
		exp any
	}{
		{in: xdb.Float(1.2345), exp: float64(1.2345)},
		{in: xdb.Float(0), exp: nil},
	}

	for _, tc := range tcases {
		dr, err := tc.in.Value()
		require.NoError(t, err)
		assert.Equal(t, tc.exp, dr)
	}
}

func TestFloatScan(t *testing.T) {
	tcases := []struct {
		exp xdb.Float
		val any
	}{
		{val: float64(1.234), exp: xdb.Float(1.234)},
		{val: float64(0), exp: xdb.Float(0)},
		{val: nil, exp: xdb.Float(0)},
	}

	for _, tc := range tcases {
		var val2 xdb.Float
		err := val2.Scan(tc.val)
		require.NoError(t, err)
		assert.EqualValues(t, tc.exp, val2)
	}
}

func TestBoolValue(t *testing.T) {
	tcases := []struct {
		in  xdb.Bool
		exp any
	}{
		{in: xdb.Bool(true), exp: true},
		{in: xdb.Bool(false), exp: nil},
	}

	for _, tc := range tcases {
		dr, err := tc.in.Value()
		require.NoError(t, err)
		assert.Equal(t, tc.exp, dr)
	}
}

func TestBoolScan(t *testing.T) {
	tcases := []struct {
		exp xdb.Bool
		val any
	}{
		{val: true, exp: xdb.Bool(true)},
		{val: false, exp: xdb.Bool(false)},
		{val: nil, exp: xdb.Bool(false)},
	}

	for _, tc := range tcases {
		var val2 xdb.Bool
		err := val2.Scan(tc.val)
		require.NoError(t, err)
		assert.EqualValues(t, tc.exp, val2)
	}
}

type withNulls struct {
	ID      xdb.ID
	Sid     xdb.ID32
	Name    xdb.NULLString
	Price   xdb.Float
	Type    xdb.Int32
	IsOwner xdb.Bool
}

func TestMarshal(t *testing.T) {
	wn := withNulls{
		ID:      xdb.NewID(12345453),
		Sid:     xdb.ID32(1234),
		Name:    xdb.NULLString("test"),
		Price:   0.123132,
		Type:    123233,
		IsOwner: true,
	}
	assert.True(t, wn.ID.Valid())

	js, err := json.Marshal(wn)
	require.NoError(t, err)
	assert.Equal(t, `{"ID":12345453,"Sid":1234,"Name":"test","Price":0.123132,"Type":123233,"IsOwner":true}`, string(js))

	var wn2 withNulls
	err = json.Unmarshal(js, &wn2)
	require.NoError(t, err)

	assert.Equal(t, wn.ID.String(), wn2.ID.String())
	assert.Equal(t, wn, wn2)

	var wn3 withNulls
	js, err = json.Marshal(wn3)
	require.NoError(t, err)
	assert.Equal(t, `{"ID":0,"Sid":0,"Name":"","Price":0.000000,"Type":0,"IsOwner":false}`, string(js))

	assert.False(t, wn3.ID.Valid())

	assert.True(t, wn3.ID.Invalid())
	assert.True(t, wn3.ID.IsZero())
}
