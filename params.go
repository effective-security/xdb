package xdb

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/effective-security/x/values"
)

// PageableByOffset is an interface for pagination.
// The limit and offset are the last two arguments.
type PageableByOffset interface {
	// Page returns the limit and offset for pagination.
	Page() (limit uint32, offset uint32)
}

// PageableByCursor is an interface for pagination.
// The cursor and limit are the last two arguments.
// The cursor argument is before limit because it's used in WHERE clause.
type PageableByCursor interface {
	// Cursor returns the limit and cursor for pagination.
	Cursor() (limit uint32, cursor any)
}

// HasQueryParams is an interface for objects with query parameters.
type HasQueryParams interface {
	QueryParams() QueryParams
}

// GetQueryParams returns query parameters from an object.
func GetQueryParams(args ...any) QueryParams {
	for _, h := range args {
		if p, ok := h.(HasQueryParams); ok {
			return p.QueryParams()
		}
		if p, ok := h.(QueryParams); ok {
			return p
		}
	}
	panic("invalid interface: no query parameters")
}

// QueryParams is an interface for query parameters.
type QueryParams interface {
	PageableByOffset
	PageableByCursor

	Name() string
	Args() []any
	// IsSet checks if a positional query parameter is set.
	IsSet(pos uint32) bool
	// GetEnum checks if an enum query parameter is set.
	GetEnum(pos uint32) (int32, bool)
	// GetFlags returns additional flags for query parameter.
	GetFlags() []int32
}

type enumPosition struct {
	position uint32
	value    int32
}

// QueryParams is a placeholder for query parameters.
type QueryParamsBuilder struct {
	queryName string

	flags     []int32
	positions uint64 // bit flags for positional parameters
	enums     []enumPosition
	args      []any
	hash      string

	// Limit specifies maximimum number of records to return
	limit uint32
	// Offset specifies the offset for pagination
	offset uint32
	// Cursor specifies the cursor for pagination
	cursor any
}

// NewQueryParams creates a new query parameters builder.
func NewQueryParams(queryName string) *QueryParamsBuilder {
	return &QueryParamsBuilder{
		queryName: queryName,
	}
}

func (b *QueryParamsBuilder) Reset() {
	b.positions = 0
	b.flags = nil
	b.enums = nil
	b.args = nil
	b.hash = ""
	b.limit = 0
	b.offset = 0
	b.cursor = nil
}

// Name returns a hash of the query parameters.
func (b *QueryParamsBuilder) Name() string {
	if b.hash == "" {
		var n strings.Builder

		n.WriteString(b.queryName)
		n.WriteRune('_')
		n.WriteRune('x')
		n.WriteString(strconv.FormatUint(b.positions, 16))

		for _, e := range b.enums {
			n.WriteRune('_')
			n.WriteString(strconv.FormatUint(uint64(e.position), 10))
			n.WriteRune('x')
			n.WriteString(strconv.FormatUint(uint64(e.value), 16))
		}
		for _, f := range b.flags {
			n.WriteString("_fx")
			n.WriteString(strconv.FormatInt(int64(f), 16))
		}
		if b.cursor != nil {
			n.WriteString("_c")
		} else if b.offset > 0 {
			n.WriteString("_o")
		}

		b.hash = n.String()
	}
	return b.hash
}

// Args returns a list of query arguments.
func (b *QueryParamsBuilder) Args() []any {
	return b.args
}

// Set sets a positional query parameter, and adds it to the list of arguments.
func (b *QueryParamsBuilder) Set(pos uint32, v any) {
	if pos > 63 {
		panic("enum position is out of range")
	}
	b.checkPage()
	b.positions |= 1 << pos
	b.args = append(b.args, v)
}

func (b *QueryParamsBuilder) checkPage() {
	if b.limit > 0 {
		panic("limit already set: limit and offset must be last arguments")
	}
}

// SetPage sets the limit for pagination, and adds it to the list of arguments.
func (b *QueryParamsBuilder) SetPage(limit, offset uint32) {
	b.checkPage()
	b.limit = values.NumbersCoalesce(limit, DefaultPageSize)
	b.offset = offset
	b.args = append(b.args, b.limit, b.offset)
}

// Page returns the limit and offset for pagination, if supported
func (b *QueryParamsBuilder) Page() (limit uint32, offset uint32) {
	return b.limit, b.offset
}

// SetCursor sets the limit for pagination, and adds it to the list of arguments.
func (b *QueryParamsBuilder) SetCursor(limit uint32, pos uint32, cursor any) {
	b.Set(pos, cursor)
	b.cursor = cursor
	b.limit = values.NumbersCoalesce(limit, DefaultPageSize)
	b.args = append(b.args, b.limit)
}

// Cursor returns the limit and cursor for pagination, if supported
func (b *QueryParamsBuilder) Cursor() (limit uint32, cursor any) {
	return b.limit, b.cursor
}

// AddArgs adds an additional query arguments, such as Limit or Offset
func (b *QueryParamsBuilder) AddArgs(v ...any) {
	b.checkPage()
	b.args = append(b.args, v...)
}

// IsSet checks if a positional query parameter is set.
func (b *QueryParamsBuilder) IsSet(pos uint32) bool {
	return b.positions&(1<<pos) != 0
}

// SetEnum sets an enum query parameter, without adding it to the list of arguments.
func (b *QueryParamsBuilder) SetEnum(pos uint32, v int32) {
	if pos > 63 {
		panic("enum position is out of range")
	}
	b.checkPage()
	b.positions |= 1 << pos
	b.enums = append(b.enums, enumPosition{pos, v})
}

// GetEnum checks if an enum query parameter is set.
func (b *QueryParamsBuilder) GetEnum(pos uint32) (int32, bool) {
	// do not use map as for small set of enums the linear search is faster
	for _, e := range b.enums {
		if e.position == pos {
			return e.value, true
		}
	}
	return 0, false
}

// SetFlags sets additional flags for query parameter.
func (b *QueryParamsBuilder) SetFlags(v ...int32) {
	b.flags = v
}

// GetFlags returns additional flags for query parameter.
func (b *QueryParamsBuilder) GetFlags() []int32 {
	return b.flags
}

// PageParam converts a parameter to uint32
func PageParam(p any) uint32 {
	switch p := p.(type) {
	case int:
		return uint32(p)
	case uint32:
		return p
	case sql.NamedArg:
		return PageParam(p.Value)
	default:
		panic("invalid parameter type: expected int, uint32, sql.NamedArg")
	}
}
