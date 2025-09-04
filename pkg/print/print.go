// Package print provides helper package to print objects.
package print

import (
	"encoding/json"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/effective-security/xdb/schema"
	"gopkg.in/yaml.v3"
)

// JSON prints value to out
func JSON(w io.Writer, value any) error {
	json, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return errors.WithMessage(err, "failed to encode")
	}
	_, _ = w.Write(json)
	_, _ = w.Write([]byte{'\n'})
	return nil
}

// Yaml prints value  to out
func Yaml(w io.Writer, value any) error {
	y, err := yaml.Marshal(value)
	if err != nil {
		return errors.WithMessage(err, "failed to encode")
	}
	_, _ = w.Write(y)
	return nil
}

// Object prints value to out in format
func Object(w io.Writer, format string, value any) error {
	if format == "yaml" {
		return Yaml(w, value)
	}
	if format == "json" {
		return JSON(w, value)
	}
	Print(w, value)
	return nil
}

// Print value
func Print(w io.Writer, value any) {
	switch t := value.(type) {
	case *schema.Table:
		SchemaTable(w, t)
	case schema.Tables:
		SchemaTables(w, t)
	case schema.ForeignKeys:
		SchemaForeingKeys(w, t)
	case schema.Indexes:
		SchemaIndexes(w, t)
	default:
		_ = JSON(w, value)
	}
}
