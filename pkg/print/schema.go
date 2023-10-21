package print

import (
	"fmt"
	"io"

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

	table := tablewriter.NewWriter(w)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Name", "Type", "UDT", "NULL", "Max", "Ref"})
	table.SetHeaderLine(true)

	for _, c := range r.Columns {
		maxL := ""
		if c.MaxLength != nil {
			maxL = fmt.Sprintf("%d", *c.MaxLength)
		}
		ref := ""
		if c.Ref != nil {
			ref = c.Ref.RefColumnSchemaName()
		}
		table.Append([]string{
			c.Name,
			c.Type,
			c.UdtType,
			c.Nullable,
			maxL,
			ref,
		})
	}

	table.Render()
	fmt.Fprintln(w)
}

// SchemaForeingKeys prints schema.ForeingKeys
func SchemaForeingKeys(w io.Writer, r schema.ForeignKeys) {
	table := tablewriter.NewWriter(w)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Name", "Schema", "Table", "Column", "FK Schema", "FK Table", "FK Column"})
	table.SetHeaderLine(true)

	for _, c := range r {
		table.Append([]string{
			c.Name,
			c.Schema,
			c.Table,
			c.Column,
			c.RefSchema,
			c.RefTable,
			c.RefColumn,
		})
	}

	table.Render()
	fmt.Fprintln(w)
}
