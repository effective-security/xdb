package xsql

import (
	"reflect"
	"strings"
	"sync"
)

type bindField struct {
	name  string
	index []int
}

var bindCache sync.Map // map[reflect.Type][]bindField

func bindFields(typ reflect.Type) []bindField {
	if cached, ok := bindCache.Load(typ); ok {
		return cached.([]bindField)
	}
	fields := collectBindFields(typ, nil)
	actual, _ := bindCache.LoadOrStore(typ, fields)
	return actual.([]bindField)
}

func collectBindFields(typ reflect.Type, prefix []int) []bindField {
	var fields []bindField
	for i := 0; i < typ.NumField(); i++ {
		t := typ.Field(i)
		index := append(append([]int{}, prefix...), i)
		if t.Anonymous && t.Type.Kind() == reflect.Struct {
			if skipDBTag(t.Tag.Get("db")) {
				continue
			}
			fields = append(fields, collectBindFields(t.Type, index)...)
			continue
		}
		col := columnName(t.Tag.Get("db"))
		if col == "" {
			continue
		}
		fields = append(fields, bindField{name: col, index: index})
	}
	return fields
}

func skipDBTag(tag string) bool {
	if tag == "" {
		return false
	}
	return columnName(tag) == ""
}

func columnName(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return ""
	}
	return name
}
