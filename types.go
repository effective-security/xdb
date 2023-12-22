package xdb

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/effective-security/x/slices"
	"github.com/pkg/errors"
)

// Max values, common for strings
const (
	MaxLenForName     = 64
	MaxLenForEmail    = 160
	MaxLenForShortURL = 256
)

// Validator provides schema validation interface
type Validator interface {
	// Validate returns error if the model is not valid
	Validate() error
}

// Validate returns error if the model is not valid
func Validate(m any) error {
	if v, ok := m.(Validator); ok {
		return v.Validate()
	}
	return nil
}

// NullTime from *time.Time
func NullTime(val *time.Time) sql.NullTime {
	if val == nil {
		return sql.NullTime{Valid: false}
	}

	return sql.NullTime{Time: *val, Valid: true}
}

// TimePtr returns nil if time is zero, or pointer with a value
func TimePtr(val Time) *time.Time {
	t := time.Time(val)
	if t.IsZero() {
		return nil
	}
	return &t
}

// String returns string
func String(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}

// ParseUint returns id from the string
func ParseUint(id string) (uint64, error) {
	i64, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return i64, nil
}

// IDString returns string id
func IDString(id uint64) string {
	return strconv.FormatUint(id, 10)
}

// Strings de/encodes the string slice to/from a SQL string.
type Strings []string

// Scan implements the Scanner interface.
func (n *Strings) Scan(value any) error {
	if value == nil {
		*n = nil
		return nil
	}
	v := fmt.Sprint(value)
	if len(v) == 0 {
		*n = Strings{}
		return nil
	}
	return errors.WithStack(json.Unmarshal([]byte(v), n))
}

// Value implements the driver Valuer interface.
func (n Strings) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	value, err := json.Marshal(n)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return string(value), nil
}

// Metadata de/encodes the string map to/from a SQL string.
type Metadata map[string]string

// Scan implements the Scanner interface.
func (n *Metadata) Scan(value any) error {
	if value == nil {
		*n = nil
		return nil
	}
	v := fmt.Sprint(value)
	if len(v) == 0 {
		*n = Metadata{}
		return nil
	}
	return errors.WithStack(json.Unmarshal([]byte(v), n))
}

// Value implements the driver Valuer interface.
func (n Metadata) Value() (driver.Value, error) {
	if len(n) == 0 {
		return nil, nil
	}
	value, err := json.Marshal(n)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return string(value), nil
}

// NULLString de/encodes the string a SQL string.
type NULLString string

// Scan implements the Scanner interface.
func (ns *NULLString) Scan(value any) error {
	var v sql.NullString
	if err := (&v).Scan(value); err != nil {
		return errors.WithStack(err)
	}
	if v.Valid {
		*ns = NULLString(v.String)
	} else {
		*ns = ""
	}

	return nil
}

// Value implements the driver Valuer interface.
func (ns NULLString) Value() (driver.Value, error) {
	if ns == "" {
		return nil, nil
	}
	return string(ns), nil
}

// UUID de/encodes the string a SQL string.
type UUID string

// Scan implements the Scanner interface.
func (ns *UUID) Scan(value any) error {
	if value == nil {
		*ns = ""
		return nil
	}

	var s string
	var err error
	switch vid := value.(type) {
	case []byte:
		if len(vid) != 16 {
			return errors.WithMessagef(err, "failed to parse UUID: %v", vid)
		}
		s = fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
			vid[3], vid[2], vid[1], vid[0], vid[5], vid[4], vid[7], vid[6], vid[8], vid[9], vid[10], vid[11], vid[12], vid[13], vid[14], vid[15])
	case string:
		s = vid
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*ns = UUID(s)
	return nil
}

// Value implements the driver Valuer interface.
func (ns UUID) Value() (driver.Value, error) {
	if ns == "" {
		return nil, nil
	}
	return string(ns), nil
}

// Int64 represents SQL int64 NULL
type Int64 int64

// MarshalJSON implements json.Marshaler interface
func (v Int64) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", v)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Int64) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "0" || s == "NULL" {
		*v = 0
		return nil
	}

	f, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return errors.Errorf("expected number value to unmarshal ID: %s", s)
	}
	*v = Int64(f)
	return nil
}

func (v Int64) String() string {
	return strconv.FormatInt(int64(v), 10)
}

// Scan implements the Scanner interface.
func (v *Int64) Scan(value any) error {
	if value == nil {
		return nil
	}

	var id int64
	switch vid := value.(type) {
	case uint64:
		id = int64(vid)
	case int64:
		id = int64(vid)
	case int:
		id = int64(vid)
	case uint:
		id = int64(vid)
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*v = Int64(id)
	return nil
}

// Value implements the driver Valuer interface.
func (v Int64) Value() (driver.Value, error) {
	// this makes sure ID can be used as NULL in SQL
	// however this also means that ID(0) will be treated as NULL
	if v == 0 {
		return nil, nil
	}

	// driver.Value support only int64
	return int64(v), nil
}

// Int32 represents SQL int NULL
type Int32 int32

// MarshalJSON implements json.Marshaler interface
func (v Int32) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", v)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Int32) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "0" || s == "NULL" {
		*v = 0
		return nil
	}

	f, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return errors.Errorf("expected number value to unmarshal ID: %s", s)
	}
	*v = Int32(f)
	return nil
}

func (v Int32) String() string {
	return strconv.FormatInt(int64(v), 10)
}

// Scan implements the Scanner interface.
func (v *Int32) Scan(value any) error {
	if value == nil {
		return nil
	}

	var id int64
	switch vid := value.(type) {
	case uint64:
		id = int64(vid)
	case int64:
		id = int64(vid)
	case int:
		id = int64(vid)
	case uint:
		id = int64(vid)
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*v = Int32(id)
	return nil
}

// Value implements the driver Valuer interface.
func (v Int32) Value() (driver.Value, error) {
	// this makes sure ID can be used as NULL in SQL
	// however this also means that ID(0) will be treated as NULL
	if v == 0 {
		return nil, nil
	}

	// driver.Value support only int64
	return int64(v), nil
}

// Float represents SQL float64 NULL
type Float float64

// MarshalJSON implements json.Marshaler interface
func (v Float) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%f", v)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Float) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "0" || s == "NULL" {
		*v = 0
		return nil
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("expected number value to unmarshal ID: %s", s)
	}
	*v = Float(f)
	return nil
}

func (v Float) String() string {
	return strconv.FormatFloat(float64(v), 'f', 6, 64)
}

// Scan implements the Scanner interface.
func (v *Float) Scan(value any) error {
	if value == nil {
		return nil
	}

	var f float64
	var err error
	switch vid := value.(type) {
	case []byte:
		sf := string(vid)
		if f, err = strconv.ParseFloat(sf, 64); err != nil {
			return errors.WithMessagef(err, "failed to parse float: %v", sf)
		}
	case uint64:
		f = float64(vid)
	case int64:
		f = float64(vid)
	case int:
		f = float64(vid)
	case uint:
		f = float64(vid)
	case float32:
		f = float64(vid)
	case float64:
		f = float64(vid)
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*v = Float(f)
	return nil
}

// Value implements the driver Valuer interface.
func (v Float) Value() (driver.Value, error) {
	// this makes sure ID can be used as NULL in SQL
	// however this also means that ID(0) will be treated as NULL
	if v == 0 {
		return nil, nil
	}

	// driver.Value support only float64
	return float64(v), nil
}

// Bool represents SQL bool NULL
type Bool bool

// MarshalJSON implements json.Marshaler interface
func (v Bool) MarshalJSON() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Bool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "true" || s == "1" || s == "TRUE" {
		*v = true
	} else {
		*v = false
	}
	return nil
}

func (v Bool) String() string {
	return slices.Select(bool(v), "true", "false")
}

// Scan implements the Scanner interface.
func (v *Bool) Scan(value any) error {
	if value == nil {
		return nil
	}

	var id bool
	switch vid := value.(type) {
	case uint64:
		id = vid > 0
	case int64:
		id = vid > 0
	case int:
		id = vid > 0
	case uint:
		id = vid > 0
	case bool:
		id = vid
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*v = Bool(id)
	return nil
}

// Value implements the driver Valuer interface.
func (v Bool) Value() (driver.Value, error) {
	// this makes sure ID can be used as NULL in SQL
	// however this also means that ID(0) will be treated as NULL
	if !v {
		return nil, nil
	}
	return bool(v), nil
}

// IsNotFoundError returns true, if error is NotFound
func IsNotFoundError(err error) bool {
	return err != nil &&
		(err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows in result set"))
}
