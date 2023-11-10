package xdb

import (
	"reflect"

	"dario.cat/mergo"
)

type transformers struct {
}

var (
	typeOfDate       = reflect.TypeOf(Time{})
	typeOfID         = reflect.TypeOf(ID{})
	typeOfNULLString = reflect.TypeOf(NULLString(""))
)

func (t transformers) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	switch typ {
	case typeOfDate:
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				val := src.Interface().(Time)
				if !val.IsZero() {
					dst.Set(src)
				}
			}
			return nil
		}
	case typeOfID:
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				val := src.Interface().(ID)
				if val.Valid() {
					dst.Set(src)
				}
			}
			return nil
		}
	case typeOfNULLString:
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				val := src.String()
				if val != "" {
					dst.Set(src)
				}
			}
			return nil
		}
	default:
		//logger.KV(xlog.DEBUG, "type", typ)
	}
	return nil
}

// MergeOpts is a mergo option to merge structs
var MergeOpts = mergo.WithTransformers(transformers{})

// Merge merges two structs with xdb types
func Merge(dst any, src any) error {
	return mergo.Merge(dst, src, MergeOpts, mergo.WithOverride)
}
