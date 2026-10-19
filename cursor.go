package xdb

import (
	"encoding"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
)

// Cursor is the position of a keyset (seek) page: the sort key of the last
// row of the previous page, in ORDER BY order. Its String form is an opaque,
// URL-safe token for API clients; ParseCursor reads it back and Scan returns
// the typed key values, which the next page binds in its WHERE clause:
//
//	// ORDER BY created_at DESC, id DESC
//	c, err := xdb.NewCursor(last.CreatedAt, last.ID)
//	next := c.String()
//
//	c, err = xdb.ParseCursor(next)
//	var at xdb.Time
//	var id xdb.ID
//	err = c.Scan(&at, &id) // WHERE (created_at, id) < (?, ?)
//
// The key keeps full precision: integers as decimal digits and times as
// RFC 3339 with nanoseconds, so rows that differ only in microseconds are
// not skipped. The zero Cursor is the first page.
type Cursor struct {
	key []string
}

// NewCursor returns the cursor that continues after a row with the given
// sort key values, in ORDER BY order.
// A key value is a Time, time.Time, an encoding.TextMarshaler (by value or
// pointer receiver), a type with an integer, unsigned, float, bool or string
// kind (ID, enums, NULLString, ...), or a pointer to one of them. A key type
// with custom text methods implements both MarshalText and UnmarshalText.
// NewCursor fails when MarshalText fails or a value has another type, so no
// token is issued that Scan cannot read back; keys of Time, integer and
// string types never fail.
func NewCursor(key ...any) (Cursor, error) {
	c := Cursor{
		key: make([]string, len(key)),
	}
	for i, v := range key {
		s, err := cursorText(v)
		if err != nil {
			return Cursor{}, errors.WithMessagef(err, "invalid cursor value %d", i)
		}
		c.key[i] = s
	}
	return c, nil
}

// ParseCursor decodes the String form of a cursor. An empty string is the
// zero cursor, which starts at the first page.
func ParseCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, nil
	}
	js, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, errors.WithMessage(err, "invalid cursor")
	}
	var key []string
	if err := json.Unmarshal(js, &key); err != nil || len(key) == 0 {
		return Cursor{}, errors.New("invalid cursor")
	}
	return Cursor{key: key}, nil
}

// String returns the opaque, URL-safe form of the cursor, or an empty string
// for the zero cursor.
func (c Cursor) String() string {
	if len(c.key) == 0 {
		return ""
	}
	// a list of strings always marshals
	js, _ := json.Marshal(c.key)
	return base64.RawURLEncoding.EncodeToString(js)
}

// IsZero returns true for the zero cursor, which starts at the first page.
func (c Cursor) IsZero() bool {
	return len(c.key) == 0
}

// Len returns the number of values in the sort key.
func (c Cursor) Len() int {
	return len(c.key)
}

// Scan copies the sort key into dest, which must match it in number and
// order. A destination is a non-nil pointer to Time, time.Time, an
// encoding.TextUnmarshaler, or a type with an integer, unsigned, float, bool
// or string kind; text methods take precedence over the kind. A cursor that
// does not decode into dest returns an error, which a service reports as an
// invalid argument.
func (c Cursor) Scan(dest ...any) error {
	if len(dest) != len(c.key) {
		return errors.Errorf("invalid cursor: %d values, expected %d", len(c.key), len(dest))
	}
	for i, d := range dest {
		if err := scanCursorText(c.key[i], d); err != nil {
			return errors.WithMessagef(err, "invalid cursor value %d", i)
		}
	}
	return nil
}

// cursorText returns the text form of a sort key value; scanCursorText
// reverses it.
func cursorText(v any) (string, error) {
	switch v := v.(type) {
	case nil:
		return "", nil
	case Time:
		return time.Time(v).UTC().Format(time.RFC3339Nano), nil
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano), nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return "", nil
	}
	// custom text methods take precedence over the kind, as in scanCursorText
	if m, ok := textMarshaler(v); ok {
		b, err := m.MarshalText()
		if err != nil {
			return "", errors.WithStack(err)
		}
		return string(b), nil
	}
	if rv.Kind() == reflect.Pointer {
		return cursorText(rv.Elem().Interface())
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, rv.Type().Bits()), nil
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool()), nil
	case reflect.String:
		return rv.String(), nil
	}
	return "", errors.Errorf("unsupported type: %T", v)
}

// textMarshaler returns the text marshaler of v, or of a pointer to a copy of
// v when the method has a pointer receiver.
func textMarshaler(v any) (encoding.TextMarshaler, bool) {
	if m, ok := v.(encoding.TextMarshaler); ok {
		return m, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		return nil, false
	}
	p := reflect.New(rv.Type())
	p.Elem().Set(rv)
	m, ok := p.Interface().(encoding.TextMarshaler)
	return m, ok
}

func scanCursorText(s string, dest any) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.Errorf("unsupported destination: %T", dest)
	}

	switch d := dest.(type) {
	case *Time:
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return errors.WithStack(err)
		}
		*d = Time(t.UTC())
		return nil
	case *time.Time:
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return errors.WithStack(err)
		}
		*d = t.UTC()
		return nil
	case encoding.TextUnmarshaler:
		return errors.WithStack(d.UnmarshalText([]byte(s)))
	}

	ev := rv.Elem()
	switch ev.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, ev.Type().Bits())
		if err != nil {
			return errors.WithStack(err)
		}
		ev.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, ev.Type().Bits())
		if err != nil {
			return errors.WithStack(err)
		}
		ev.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, ev.Type().Bits())
		if err != nil {
			return errors.WithStack(err)
		}
		ev.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return errors.WithStack(err)
		}
		ev.SetBool(b)
	case reflect.String:
		ev.SetString(s)
	default:
		return errors.Errorf("unsupported destination: %T", dest)
	}
	return nil
}
