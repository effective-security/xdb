package xsql

import (
	"unsafe"

	"github.com/valyala/bytebufferpool"
)

type sqlCache map[string]string

/*
ClearCache clears the statement cache.

In most cases you don't need to care about it. It's there to
let caller free memory when a caller executes zillions of unique
SQL statements.
*/
func (d *Dialect) ClearCache() {
	d.cacheLock.Lock()
	d.cache = make(sqlCache)
	d.cacheLock.Unlock()
}

func (d *Dialect) getCache() sqlCache {
	d.cacheOnce.Do(func() {
		d.cache = make(sqlCache)
	})
	return d.cache
}

func (d *Dialect) GetCachedQuery(name string) (string, bool) {
	c := d.getCache()
	d.cacheLock.RLock()
	res, ok := c[name]
	d.cacheLock.RUnlock()
	return res, ok
}

func (d *Dialect) PutCachedQuery(name, sql string) {
	c := d.getCache()
	d.cacheLock.Lock()
	c[name] = sql
	d.cacheLock.Unlock()
}

// bufToString returns a string pointing to a ByteBuffer contents
// It helps to avoid memory copyng.
// Use the returned string with care, make sure to never use it after
// the ByteBuffer is deallocated or returned to a pool.
func bufToString(buf *bytebufferpool.ByteBuffer) string {
	return *(*string)(unsafe.Pointer(&buf.B))
}
