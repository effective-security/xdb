package xdb_test

import (
	"testing"
	"time"

	"github.com/effective-security/xdb"
	"github.com/stretchr/testify/assert"
)

func TestTimeFormat(t *testing.T) {
	tcases := []struct {
		t         time.Time
		expXdbStr string
		expXdb    xdb.Time
	}{
		{
			time.Date(2019, 11, 30, 17, 45, 59, 999999999, time.UTC),
			"2019-11-30T17:45:59.999Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 999000000, time.UTC)),
		},
		{
			time.Date(2019, 11, 30, 17, 45, 59, 123456789, time.UTC),
			"2019-11-30T17:45:59.123Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 123000000, time.UTC)),
		},
		{
			time.Date(2019, 11, 30, 17, 45, 59, 12345678, time.UTC),
			"2019-11-30T17:45:59.012Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 12000000, time.UTC)),
		},
		{
			time.Date(2019, 11, 30, 17, 45, 59, 1234, time.UTC),
			"2019-11-30T17:45:59Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 0, time.UTC)),
		},
		{
			time.Date(2019, 11, 30, 17, 45, 59, 678000000, time.UTC),
			"2019-11-30T17:45:59.678Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 678000000, time.UTC)),
		},
	}

	for _, tc := range tcases {
		s := xdb.Time(tc.t).String()
		assert.Equal(t, tc.expXdbStr, s)
		assert.Equal(t, tc.expXdb.UTC(), xdb.ParseTime(s).UTC(), "case:"+tc.expXdbStr)
	}
}

func TestTimeParse(t *testing.T) {
	tcases := []struct {
		s      string
		expXdb xdb.Time
	}{
		{
			"2019-11-30T17:45:59.999Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 999000000, time.UTC)),
		},
		{
			"2019-11-30T17:45:59.123Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 123000000, time.UTC)),
		},
		{
			"2019-11-30T17:45:59.012Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 12000000, time.UTC)),
		},
		{
			"2019-11-30T17:45:59Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 0, time.UTC)),
		},
		{
			"2019-11-30T17:45:59.678Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 678000000, time.UTC)),
		},
		{
			"2019-11-30T17:45:59.999999999Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 999999999, time.UTC)),
		},
		{
			"2019-11-30T17:45:59.99999999Z",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 999999990, time.UTC)),
		},
		{
			"2019-11-30",
			xdb.Time(time.Date(2019, 11, 30, 0, 0, 0, 0, time.UTC)),
		},
		{
			"2019-11-30 17:45:59",
			xdb.Time(time.Date(2019, 11, 30, 17, 45, 59, 0, time.UTC)),
		},
	}

	for _, tc := range tcases {
		assert.Equal(t, tc.expXdb.UTC(), xdb.ParseTime(tc.s).UTC(), "case:"+tc.s)
	}
}

func TestTimeTruncate(t *testing.T) {
	d := time.Date(2019, 1, 2, 3, 4, 5, 1234567, time.UTC)
	d2 := time.Date(2019, 1, 2, 3, 4, 5, 1000000, time.UTC)

	now := xdb.Time(d)
	assert.Equal(t, "2019-01-02T03:04:05.001Z", now.String())

	nowBackFromString := xdb.Time(d2)
	assert.Equal(t, nowBackFromString.UTC(), xdb.ParseTime(now.String()).UTC())

	now = nowBackFromString.Add(time.Second)
	assert.Equal(t, now.UTC(), xdb.ParseTime(now.String()).UTC())
}
