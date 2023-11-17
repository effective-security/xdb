package schema

import (
	"testing"

	dbschema "github.com/effective-security/xdb/schema"
	"github.com/stretchr/testify/assert"
)

func TestSqlToGoType(t *testing.T) {

	tcases := []struct {
		col dbschema.Column
		exp string
	}{
		{
			col: dbschema.Column{Type: "int", Nullable: false},
			exp: "int32",
		},
		{
			col: dbschema.Column{Type: "int", Nullable: true},
			exp: "*int32",
		},
		{
			col: dbschema.Column{Type: "int", Nullable: false, Name: "AccountId"},
			exp: "uint32",
		},
		{
			col: dbschema.Column{Type: "int", Nullable: true, Name: "RefId"},
			exp: "*uint32",
		},

		{
			col: dbschema.Column{Type: "bigint", Nullable: false},
			exp: "int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: true},
			exp: "*int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: false, Name: "id"},
			exp: "xdb.ID",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: true, Name: "id"},
			exp: "xdb.ID",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: false},
			exp: "float64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: true},
			exp: "*float64",
		},
		{
			col: dbschema.Column{Type: "bit", Nullable: false},
			exp: "bool",
		},
		{
			col: dbschema.Column{Type: "bit", Nullable: true},
			exp: "*bool",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: false},
			exp: "string",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: true},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "time", Nullable: false},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "time", Nullable: true},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "uniqueidentifier", Nullable: false},
			exp: "string",
		},
		{
			col: dbschema.Column{Type: "uniqueidentifier", Nullable: true},
			exp: "xdb.NULLString",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.col.Type, func(t *testing.T) {
			got := sqlserverToGoType(&tc.col)
			assert.Equal(t, tc.exp, got, "sqlToGoType(%v) = %s; want %s", tc.col, got, tc.exp)
		})
	}

	assert.Panics(t, func() { sqlserverToGoType(&dbschema.Column{Type: "unknown"}) }, "sqlserverToGoType(unknown) should panic")
	assert.Panics(t, func() { sqlToGoType("unknown") }, "sqlToGoType(unknown) should panic")
}

func TestPgToGoType(t *testing.T) {

	tcases := []struct {
		col dbschema.Column
		exp string
	}{
		{
			col: dbschema.Column{Type: "smallint", UdtType: "int2", Nullable: false},
			exp: "int16",
		},
		{
			col: dbschema.Column{Type: "smallint", UdtType: "int2", Nullable: true},
			exp: "*int16",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int2", Nullable: false},
			exp: "int16",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int2", Nullable: true},
			exp: "*int16",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int4", Nullable: false},
			exp: "int32",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int4", Nullable: true},
			exp: "*int32",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int4", Nullable: false, Name: "AccountId"},
			exp: "uint32",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int4", Nullable: true, Name: "refId"},
			exp: "*uint32",
		},

		{
			col: dbschema.Column{Type: "int", UdtType: "int8", Nullable: false},
			exp: "int64",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int8", Nullable: true},
			exp: "*int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Name: "test_id", Nullable: false},
			exp: "xdb.ID",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: false},
			exp: "int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: true},
			exp: "*int64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: false},
			exp: "float64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: true},
			exp: "*float64",
		},
		{
			col: dbschema.Column{Type: "real", Nullable: false},
			exp: "float32",
		},
		{
			col: dbschema.Column{Type: "real", Nullable: true},
			exp: "*float32",
		},
		{
			col: dbschema.Column{Type: "boolean", Nullable: false},
			exp: "bool",
		},
		{
			col: dbschema.Column{Type: "boolean", Nullable: true},
			exp: "*bool",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: false},
			exp: "string",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: true},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "timestamp with time zone", Nullable: false},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "timestamp without time zone", Nullable: true},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "jsonb", Nullable: false},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "jsonb", Nullable: true},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "bytea", Nullable: true},
			exp: "[]byte",
		},
		{
			col: dbschema.Column{Type: "ARRAY", UdtType: "_int8", Nullable: true},
			exp: "pq.Int64Array",
		},
		{
			col: dbschema.Column{Type: "ARRAY", UdtType: "_int8", Nullable: true, Name: "test_ids"},
			exp: "xdb.IDArray",
		},
		{
			col: dbschema.Column{Type: "ARRAY", UdtType: "_varchar", Nullable: true},
			exp: "pq.StringArray",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.col.Type, func(t *testing.T) {
			got := postgresToGoType(&tc.col)
			assert.Equal(t, tc.exp, got, "postgresToGoType(%v) = %s; want %s", tc.col, got, tc.exp)
		})
	}

	assert.Panics(t, func() { postgresToGoType(&dbschema.Column{Type: "unknown"}) }, "postgresToGoType(unknown) should panic")
	assert.Panics(t, func() { sqlToGoType("unknown") }, "sqlToGoType(unknown) should panic")
}

func TestGoName(t *testing.T) {

	tcases := map[string]string{
		"id":         "ID",
		"_id":        "Xid",
		"createdAt":  "CreatedAt",
		"created_at": "CreatedAt",
	}
	for n, exp := range tcases {
		assert.Equal(t, exp, goName(n))
	}
}
