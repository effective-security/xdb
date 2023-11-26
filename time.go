package xdb

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// DefaultTimeFormat is the default format for Time.String()
var DefaultTimeFormat = "2006-01-02T15:04:05.999Z07:00"

// DefaultTrucate is the default time to truncate as Postgres time precision is default to 6
var DefaultTrucate = time.Microsecond

// Time implements sql.Time functionality and always returns UTC
type Time time.Time

// Scan implements the Scanner interface.
func (ns *Time) Scan(value any) error {
	var v sql.NullTime

	if str, ok := value.(string); ok {
		*ns = ParseTime(str)
		return nil
	}

	if err := (&v).Scan(value); err != nil {
		return errors.WithStack(err)
	}
	var zero Time
	if v.Valid {
		zero = Time(v.Time.UTC())
	}
	*ns = zero

	return nil
}

// Value implements the driver Valuer interface.
func (ns Time) Value() (driver.Value, error) {
	nst := time.Time(ns)
	return sql.NullTime{
		Valid: !nst.IsZero(),
		Time:  nst.UTC(),
	}.Value()
}

// Now returns Time in UTC
func Now() Time {
	return Time(time.Now().Truncate(DefaultTrucate).UTC())
}

// UTC returns Time in UTC,
func UTC(t time.Time) Time {
	return Time(t.Truncate(DefaultTrucate).UTC())
}

// FromNow returns Time in UTC after now,
// with Second presicions
func FromNow(after time.Duration) Time {
	return Time(time.Now().Add(after).Truncate(DefaultTrucate).UTC())
}

// FromUnixMilli returns Time from Unix milliseconds elapsed since January 1, 1970 UTC.
func FromUnixMilli(tm int64) Time {
	sec := tm / 1000
	msec := tm % 1000
	return Time(time.Unix(sec, msec*int64(time.Millisecond)).UTC())
}

// ParseTime returns Time from RFC3339 format
func ParseTime(val string) Time {
	if val == "" {
		return Time{}
	}

	var t time.Time
	switch len(val) {
	case len(DefaultTimeFormat):
		t, _ = time.Parse(DefaultTimeFormat, val)
	case len(time.RFC3339):
		t, _ = time.Parse(time.RFC3339, val)
	case len(time.DateTime):
		t, _ = time.Parse(time.DateTime, val)
	case len(time.DateOnly):
		t, _ = time.Parse(time.DateOnly, val)
	default:
		t, _ = time.Parse(time.RFC3339Nano, val)
	}
	return Time(t.Truncate(DefaultTrucate).UTC())
}

// UnixMilli returns t as a Unix time, the number of milliseconds elapsed since January 1, 1970 UTC.
func (ns Time) UnixMilli() int64 {
	return time.Time(ns).UnixMilli()
}

// Add returns Time in UTC after this thime,
// with Second presicions
func (ns Time) Add(after time.Duration) Time {
	return Time(time.Time(ns).Add(after).Truncate(DefaultTrucate).UTC())
}

// UTC returns t with the location set to UTC.
func (ns Time) UTC() time.Time {
	return time.Time(ns).UTC()
}

// IsZero reports whether t represents the zero time instant, January 1, year 1, 00:00:00 UTC.
func (ns Time) IsZero() bool {
	return time.Time(ns).IsZero()
}

// IsNil reports whether t represents the zero time instant, January 1, year 1, 00:00:00 UTC.
func (ns Time) IsNil() bool {
	return time.Time(ns).IsZero()
}

// Ptr returns pointer to Time, or nil if the time is zero
func (ns Time) Ptr() *time.Time {
	t := ns.UTC()
	if t.IsZero() {
		return nil
	}
	return &t
}

// String returns string in RFC3339 format,
// if it's Zero time, an empty string is returned
func (ns Time) String() string {
	t := ns.UTC()
	if t.IsZero() {
		return ""
	}
	return t.Format(DefaultTimeFormat)
}

// MarshalJSON implements the json.Marshaler interface.
// The time is a quoted string in RFC 3339 format, with sub-second precision added if present.
func (ns Time) MarshalJSON() ([]byte, error) {
	t := ns.UTC()
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return t.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in RFC 3339 format.
func (ns *Time) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal([]byte(`""`), data) {
		*ns = Time{}
		return nil
	}
	return errors.WithStack(json.Unmarshal([]byte(data), (*time.Time)(ns)))
}
