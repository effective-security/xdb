package xdb

import (
	"database/sql/driver"
	"slices"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/lib/pq"
)

// ID defines a type to convert between internal uint64 and external string representations of ID.
// ID is a plain value type: copies are independent, safe for concurrent use,
// comparable with == and ordered with <.
// The zero value is invalid ID, and is stored as NULL in DB.
type ID uint64

// NewID returns ID
func NewID(id uint64) ID {
	return ID(id)
}

// MustID returns ID or panics if the value is invalid
func MustID(val string) ID {
	var id ID
	if err := id.Set(val); err != nil {
		panic(err)
	}
	return id
}

// ParseID returns ID or empty if val is not valid ID
func ParseID(val string) (ID, error) {
	var id ID
	return id, id.Set(val)
}

// TryParseID returns ID or empty if val is not valid ID
func TryParseID(val string) ID {
	var id ID
	_ = id.Set(val)
	return id
}

// MarshalJSON implements json.Marshaler interface
func (v ID) MarshalJSON() ([]byte, error) {
	return strconv.AppendUint(nil, uint64(v), 10), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *ID) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "0" {
		*v = NewID(0)
		return nil
	}

	f, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return errors.Errorf("expected number value to unmarshal ID: %s", s)
	}
	*v = NewID(f)
	return nil
}

func (v ID) MarshalYAML() (any, error) {
	return v.UInt64(), nil
}

func (v *ID) UnmarshalYAML(unmarshal func(any) error) error {
	var val uint64
	if err := unmarshal(&val); err != nil {
		return err
	}
	*v = NewID(val)
	return nil
}

// String returns string representation of ID, or empty string if ID is 0
func (v ID) String() string {
	if v == 0 {
		return ""
	}
	return IDString(uint64(v))
}

// UnderscoreString returns string with underscore prefix.
// This is useful for logs and metrics, where JS can't handle 64-bit numbers.
func (v ID) UnderscoreString() string {
	if v == 0 {
		return ""
	}
	return "_" + IDString(uint64(v))
}

// QuotedString returns string
func (v ID) QuotedString() string {
	if v == 0 {
		return ""
	}
	return QuotedIDString(uint64(v))
}

// Invalid returns if ID is invalid
func (v ID) Invalid() bool {
	return v == 0
}

// IsZero returns if ID is 0
func (v ID) IsZero() bool {
	return v == 0
}

// Valid returns if ID is valid
func (v ID) Valid() bool {
	return v != 0
}

// UInt64 returns uint64 value
func (v ID) UInt64() uint64 {
	return uint64(v)
}

// Reset the value
func (v *ID) Reset() {
	*v = 0
}

// Set the value
func (v *ID) Set(val string) error {
	if val == "" || val == "0" {
		return errors.Errorf("invalid ID: empty value")
	}

	id, err := ParseUint(val)
	if err != nil || id == 0 {
		return errors.Errorf("invalid ID: '%s'", val)
	}
	*v = NewID(id)

	return nil
}

// Scan implements the Scanner interface.
func (v *ID) Scan(value any) error {
	if value == nil {
		return nil
	}

	var id uint64
	switch vid := value.(type) {
	case uint64:
		id = vid
	case int64:
		id = uint64(vid)
	case int:
		id = uint64(vid)
	case uint:
		id = uint64(vid)
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*v = NewID(id)

	return nil
}

// Value implements the driver Valuer interface.
func (v ID) Value() (driver.Value, error) {
	// this makes sure ID can be used as NULL in SQL
	// however this also means that ID(0) will be treated as NULL
	if v == 0 {
		return nil, nil
	}

	// driver.Value support only int64
	return int64(v), nil
}

// IDArray defines a list of IDArray
type IDArray []ID

// NewIDArray returns IDArray
func NewIDArray(vals []uint64) IDArray {
	var ids IDArray
	for _, id := range vals {
		ids = ids.Add(NewID(id))
	}
	return ids
}

// IDArray returns IDArray
func IDArrayFromStrings(vals []string) IDArray {
	var ids IDArray
	for _, val := range vals {
		id, err := ParseID(val)
		if err == nil {
			ids = ids.Add(id)
		}
	}
	return ids
}

// Scan implements the Scanner interface for IDs
func (n *IDArray) Scan(value any) error {
	*n = nil
	if value == nil {
		return nil
	}

	var int64Array pq.Int64Array
	err := int64Array.Scan(value)
	if err != nil {
		return errors.Wrap(err, "failed to scan IDs")
	}

	count := len(int64Array)
	if count > 0 {
		ids := make([]ID, count)
		for i, id := range int64Array {
			ids[i] = NewID(uint64(id))
		}
		*n = ids
	}

	return nil
}

// Value implements the driver Valuer interface for IDs
func (n IDArray) Value() (driver.Value, error) {
	if len(n) == 0 {
		return nil, nil
	}

	ids := make([]int64, len(n))
	for i, id := range n {
		ids[i] = int64(id.UInt64())
	}

	int64Array, err := pq.Int64Array(ids).Value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get IDs value")
	}

	return int64Array, nil
}

// Strings returns string list representation of IDs
func (n IDArray) Strings() []string {
	var list []string
	for _, id := range n {
		list = append(list, id.String())
	}
	return list
}

// String returns string representation of IDs, concatenated with comma
func (n IDArray) String() string {
	return strings.Join(n.Strings(), ",")
}

// List returns list of IDs
func (n IDArray) List() []uint64 {
	var list []uint64
	for _, id := range n {
		list = append(list, id.UInt64())
	}
	return list
}

// Add returns new list
func (n IDArray) Add(id ID) IDArray {
	if n.Contains(id) {
		return n
	}
	return append(n, id)
}

// Concat returns new list
func (n IDArray) Concat(other IDArray) IDArray {
	if len(other) < len(n) {
		return other.Concat(n)
	}
	for _, id := range other {
		n = n.Add(id)
	}
	return n
}

// Sort returns sorted list
func (n IDArray) Sort() IDArray {
	slices.Sort(n)
	return n
}

// Int64Array returns pq.Int64Array
func (n IDArray) Int64Array() pq.Int64Array {
	ids := make(pq.Int64Array, len(n))
	for i, id := range n {
		ids[i] = int64(id.UInt64())
	}
	return ids
}

// Int64Array returns pq.Int64Array
func Int64Array(list []uint64) pq.Int64Array {
	ids := make(pq.Int64Array, len(list))
	for i, id := range list {
		ids[i] = int64(id)
	}
	return ids
}

// Contains returns true if id is in the list
func (n IDArray) Contains(id ID) bool {
	return slices.Contains(n, id)
}

// Overlaps returns true if any of the IDs in the list is in the other list
func (n IDArray) Overlaps(other IDArray) bool {
	return slices.ContainsFunc(n, other.Contains)
}

// ID32 defines a type to convert between internal uint32 and NULL values in DB
type ID32 uint32

// MarshalJSON implements json.Marshaler interface
func (v ID32) MarshalJSON() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *ID32) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "0" || s == "NULL" {
		*v = 0
		return nil
	}

	f, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return errors.Errorf("expected number value to unmarshal ID: %s", s)
	}
	*v = ID32(f)
	return nil
}

func (v ID32) String() string {
	return strconv.FormatUint(uint64(v), 10)
}

// Scan implements the Scanner interface.
func (v *ID32) Scan(value any) error {
	if value == nil {
		return nil
	}

	var id uint64
	switch vid := value.(type) {
	case uint64:
		id = vid
	case int64:
		id = uint64(vid)
	case int:
		id = uint64(vid)
	case uint:
		id = uint64(vid)
	default:
		return errors.Errorf("unsupported scan type: %T", value)
	}

	*v = ID32(id)
	return nil
}

// Value implements the driver Valuer interface.
func (v ID32) Value() (driver.Value, error) {
	// this makes sure ID can be used as NULL in SQL
	// however this also means that ID(0) will be treated as NULL
	if v == 0 {
		return nil, nil
	}

	// driver.Value support only int64
	return int64(v), nil
}

// Enum interface for generic enum
type Enum interface {
	~int32 | ~uint32
}

// Int32Array returns pq.Int32Array
func Int32Array[E Enum](list []E) pq.Int32Array {
	ids := make(pq.Int32Array, len(list))
	for i, id := range list {
		ids[i] = int32(id)
	}
	return ids
}
