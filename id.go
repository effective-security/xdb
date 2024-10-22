package xdb

import (
	"database/sql/driver"
	"sort"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// ID defines a type to convert between internal uint64 and external string representations of ID
type ID struct {
	id *idptr
}

// NewID returns ID
func NewID(id uint64) ID {
	var v ID
	if id > 0 {
		v.id = &idptr{id: id}
	}
	return v
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
	return []byte(strconv.FormatUint(v.id.val(), 10)), nil
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
	*v = ID{id: &idptr{id: f, str: s}}
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

func (v ID) String() string {
	return v.id.String()
}

// Invalid returns if ID is invalid
func (v ID) Invalid() bool {
	return v.id.val() == 0
}

// IsZero returns if ID is 0
func (v ID) IsZero() bool {
	return v.id.val() == 0
}

// Valid returns if ID is valid
func (v ID) Valid() bool {
	return v.id.val() != 0
}

// UInt64 returns uint64 value
func (v ID) UInt64() uint64 {
	return v.id.val()
}

// Reset the value
func (v *ID) Reset() {
	if v.id == nil {
		v.id = &idptr{}
	} else {
		v.id.id = 0
		v.id.str = ""
	}
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
	if v.id == nil {
		v.id = &idptr{}
	}
	v.id.id = id
	v.id.str = ""

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
	if v.id.val() == 0 {
		return nil, nil
	}

	// driver.Value support only int64
	return int64(v.id.id), nil
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
	for _, v := range n {
		if v.UInt64() == id.UInt64() {
			return n
		}
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
	if len(n) < 2 {
		return n
	}

	sort.Slice(n, func(i, j int) bool {
		return n[i].UInt64() < n[j].UInt64()
	})
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
	for _, v := range n {
		if v.UInt64() == id.UInt64() {
			return true
		}
	}
	return false
}

// Overlaps returns true if any of the IDs in the list is in the other list
func (n IDArray) Overlaps(other IDArray) bool {
	for _, v := range n {
		if other.Contains(v) {
			return true
		}
	}
	return false
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

type idptr struct {
	id  uint64
	str string
}

func (v *idptr) val() uint64 {
	if v == nil {
		return 0
	}
	return v.id
}

func (v *idptr) String() string {
	if v == nil {
		return ""
	}
	if v.id != 0 && v.str == "" {
		v.str = IDString(v.id)
	}
	return v.str
}
