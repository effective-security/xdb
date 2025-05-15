package print

import (
	"fmt"
	"io"
	"strings"

	"github.com/effective-security/x/values"
	"github.com/effective-security/xdb/schema"
	"github.com/olekukonko/tablewriter"
)

func SchemaTables(w io.Writer, r schema.Tables) {
	for _, t := range r {
		SchemaTable(w, t)
	}
}

// SchemaTable prints schema.Table
func SchemaTable(w io.Writer, r *schema.Table) {
	fmt.Fprintf(w, "Schema: %s\nTable: %s\n\n", r.Schema, r.Name)

	table := tablewriter.NewTable(w)
	table.Header([]string{"Ord", "Name", "Type", "UDT", "NULL", "Max", "Index", "Ref"})

	for _, c := range r.Columns {
		// TODO: select
		maxL := ""
		if c.MaxLength > 0 {
			maxL = fmt.Sprintf("%d", c.MaxLength)
		}
		ref := ""
		if c.Ref != nil {
			ref = c.Ref.RefColumnSchemaName()
		}

		_ = table.Append([]string{
			fmt.Sprintf("%d", c.Position),
			c.Name,
			c.Type,
			c.UdtType,
			values.Select(c.Nullable, "YES", ""),
			maxL,
			values.Select(c.IsIndex(), "YES", ""),
			ref,
		})
	}

	_ = table.Render()

	if len(r.Indexes) > 0 {
		fmt.Fprintf(w, "\nIndexes:\n")
		SchemaIndexes(w, r.Indexes)
	} else {
		fmt.Fprintln(w)
	}
}

// SchemaIndexes prints schema.Indexes
func SchemaIndexes(w io.Writer, r schema.Indexes) {
	table := tablewriter.NewTable(w)
	table.Header([]string{"Name", "Primary", "Unique", "Columns"})

	for _, c := range r {
		_ = table.Append([]string{
			c.Name,
			values.Select(c.IsPrimary, "YES", ""),
			values.Select(c.IsUnique, "YES", ""),
			strings.Join(c.ColumnNames, ", "),
		})
	}

	_ = table.Render()
	fmt.Fprintln(w)
}

// SchemaForeingKeys prints schema.ForeingKeys
func SchemaForeingKeys(w io.Writer, r schema.ForeignKeys) {
	table := tablewriter.NewTable(w)
	table.Header([]string{"Name", "Schema", "Table", "Column", "FK Schema", "FK Table", "FK Column"})

	for _, c := range r {
		_ = table.Append([]string{
			c.Name,
			c.Schema,
			c.Table,
			c.Column,
			c.RefSchema,
			c.RefTable,
			c.RefColumn,
		})
	}

	_ = table.Render()
	fmt.Fprintln(w)
}
