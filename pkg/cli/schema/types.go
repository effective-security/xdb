package schema

import (
	"fmt"
	"strings"

	"github.com/effective-security/xdb/schema"
)

const yesVal = "YES"

func sqlToGoType(provider string) func(c *schema.Column) string {
	switch provider {
	case "postgres":
		return postgresToGoType
	case "sqlserver":
		return sqlserverToGoType
	default:
		panic("unknown provider")
	}
}

func postgresToGoType(c *schema.Column) string {
	switch c.Type {

	case "bigint":
		if c.Name == "id" || strings.HasSuffix(c.Name, "_id") {
			return "xdb.ID"
		}
		if c.Nullable == yesVal {
			return "*int64"
		}
		return "int64"

	case "integer":
		typeName := "int"
		switch c.UdtType {
		case "int2":
			typeName = "int16"
		case "int4":
			typeName = "int32"
		case "int8":
			typeName = "int64"
		}
		if c.Nullable == yesVal {
			typeName = "*" + typeName
		}
		return typeName
	case "smallint":
		if c.Nullable == yesVal {
			return "*int16"
		}
		return "int16"
	case "decimal", "numeric":
		if c.Nullable == yesVal {
			return "*float64"
		}
		return "float64"

	case "real":
		if c.Nullable == yesVal {
			return "*float32"
		}
		return "float32"

	case "boolean":
		if c.Nullable == yesVal {
			return "*bool"
		}
		return "bool"

	case "jsonb":
		return "xdb.NULLString"

	case "char", "character varying", "text":
		if c.Nullable == yesVal {
			return "xdb.NULLString"
		}
		return "string"

	case "timestamp", "timestamp with time zone", "timestamp without time zone":
		return "xdb.Time"

	case "bytea":
		return "[]byte"

	case "ARRAY":
		typeName := "[]"
		switch c.UdtType {
		case "_int8":
			if strings.HasSuffix(c.Name, "_ids") {
				typeName = "xdb.IDArray"
			} else {
				typeName = "pq.Int64Array"
			}
		case "_text", "_varchar":
			typeName = "pq.StringArray"
		default:
			panic(fmt.Sprintf("don't know how to convert ARRAY: %s [%s]", c.UdtType, c.Name))
		}
		return typeName

	default:
		panic(fmt.Sprintf("don't know how to convert type: %s [%s]", c.Type, c.Name))
	}
}

func sqlserverToGoType(c *schema.Column) string {
	switch c.Type {

	case "bigint":
		if c.Name == "id" || strings.HasSuffix(c.Name, "_id") {
			return "xdb.ID"
		}

		if c.Nullable == yesVal {
			return "*int64"
		}
		return "int64"

	case "int", "integer":
		if c.Nullable == yesVal {
			return "*int32"
		}
		return "int32"

	case "smallint":
		if c.Nullable == yesVal {
			return "*int16"
		}
		return "int16"

	case "tinyint":
		if c.Nullable == yesVal {
			return "*int8"
		}
		return "int8"

	case "decimal", "numeric":
		if c.Nullable == yesVal {
			return "*float64"
		}
		return "float64"

	case "bit", "boolean":
		if c.Nullable == yesVal {
			return "*bool"
		}
		return "bool"

	case "jsonb":
		return "xdb.NULLString"

	case "char", "nchar", "varchar", "varchar2", "nvarchar", "character varying", "text":
		if c.Nullable == yesVal {
			return "xdb.NULLString"
		}
		return "string"

	case "uniqueidentifier":
		if c.Nullable == yesVal {
			return "xdb.NULLString"
		}
		return "string"

	case "time", "date", "datetime", "datetime2":
		return "xdb.Time"
	default:
		panic(fmt.Sprintf("don't know how to convert type: %s [%s]", c.Type, c.Name))
	}
}
