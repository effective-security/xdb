package xsql

import (
	"strconv"
	"strings"
)

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
	b.positions |= 1 << pos
	b.args = append(b.args, v)
}

// AddArgs adds an additional query arguments, such as Limit or Offset
func (b *QueryParamsBuilder) AddArgs(v ...any) {
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
