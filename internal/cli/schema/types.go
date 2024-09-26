package schema

import (
	"fmt"
	"strings"

	"github.com/effective-security/x/values"
	"github.com/effective-security/xdb/schema"
)

var typesMap = map[string]string{}
var fieldNamesMap = map[string]string{}
var tableNamesMap = map[string]string{}
var modelWithCacheMap = map[string]bool{}

var typeByColumnType = map[string]string{
	"id bigint":      "xdb.ID",
	"id bigint NULL": "xdb.ID",
	"id int8":        "xdb.ID",
	"id int8 NULL":   "xdb.ID",

	"id int":       "xdb.ID32",
	"id int NULL":  "xdb.ID32",
	"id int4":      "xdb.ID32",
	"id int4 NULL": "xdb.ID32",

	"bigint":      "int64",
	"bigint NULL": "xdb.Int64",

	"int8":     "int64",
	"int4":     "int32",
	"int":      "int32",
	"int2":     "int16",
	"smallint": "int16",
	"tinyint":  "int8",

	"decimal": "float64",
	"numeric": "float64",
	"real":    "float32",
	"float4":  "float32",
	"float8":  "float64",

	"bool":    "bool",
	"boolean": "bool",
	"bit":     "bool",

	"jsonb": "xdb.NULLString",
	"bytea": "[]byte",

	"nchar":    "string",
	"nvarchar": "string",
	"char":     "string",
	"varchar":  "string",
	"bpchar":   "string",
	"text":     "string",

	"int8 NULL":     "xdb.Int64",
	"int4 NULL":     "xdb.Int32",
	"int NULL":      "xdb.Int32",
	"int2 NULL":     "xdb.Int32",
	"smallint NULL": "xdb.Int32",
	"tinyint NULL":  "xdb.Int32",

	"bool NULL":    "xdb.Bool",
	"boolean NULL": "xdb.Bool",
	"bit NULL":     "xdb.Bool",

	"decimal NULL": "xdb.Float",
	"numeric NULL": "xdb.Float",
	"real NULL":    "xdb.Float",
	"float4 NULL":  "xdb.Float",
	"float8 NULL":  "xdb.Float",

	"time":        "xdb.Time",
	"date":        "xdb.Time",
	"datetime":    "xdb.Time",
	"datetime2":   "xdb.Time",
	"timestamp":   "xdb.Time",
	"timestamptz": "xdb.Time",

	"nchar NULL":    "xdb.NULLString",
	"nvarchar NULL": "xdb.NULLString",
	"char NULL":     "xdb.NULLString",
	"bpchar NULL":   "xdb.NULLString",
	"varchar NULL":  "xdb.NULLString",
	"text NULL":     "xdb.NULLString",

	"uniqueidentifier":      "xdb.UUID",
	"uuid":                  "xdb.UUID",
	"uniqueidentifier NULL": "xdb.UUID",
	"uuid NULL":             "xdb.UUID",
}

func isID(c *schema.Column) bool {
	return strings.EqualFold(c.Name, "id") ||
		strings.HasSuffix(c.Name, "_id") ||
		strings.HasSuffix(c.Name, "Id") ||
		strings.HasSuffix(c.Name, "ID")
}

func toGoType(c *schema.Column) string {
	if res, ok := typesMap[c.Name]; ok {
		return res
	}
	if res, ok := typesMap[c.SchemaName]; ok {
		return res
	}
	if res, ok := typesMap["_count"]; ok && c.UdtType == "int4" && !c.Nullable && strings.HasSuffix(c.Name, "_count") {
		return res
	}

	if c.Type == "ARRAY" {
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
	}

	typ := values.StringsCoalesce(c.UdtType, c.Type)
	typs := []string{typ}

	if isID(c) {
		typs = []string{"id " + typ, typ}
	}

	for _, typ := range typs {
		if c.Nullable {
			if res := typeByColumnType[typ+" NULL"]; res != "" {
				return res
			}
		}
		if res := typeByColumnType[typ]; res != "" {
			return res
		}
	}

	panic(fmt.Sprintf("don't know how to convert type: %s (%s) %s [%s]",
		c.Type,
		c.UdtType,
		values.Select(c.Nullable, "NULL", "NOT NULL"),
		c.Name))
}
