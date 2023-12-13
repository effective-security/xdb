package xsql

import (
	"unsafe"

	"github.com/valyala/bytebufferpool"
)

func (d *Dialect) GetCachedQuery(name string) (string, bool) {
	res, ok := d.cache.Load(name)
	if ok {
		return res.(string), ok
	}

	return "", ok
}

func (d *Dialect) PutCachedQuery(name, sql string) {
	d.cache.Store(name, sql)
}

// GetOrCreateQuery returns a cached query by name or creates a new one.
func (d *Dialect) GetOrCreateQuery(name string, create func(name string) Builder) string {
	if qstr, ok := d.GetCachedQuery(name); ok {
		return qstr
	}
	q := create(name)
	// will store query in cache
	return q.SetName(name).String()
}

// bufToString returns a string pointing to a ByteBuffer contents
// It helps to avoid memory copyng.
// Use the returned string with care, make sure to never use it after
// the ByteBuffer is deallocated or returned to a pool.
func bufToString(buf *bytebufferpool.ByteBuffer) string {
	return *(*string)(unsafe.Pointer(&buf.B))
}
