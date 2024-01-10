package print

import (
	"fmt"
	"io"
	"strings"

	"github.com/effective-security/x/slices"
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
	table.SetHeader([]string{"Name", "Type", "UDT", "NULL", "Max", "Index", "Ref"})
	table.SetHeaderLine(true)

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

		table.Append([]string{
			c.Name,
			c.Type,
			c.UdtType,
			slices.Select(c.Nullable, "YES", ""),
			maxL,
			slices.Select(c.IsIndex(), "YES", ""),
			ref,
		})
	}

	table.Render()

	if len(r.Indexes) > 0 {
		fmt.Fprintf(w, "\nIndexes:\n")
		SchemaIndexes(w, r.Indexes)
	} else {
		fmt.Fprintln(w)
	}
}

// SchemaIndexes prints schema.Indexes
func SchemaIndexes(w io.Writer, r schema.Indexes) {
	table := tablewriter.NewWriter(w)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Name", "Primary", "Unique", "Columns"})
	table.SetHeaderLine(true)

	for _, c := range r {
		table.Append([]string{
			c.Name,
			slices.Select(c.IsPrimary, "YES", ""),
			slices.Select(c.IsUnique, "YES", ""),
			strings.Join(c.ColumnNames, ", "),
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
