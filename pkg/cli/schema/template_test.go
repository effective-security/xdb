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
			col: dbschema.Column{Type: "int", Nullable: "NO"},
			exp: "int32",
		},
		{
			col: dbschema.Column{Type: "int", Nullable: "YES"},
			exp: "*int32",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: "NO"},
			exp: "int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: "YES"},
			exp: "*int64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: "NO"},
			exp: "float64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: "YES"},
			exp: "*float64",
		},
		{
			col: dbschema.Column{Type: "bit", Nullable: "NO"},
			exp: "bool",
		},
		{
			col: dbschema.Column{Type: "bit", Nullable: "YES"},
			exp: "*bool",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: "NO"},
			exp: "string",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: "YES"},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "time", Nullable: "NO"},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "time", Nullable: "YES"},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "uniqueidentifier", Nullable: "NO"},
			exp: "string",
		},
		{
			col: dbschema.Column{Type: "uniqueidentifier", Nullable: "YES"},
			exp: "xdb.NULLString",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.col.Type+tc.col.Nullable, func(t *testing.T) {
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
			col: dbschema.Column{Type: "smallint", UdtType: "int2", Nullable: "NO"},
			exp: "int16",
		},
		{
			col: dbschema.Column{Type: "smallint", UdtType: "int2", Nullable: "YES"},
			exp: "*int16",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int2", Nullable: "NO"},
			exp: "int16",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int2", Nullable: "YES"},
			exp: "*int16",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int4", Nullable: "NO"},
			exp: "int32",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int4", Nullable: "YES"},
			exp: "*int32",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int8", Nullable: "NO"},
			exp: "int64",
		},
		{
			col: dbschema.Column{Type: "int", UdtType: "int8", Nullable: "YES"},
			exp: "*int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Name: "test_id", Nullable: "NO"},
			exp: "xdb.ID",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: "NO"},
			exp: "int64",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: "YES"},
			exp: "*int64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: "NO"},
			exp: "float64",
		},
		{
			col: dbschema.Column{Type: "decimal", Nullable: "YES"},
			exp: "*float64",
		},
		{
			col: dbschema.Column{Type: "real", Nullable: "NO"},
			exp: "float32",
		},
		{
			col: dbschema.Column{Type: "real", Nullable: "YES"},
			exp: "*float32",
		},
		{
			col: dbschema.Column{Type: "boolean", Nullable: "NO"},
			exp: "bool",
		},
		{
			col: dbschema.Column{Type: "boolean", Nullable: "YES"},
			exp: "*bool",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: "NO"},
			exp: "string",
		},
		{
			col: dbschema.Column{Type: "varchar", Nullable: "YES"},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "timestamp with time zone", Nullable: "NO"},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "timestamp without time zone", Nullable: "YES"},
			exp: "xdb.Time",
		},
		{
			col: dbschema.Column{Type: "jsonb", Nullable: "NO"},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "jsonb", Nullable: "YES"},
			exp: "xdb.NULLString",
		},
		{
			col: dbschema.Column{Type: "bytea", Nullable: "YES"},
			exp: "[]byte",
		},
		{
			col: dbschema.Column{Type: "ARRAY", UdtType: "_int8", Nullable: "YES"},
			exp: "pq.Int64Array",
		},
		{
			col: dbschema.Column{Type: "ARRAY", UdtType: "_int8", Nullable: "YES", Name: "test_ids"},
			exp: "xdb.IDArray",
		},
		{
			col: dbschema.Column{Type: "ARRAY", UdtType: "_varchar", Nullable: "YES"},
			exp: "pq.StringArray",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.col.Type+tc.col.Nullable, func(t *testing.T) {
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
