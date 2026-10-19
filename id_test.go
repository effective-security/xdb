package xdb_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/effective-security/xdb"
	"github.com/effective-security/xlog"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	assert.Panics(t, func() { xdb.MustID("abd") })
	assert.Panics(t, func() { xdb.MustID("") })

	var id xdb.ID
	assert.Empty(t, id.String())
	assert.Equal(t, uint64(0), id.UInt64())

	id = xdb.MustID("123456789")
	assert.Equal(t, uint64(123456789), id.UInt64())
	assert.Equal(t, "123456789", id.String())
	assert.False(t, id.IsZero())
	assert.False(t, id.Invalid())
	assert.True(t, id.Valid())

	id.Reset()
	assert.True(t, id.IsZero())

	tcases := []struct {
		val xdb.ID
		exp int64
	}{
		{val: xdb.MustID("123456789"), exp: int64(123456789)},
	}

	for _, tc := range tcases {
		dr, err := tc.val.Value()
		require.NoError(t, err)

		v, ok := dr.(int64)
		assert.True(t, ok, "type: %t", dr)
		assert.Equal(t, tc.exp, v)

		var val2 xdb.ID
		err = val2.Scan(dr)
		require.NoError(t, err)
		assert.Equal(t, tc.val.String(), val2.String())
	}

	dr, err := xdb.ID{}.Value()
	require.Nil(t, dr)
	require.NoError(t, err)
}

func TestIDs(t *testing.T) {
	var ids xdb.IDArray
	ids = ids.Add(xdb.NewID(4))
	ids = ids.Add(xdb.NewID(1))
	ids = ids.Add(xdb.NewID(2))
	ids = ids.Add(xdb.NewID(3))
	ids = ids.Add(xdb.NewID(1))
	l := []uint64{4, 1, 2, 3}
	assert.Equal(t, l, ids.List())
	al := xdb.NewIDArray(l)
	assert.Equal(t, l, al.List())

	assert.Equal(t, l, al.Concat(al).List())

	var ids2 xdb.IDArray
	ids2 = ids2.Add(xdb.NewID(9))
	ids2 = ids2.Add(xdb.NewID(8))
	ids2 = ids2.Add(xdb.NewID(5))
	ids2 = ids2.Add(xdb.NewID(6))
	ids2 = ids2.Add(xdb.NewID(7))
	ids2 = ids2.Add(xdb.NewID(5))
	assert.Equal(t, "9,8,5,6,7", ids2.String())
	assert.True(t, ids2.Contains(xdb.NewID(5)))
	assert.False(t, ids2.Contains(xdb.NewID(55)))

	assert.False(t, ids2.Overlaps(ids))
	assert.True(t, ids.Overlaps(al))

	assert.Equal(t, []uint64{4, 1, 2, 3, 9, 8, 5, 6, 7}, al.Concat(ids2).List())
	assert.Equal(t, "1,2,3,4,5,6,7,8,9", al.Concat(ids2).Sort().String())

	assert.Equal(t, pq.Int64Array{9, 8, 5, 6, 7}, ids2.Int64Array())
	assert.Equal(t, pq.Int64Array{9, 8, 5, 6, 7}, xdb.Int64Array([]uint64{9, 8, 5, 6, 7}))

	ids3 := xdb.IDArrayFromStrings([]string{"1", "2", "3", "4", "5"})
	assert.Equal(t, []uint64{1, 2, 3, 4, 5}, ids3.List())
}

func TestInt32Array(t *testing.T) {
	assert.Equal(t, pq.Int32Array{1, 2, 3, 4, 5}, xdb.Int32Array([]xdb.ID32{1, 2, 3, 4, 5}))
	assert.Equal(t, pq.Int32Array{1, 2, 3, 4, 5}, xdb.Int32Array([]uint32{1, 2, 3, 4, 5}))
	assert.Equal(t, pq.Int32Array{1, 2, 3, 4, 5}, xdb.Int32Array([]int32{1, 2, 3, 4, 5}))
}

func TestIDsValue(t *testing.T) {
	tcases := []struct {
		in  xdb.IDArray
		exp string
	}{
		{in: xdb.IDArray{xdb.NewID(1), xdb.NewID(2)}, exp: "{1,2}"},
		{in: xdb.IDArray{}, exp: ""},
		{in: nil, exp: ""},
	}

	for _, tc := range tcases {
		dr, err := tc.in.Value()
		require.NoError(t, err)

		v, ok := dr.(string)
		if len(tc.in) > 0 {
			assert.True(t, ok)
			assert.Equal(t, tc.exp, v)
		} else {
			assert.False(t, ok)
			assert.Nil(t, dr)
		}
	}
}

func TestIDsScan(t *testing.T) {
	tcases := []struct {
		exp xdb.IDArray
		val string
	}{
		{val: "{1,2}", exp: xdb.IDArray{xdb.NewID(1), xdb.NewID(2)}},
		{val: "{}", exp: nil},
	}

	for _, tc := range tcases {
		var val2 xdb.IDArray
		err := val2.Scan(tc.val)
		require.NoError(t, err)
		assert.EqualValues(t, tc.exp, val2)
	}

	var val xdb.IDArray
	err := val.Scan("")
	assert.EqualError(t, err, "failed to scan IDs: pq: unable to parse array; expected '{' at offset 0")
	err = val.Scan("{abc}")
	assert.EqualError(t, err, "failed to scan IDs: pq: parsing array element index 0: strconv.ParseInt: parsing \"abc\": invalid syntax")
}

func TestIDsString(t *testing.T) {
	tcases := []struct {
		val xdb.IDArray
		exp []string
	}{
		{val: xdb.IDArray{xdb.NewID(1), xdb.NewID(2)}, exp: []string{"1", "2"}},
		{val: xdb.IDArray{}, exp: []string(nil)},
		{val: nil, exp: nil},
	}

	for _, tc := range tcases {
		assert.Equal(t, tc.exp, tc.val.Strings())
	}
}

type idTest struct {
	ID        xdb.ID
	OrgID     xdb.ID
	Name      string
	UpdatedAt xdb.Time
	CreatedAt xdb.Time
}

func TestIDLog(t *testing.T) {
	var id xdb.ID

	s, ok := any(&id).(fmt.Stringer)
	require.True(t, ok)
	_, ok = any(id).(fmt.Stringer)
	require.True(t, ok)
	assert.Equal(t, "", s.String())
	assert.Equal(t, "", id.String())
	assert.Equal(t, ``, xlog.EscapedString(&id))
	assert.Equal(t, ``, xlog.EscapedString(id))

	b, err := json.Marshal(id)
	require.NoError(t, err)
	assert.Equal(t, `0`, string(b))

	id = xdb.MustID("123456789")
	require.NoError(t, err)
	b, err = json.Marshal(id)
	require.NoError(t, err)
	assert.Equal(t, `123456789`, string(b))

	id2 := id
	assert.Equal(t, id, id2)
	assert.Equal(t, id.String(), id2.String())

	b, err = json.Marshal(idTest{})
	require.NoError(t, err)
	assert.Equal(t, `{"ID":0,"OrgID":0,"Name":"","UpdatedAt":"","CreatedAt":""}`, string(b))

	c1 := idTest{ID: xdb.MustID("123456789"), CreatedAt: xdb.ParseTime("2023-05-29T16:26:14Z")}
	b, err = json.Marshal(c1)
	require.NoError(t, err)
	exp := `{"ID":123456789,"OrgID":0,"Name":"","UpdatedAt":"","CreatedAt":"2023-05-29T16:26:14Z"}`
	assert.Equal(t, exp, string(b))

	var c2 idTest
	err = json.Unmarshal([]byte(exp), &c2)
	require.NoError(t, err)
	assert.Equal(t, c1.ID.String(), c2.ID.String())
	assert.Equal(t, c1, c2)
}

// sharedTestID mimics a package-level fixture shared by many tests,
// which is one of the places the race detector flagged ID.String().
var sharedTestID = xdb.MustID("123456789")

// TestIDStringRace checks that String() and the helpers built on it can be
// called concurrently on copies of the same ID. ID is a value type over a
// shared pointer, so copies held in struct fields, slices, cached rows or
// package-level variables all alias the same backing data.
// Run with `go test -race` to exercise the check.
func TestIDStringRace(t *testing.T) {
	const workers = 32

	run := func(t *testing.T, get func() xdb.ID, exp string) {
		t.Helper()
		var start, done sync.WaitGroup
		start.Add(1)
		done.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer done.Done()
				start.Wait()
				id := get()
				assert.Equal(t, exp, id.String())
				assert.Equal(t, `"`+exp+`"`, id.QuotedString())
				assert.Equal(t, "_"+exp, id.UnderscoreString())
				assert.Equal(t, exp, xdb.IDArray{id}.String())
			}()
		}
		start.Done()
		done.Wait()
	}

	t.Run("copies of NewID", func(t *testing.T) {
		id := xdb.NewID(123456789)
		run(t, func() xdb.ID { return id }, "123456789")
	})

	t.Run("copies of parsed ID", func(t *testing.T) {
		id := xdb.MustID("987654321")
		run(t, func() xdb.ID { return id }, "987654321")
	})

	t.Run("package-level ID", func(t *testing.T) {
		run(t, func() xdb.ID { return sharedTestID }, "123456789")
	})

	t.Run("cached rows", func(t *testing.T) {
		rows := []idTest{{ID: xdb.NewID(555), OrgID: xdb.NewID(777)}}
		run(t, func() xdb.ID { return rows[0].ID }, "555")
		run(t, func() xdb.ID { return rows[0].OrgID }, "777")
	})
}
