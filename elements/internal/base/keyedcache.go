package base

import (
	"sync"
	"weak"
)

// KeyedCache is a typed, self-contained version of Keyed: a map of
// per-item state that drops entries not asked for in a while. Hold one
// in a stateful component:
//
//	type fileList struct {
//		rows *giode.KeyedCache[string, *giode.PressableElement]
//	}
//	...
//	row := l.rows.Get(file.ID, func() *giode.PressableElement { ... })
//
// Expired entries are dropped at the end of each frame. It is safe for
// concurrent use, though state is normally only touched on the UI
// goroutine.
type KeyedCache[K comparable, V any] struct {
	mu      sync.Mutex
	entries map[K]*keyedCacheEntry[V]
	retain  uint64
	onEvict func(K, V)
}

type keyedCacheEntry[V any] struct {
	value V
	frame uint64
}

// NewKeyedCache returns an empty cache that keeps entries for
// DefaultRetainFrames unrequested frames.
func NewKeyedCache[K comparable, V any]() *KeyedCache[K, V] {
	c := &KeyedCache[K, V]{entries: map[K]*keyedCacheEntry[V]{}, retain: DefaultRetainFrames}
	wp := weak.Make(c)
	keyedCacheMu.Lock()
	keyedCaches = append(keyedCaches, func(frame uint64) bool {
		if c := wp.Value(); c != nil {
			c.sweep(frame)
			return true
		}
		return false
	})
	keyedCacheMu.Unlock()
	return c
}

// Retain sets how many frames an entry survives without being asked
// for (minimum 1). It returns c for chaining.
func (c *KeyedCache[K, V]) Retain(frames int) *KeyedCache[K, V] {
	c.mu.Lock()
	c.retain = uint64(max(1, frames))
	c.mu.Unlock()
	return c
}

// OnEvict sets a function called with each dropped entry, to release
// resources it holds. It returns c for chaining.
func (c *KeyedCache[K, V]) OnEvict(fn func(K, V)) *KeyedCache[K, V] {
	c.mu.Lock()
	c.onEvict = fn
	c.mu.Unlock()
	return c
}

// Get returns the entry for key, creating it with init the first time.
func (c *KeyedCache[K, V]) Get(key K, init func() V) V {
	c.mu.Lock()
	if e, ok := c.entries[key]; ok {
		e.frame = frameNumber
		c.mu.Unlock()
		retainKeyed(func() { c.touch(e) })
		return e.value
	}
	c.mu.Unlock()
	v := init()
	e := &keyedCacheEntry[V]{value: v, frame: frameNumber}
	c.mu.Lock()
	c.entries[key] = e
	c.mu.Unlock()
	retainKeyed(func() { c.touch(e) })
	return v
}

func (c *KeyedCache[K, V]) touch(e *keyedCacheEntry[V]) {
	c.mu.Lock()
	e.frame = frameNumber
	c.mu.Unlock()
}

// Peek returns the entry for key without creating it or keeping it
// alive.
func (c *KeyedCache[K, V]) Peek(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Delete drops the entry for key right away.
func (c *KeyedCache[K, V]) Delete(key K) {
	c.mu.Lock()
	e, ok := c.entries[key]
	delete(c.entries, key)
	fn := c.onEvict
	c.mu.Unlock()
	if ok && fn != nil {
		fn(key, e.value)
	}
}

// Len returns the number of entries.
func (c *KeyedCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Range calls fn for each entry until it returns false.
func (c *KeyedCache[K, V]) Range(fn func(K, V) bool) {
	c.mu.Lock()
	snapshot := make(map[K]V, len(c.entries))
	for k, e := range c.entries {
		snapshot[k] = e.value
	}
	c.mu.Unlock()
	for k, v := range snapshot {
		if !fn(k, v) {
			return
		}
	}
}

func (c *KeyedCache[K, V]) sweep(frame uint64) {
	type evicted struct {
		k K
		v V
	}
	var gone []evicted
	c.mu.Lock()
	for k, e := range c.entries {
		if frame-e.frame > c.retain {
			gone = append(gone, evicted{k, e.value})
			delete(c.entries, k)
		}
	}
	fn := c.onEvict
	c.mu.Unlock()
	if fn != nil {
		for _, g := range gone {
			fn(g.k, g.v)
		}
	}
}
