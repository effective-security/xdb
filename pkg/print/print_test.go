package print_test

import (
	"bytes"
	"testing"

	"github.com/effective-security/xdb/pkg/print"
	"github.com/effective-security/xdb/schema"
	"github.com/stretchr/testify/assert"
)

func TestObject(t *testing.T) {
	ver := schema.Table{
		Name:   "test",
		Schema: "dbo",
	}
	tcases := []struct {
		format string
		has    []string
	}{
		{"yaml", []string{"schema: dbo\nname: test\ncolumns: []\n"}},
		{"json", []string{"{\n  \"Schema\": \"dbo\",\n  \"Name\": \"test\",\n  \"Columns\": null,\n  \"Indexes\": null,\n  \"PrimaryKey\": null\n}\n"}},
		{"", []string{"Schema: dbo\nTable: test\n\n  NAME | TYPE | UDT | NULL | MAX | REF  \n-------+------+-----+------+-----+------\n\n"}},
	}
	w := bytes.NewBuffer([]byte{})
	for _, tc := range tcases {
		w.Reset()

		_ = print.Object(w, tc.format, &ver)
		out := w.String()
		for _, exp := range tc.has {
			assert.Contains(t, out, exp)
		}
	}

	// print value
	w.Reset()
	_ = print.Object(w, "", &ver)
	assert.Equal(t,
		"Schema: dbo\n"+
			"Table: test\n\n"+
			"  NAME | TYPE | UDT | NULL | MAX | REF  \n"+
			"-------+------+-----+------+-----+------\n\n",
		w.String())
}

func checkFormat(t *testing.T, val any, has ...string) {
	w := bytes.NewBuffer([]byte{})
	print.Print(w, val)
	out := w.String()
	for _, exp := range has {
		assert.Contains(t, out, exp, "%T", val)
	}
}

func TestPrintSchema(t *testing.T) {
	t.Run("Table", func(t *testing.T) {
		maxL := 255
		o := schema.Table{
			Name:   "test",
			Schema: "dbo",
			Columns: schema.Columns{
				{
					Name:     "ID",
					Type:     "uint64",
					UdtType:  "int8",
					Nullable: "NO",
				},
				{
					Name:      "Name",
					Type:      "string",
					UdtType:   "varchar",
					Nullable:  "YES",
					MaxLength: &maxL,
				},
			},
		}
		checkFormat(t, &o,
			"Schema: dbo\n"+
				"Table: test\n\n"+
				"  NAME |  TYPE  |   UDT   | NULL | MAX | REF  \n"+
				"-------+--------+---------+------+-----+------\n"+
				"  ID   | uint64 | int8    | NO   |     |      \n"+
				"  Name | string | varchar | YES  | 255 |      \n\n",
		)
	})

	t.Run("FK", func(t *testing.T) {
		o := schema.ForeignKeys{
			{
				Name:      "FK_1",
				Schema:    "dbo",
				Table:     "from",
				Column:    "col1",
				RefSchema: "dbo",
				RefTable:  "to",
				RefColumn: "col2",
			},
		}
		checkFormat(t, o,
			"  NAME | SCHEMA | TABLE | COLUMN | FK SCHEMA | FK TABLE | FK COLUMN  \n"+
				"-------+--------+-------+--------+-----------+----------+------------\n"+
				"  FK_1 | dbo    | from  | col1   | dbo       | to       | col2       \n\n",
		)
	})
}
