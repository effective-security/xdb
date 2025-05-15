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
		{"yaml", []string{"schema: dbo\nname: test\nisview: false\ncolumns: []\nindexes: []\nprimarykey: null\n"}},
		{"json", []string{"{\n  \"Schema\": \"dbo\",\n  \"Name\": \"test\",\n  \"IsView\": false,\n  \"Columns\": null,\n  \"Indexes\": null,\n  \"PrimaryKey\": null\n}\n"}},
		{"", []string{"Schema: dbo\nTable: test\n\n┌─────┬──────┬──────┬─────┬──────┬─────┬───────┬─────┐\n│ ORD │ NAME │ TYPE │ UDT │ NULL │ MAX │ INDEX │ REF │\n└─────┴──────┴──────┴─────┴──────┴─────┴───────┴─────┘\n\n"}},
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
		`Schema: dbo
Table: test

┌─────┬──────┬──────┬─────┬──────┬─────┬───────┬─────┐
│ ORD │ NAME │ TYPE │ UDT │ NULL │ MAX │ INDEX │ REF │
└─────┴──────┴──────┴─────┴──────┴─────┴───────┴─────┘

`,
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

func checkEqual(t *testing.T, val any, exp string) {
	w := bytes.NewBuffer([]byte{})
	print.Print(w, val)
	out := w.String()
	assert.Equal(t, exp, out)
}

func TestPrintSchema(t *testing.T) {
	t.Run("Table", func(t *testing.T) {
		o := schema.Table{
			Name:   "test",
			Schema: "dbo",
			Columns: schema.Columns{
				{
					Name:     "ID",
					Type:     "uint64",
					UdtType:  "int8",
					Nullable: false,
				},
				{
					Name:      "Name",
					Type:      "string",
					UdtType:   "varchar",
					Nullable:  true,
					MaxLength: 255,
				},
			},
			Indexes: schema.Indexes{
				{
					Name:        "a",
					IsPrimary:   true,
					ColumnNames: []string{"col1", "col2"},
				},
			},
		}
		checkEqual(t, &o,
			`Schema: dbo
Table: test

┌─────┬──────┬────────┬─────────┬──────┬─────┬───────┬─────┐
│ ORD │ NAME │  TYPE  │   UDT   │ NULL │ MAX │ INDEX │ REF │
├─────┼──────┼────────┼─────────┼──────┼─────┼───────┼─────┤
│ 0   │ ID   │ uint64 │ int8    │      │     │       │     │
│ 0   │ Name │ string │ varchar │ YES  │ 255 │       │     │
└─────┴──────┴────────┴─────────┴──────┴─────┴───────┴─────┘

Indexes:
┌──────┬─────────┬────────┬────────────┐
│ NAME │ PRIMARY │ UNIQUE │  COLUMNS   │
├──────┼─────────┼────────┼────────────┤
│ a    │ YES     │        │ col1, col2 │
└──────┴─────────┴────────┴────────────┘

`,
		)

		o.Indexes = nil
		checkEqual(t, &o,
			`Schema: dbo
Table: test

┌─────┬──────┬────────┬─────────┬──────┬─────┬───────┬─────┐
│ ORD │ NAME │  TYPE  │   UDT   │ NULL │ MAX │ INDEX │ REF │
├─────┼──────┼────────┼─────────┼──────┼─────┼───────┼─────┤
│ 0   │ ID   │ uint64 │ int8    │      │     │       │     │
│ 0   │ Name │ string │ varchar │ YES  │ 255 │       │     │
└─────┴──────┴────────┴─────────┴──────┴─────┴───────┴─────┘

`,
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
		checkEqual(t, o,
			`┌──────┬────────┬───────┬────────┬───────────┬──────────┬───────────┐
│ NAME │ SCHEMA │ TABLE │ COLUMN │ FK SCHEMA │ FK TABLE │ FK COLUMN │
├──────┼────────┼───────┼────────┼───────────┼──────────┼───────────┤
│ FK_1 │ dbo    │ from  │ col1   │ dbo       │ to       │ col2      │
└──────┴────────┴───────┴────────┴───────────┴──────────┴───────────┘

`,
		)
	})

	t.Run("Indexes", func(t *testing.T) {
		o := schema.Indexes{
			{
				Name:        "a",
				IsPrimary:   true,
				ColumnNames: []string{"col1", "col2"},
			},
		}
		checkEqual(t, o,
			`┌──────┬─────────┬────────┬────────────┐
│ NAME │ PRIMARY │ UNIQUE │  COLUMNS   │
├──────┼─────────┼────────┼────────────┤
│ a    │ YES     │        │ col1, col2 │
└──────┴─────────┴────────┴────────────┘

`)
	})
}
