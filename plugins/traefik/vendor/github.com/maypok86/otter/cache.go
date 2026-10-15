// Package otter is a Yaegi compatible replacement of
// github.com/maypok86/otter for the Traefik plugin.
//
// Yaegi (the Go interpreter used by Traefik) can't compile the upstream
// package: it converts unsafe.Pointer to generic node types, links to the Go
// runtime and instantiates generics across packages. This package keeps the
// upstream observable behaviour for the only instantiation the otter storage
// uses (string keys, []byte values, variable TTL, a cost of 1 per entry):
//   - the expiration is computed with the upstream formula, in seconds,
//     on uint32: a negative TTL wraps around and never expires, a zero TTL
//     expires immediately;
//   - the size is bounded by the capacity, the oldest inserted entry is
//     evicted first (upstream uses S3-FIFO, which also favours old entries).
package otter

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

type entry struct {
	key        string
	value      []byte
	expiration uint32
}

type store struct {
	mu       sync.RWMutex
	items    map[string]*list.Element
	order    *list.List
	capacity int
}

// StringBytesCache replaces CacheWithVariableTTL[string, []byte]. Like
// upstream it is a value type whose copies share the same storage.
type StringBytesCache struct {
	s *store
}

var startTime = time.Now()

// now returns the seconds elapsed since the package initialization, like
// the upstream unixtime clock.
func now() uint32 {
	//nolint:gosec // there will never be an overflow
	return uint32(time.Since(startTime) / time.Second)
}

func getExpiration(ttl time.Duration) uint32 {
	//nolint:gosec // same overflow as upstream, it is relied upon
	return now() + uint32((ttl+time.Second-1)/time.Second)
}

func (e *entry) hasExpired() bool {
	return e.expiration <= now()
}

// NewStringBytesCache replaces
// MustBuilder[string, []byte](capacity).Cost(1).WithVariableTTL().Build().
func NewStringBytesCache(capacity int) (StringBytesCache, error) {
	if capacity <= 0 {
		return StringBytesCache{}, errors.New("otter: capacity should be positive")
	}

	return StringBytesCache{s: &store{
		items:    make(map[string]*list.Element),
		order:    list.New(),
		capacity: capacity,
	}}, nil
}

// Has checks if there is an entry with the given key in the cache.
func (c StringBytesCache) Has(key string) bool {
	_, ok := c.Get(key)

	return ok
}

// Get returns the value associated with the key in this cache.
func (c StringBytesCache) Get(key string) ([]byte, bool) {
	c.s.mu.RLock()
	elem, ok := c.s.items[key]
	if !ok {
		c.s.mu.RUnlock()

		return nil, false
	}

	e := elem.Value.(*entry)
	if !e.hasExpired() {
		c.s.mu.RUnlock()

		return e.value, true
	}
	c.s.mu.RUnlock()

	c.s.mu.Lock()
	if current, ok := c.s.items[key]; ok && current == elem {
		c.s.remove(elem)
	}
	c.s.mu.Unlock()

	return nil, false
}

// Set associates the value with the key in this cache and sets the custom ttl for this key-value pair.
//
// It always returns true as every entry costs 1, which never exceeds the capacity.
func (c StringBytesCache) Set(key string, value []byte, ttl time.Duration) bool {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if elem, ok := c.s.items[key]; ok {
		c.s.remove(elem)
	}

	for len(c.s.items) >= c.s.capacity {
		c.s.evictOne()
	}

	c.s.items[key] = c.s.order.PushBack(&entry{key: key, value: value, expiration: getExpiration(ttl)})

	return true
}

// SetIfAbsent if the specified key is not already associated with a value associates it with the given value
// and sets the custom ttl for this key-value pair.
//
// If the specified key is not already associated with a value, then it returns false.
func (c StringBytesCache) SetIfAbsent(key string, value []byte, ttl time.Duration) bool {
	if _, ok := c.Get(key); ok {
		return false
	}

	return c.Set(key, value, ttl)
}

// Delete removes the association for this key from the cache.
func (c StringBytesCache) Delete(key string) {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if elem, ok := c.s.items[key]; ok {
		c.s.remove(elem)
	}
}

// DeleteByFunc removes the association for this key from the cache when the given function returns true.
func (c StringBytesCache) DeleteByFunc(f func(key string, value []byte) bool) {
	var keys []string
	c.Range(func(key string, value []byte) bool {
		if f(key, value) {
			keys = append(keys, key)
		}

		return true
	})

	for _, key := range keys {
		c.Delete(key)
	}
}

// Range iterates over all entries in the cache.
//
// Iteration stops early when the given function returns false.
func (c StringBytesCache) Range(f func(key string, value []byte) bool) {
	c.s.mu.RLock()
	entries := make([]*entry, 0, len(c.s.items))
	for elem := c.s.order.Front(); elem != nil; elem = elem.Next() {
		if e := elem.Value.(*entry); !e.hasExpired() {
			entries = append(entries, e)
		}
	}
	c.s.mu.RUnlock()

	for _, e := range entries {
		if !f(e.key, e.value) {
			return
		}
	}
}

// Clear clears the hash table, all policies, buffers, etc.
func (c StringBytesCache) Clear() {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	c.s.items = make(map[string]*list.Element)
	c.s.order.Init()
}

// Close clears the hash table, all policies, buffers, etc and stop all goroutines.
func (c StringBytesCache) Close() {
	c.Clear()
}

// Size returns the current number of entries in the cache.
func (c StringBytesCache) Size() int {
	c.s.mu.RLock()
	defer c.s.mu.RUnlock()

	return len(c.s.items)
}

// Capacity returns the cache capacity.
func (c StringBytesCache) Capacity() int {
	return c.s.capacity
}

// remove must be called with the write lock held.
func (s *store) remove(elem *list.Element) {
	delete(s.items, elem.Value.(*entry).key)
	s.order.Remove(elem)
}

// evictOne drops the oldest inserted entry. It must be called with the
// write lock held.
func (s *store) evictOne() {
	if front := s.order.Front(); front != nil {
		s.remove(front)
	}
}
