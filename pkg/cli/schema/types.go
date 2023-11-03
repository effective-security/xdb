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
	ptr := ""
	if c.Nullable == yesVal {
		ptr = "*"
	}

	switch c.Type {

	case "bigint":
		if c.Name == "id" || strings.HasSuffix(c.Name, "_id") {
			return "xdb.ID"
		}

		return ptr + "int64"

	case "int", "integer":
		typeName := "int"
		switch c.UdtType {
		case "int2":
			typeName = "int16"
		case "int4":
			typeName = "int32"
		case "int8":
			typeName = "int64"
		}
		return ptr + typeName
	case "smallint":
		return ptr + "int16"
	case "decimal", "numeric":
		return ptr + "float64"

	case "real":
		return ptr + "float32"

	case "boolean":
		return ptr + "bool"

	case "jsonb":
		return "xdb.NULLString"

	case "char", "varchar", "character", "character varying", "text":
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
	ptr := ""
	if c.Nullable == yesVal {
		ptr = "*"
	}

	switch c.Type {

	case "bigint":
		if c.Name == "id" || strings.HasSuffix(c.Name, "_id") {
			return "xdb.ID"
		}

		return ptr + "int64"

	case "int", "integer":
		return ptr + "int32"

	case "smallint":
		return ptr + "int16"

	case "tinyint":
		return ptr + "int8"

	case "decimal", "numeric":
		return ptr + "float64"

	case "bit", "boolean":
		return ptr + "bool"

	case "jsonb":
		return "xdb.NULLString"

	case "char", "nchar", "varchar", "varchar2", "nvarchar", "character", "character varying", "text":
		if c.Nullable == yesVal {
			return "xdb.NULLString"
		}
		return ptr + "string"

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
