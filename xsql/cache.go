package xsql

import (
	"unsafe"

	"github.com/valyala/bytebufferpool"
)

func (d *Dialect) GetCachedQuery(name string) (string, bool) {
	d.cacheMu.Lock()
	sql, ok := d.cache[name]
	d.cacheMu.Unlock()
	return sql, ok
}

func (d *Dialect) PutCachedQuery(name, sql string) {
	d.cacheMu.Lock()
	defer d.cacheMu.Unlock()
	if _, ok := d.cache[name]; ok {
		d.cache[name] = sql
		return
	}
	if d.maxCache > 0 && int64(len(d.cache)) >= d.maxCache {
		return
	}
	d.cache[name] = sql
}

// GetOrCreateQuery returns a cached query by name or creates a new one.
// The Builder returned by create is closed after the SQL is rendered and cached.
// create must not call Close on the Builder it returns.
func (d *Dialect) GetOrCreateQuery(name string, create func(name string) Builder) (string, string) {
	if qstr, ok := d.GetCachedQuery(name); ok {
		return qstr, name
	}
	q := create(name)
	sql := q.SetName(name).String()
	q.Close()
	return sql, name
}

// ClearCache drops all cached SQL strings for this dialect.
// Entries are deleted in place so dialects created via WithNewLines
// keep sharing the same map.
func (d *Dialect) ClearCache() {
	d.cacheMu.Lock()
	for k := range d.cache {
		delete(d.cache, k)
	}
	d.cacheMu.Unlock()
}

// bufToString returns a string pointing to a ByteBuffer contents
// It helps to avoid memory copying.
// Use the returned string with care, make sure to never use it after
// the ByteBuffer is deallocated or returned to a pool.
func bufToString(buf *bytebufferpool.ByteBuffer) string {
	return *(*string)(unsafe.Pointer(&buf.B))
}
