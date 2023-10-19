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
			exp: "int",
		},
		{
			col: dbschema.Column{Type: "int", Nullable: "YES"},
			exp: "*int",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: "NO"},
			exp: "int",
		},
		{
			col: dbschema.Column{Type: "bigint", Nullable: "YES"},
			exp: "*int",
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
			got := sqlToGoType(&tc.col)
			assert.Equal(t, tc.exp, got, "sqlToGoType(%v) = %s; want %s", tc.col, got, tc.exp)
		})
	}

	assert.Panics(t, func() { sqlToGoType(&dbschema.Column{Type: "unknown"}) }, "sqlToGoType(unknown) should panic")
}
