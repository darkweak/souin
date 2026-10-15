// Package ttlcache is a Yaegi compatible replacement of
// github.com/jellydator/ttlcache/v3 for the Traefik plugin.
//
// Yaegi (the Go interpreter used by Traefik) can't compile a generic type
// used in a type assertion or a conversion inside a generic function, which
// the upstream package relies on everywhere. This package keeps the upstream
// behaviour (LRU order, touch on hit, capacity and expiration evictions,
// insertion/eviction hooks) for the only instantiation the simplefs storage
// uses: string keys and []byte values.
package ttlcache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

const (
	// NoTTL indicates that an item should never expire.
	NoTTL time.Duration = -1

	// PreviousOrDefaultTTL indicates that existing TTL of item should be used
	// default TTL will be used as fallback if item doesn't exist
	PreviousOrDefaultTTL time.Duration = -2

	// DefaultTTL indicates that the default TTL value of the cache
	// instance should be used.
	DefaultTTL time.Duration = 0
)

// EvictionReason is used to specify why a certain item was
// evicted/deleted.
type EvictionReason int

// Available eviction reasons.
const (
	EvictionReasonDeleted EvictionReason = iota + 1
	EvictionReasonCapacityReached
	EvictionReasonExpired
)

// StringBytesItem holds a key/value pair and its expiration metadata.
type StringBytesItem struct {
	mu        sync.RWMutex
	key       string
	value     []byte
	ttl       time.Duration
	expiresAt time.Time
}

func newItem(key string, value []byte, ttl time.Duration) *StringBytesItem {
	item := &StringBytesItem{key: key, value: value, ttl: ttl}
	item.touch()

	return item
}

func (item *StringBytesItem) update(value []byte, ttl time.Duration) {
	item.mu.Lock()
	defer item.mu.Unlock()

	item.value = value
	if ttl == PreviousOrDefaultTTL {
		return
	}

	item.ttl = ttl
	item.expiresAt = time.Time{}
	item.touchUnsafe()
}

func (item *StringBytesItem) touch() {
	item.mu.Lock()
	defer item.mu.Unlock()

	item.touchUnsafe()
}

func (item *StringBytesItem) touchUnsafe() {
	if item.ttl <= 0 {
		return
	}

	item.expiresAt = time.Now().Add(item.ttl)
}

// IsExpired returns a bool value that indicates whether the item
// is expired.
func (item *StringBytesItem) IsExpired() bool {
	item.mu.RLock()
	defer item.mu.RUnlock()

	return item.isExpiredUnsafe()
}

func (item *StringBytesItem) isExpiredUnsafe() bool {
	if item.ttl <= 0 {
		return false
	}

	return item.expiresAt.Before(time.Now())
}

// Key returns the key of the item.
func (item *StringBytesItem) Key() string {
	item.mu.RLock()
	defer item.mu.RUnlock()

	return item.key
}

// Value returns the value of the item.
func (item *StringBytesItem) Value() []byte {
	item.mu.RLock()
	defer item.mu.RUnlock()

	return item.value
}

// TTL returns the TTL value of the item.
func (item *StringBytesItem) TTL() time.Duration {
	item.mu.RLock()
	defer item.mu.RUnlock()

	return item.ttl
}

// ExpiresAt returns the expiration timestamp of the item.
func (item *StringBytesItem) ExpiresAt() time.Time {
	item.mu.RLock()
	defer item.mu.RUnlock()

	return item.expiresAt
}

// StringBytesCache is a synchronised map of items that are automatically
// removed when they expire or the capacity is reached.
type StringBytesCache struct {
	mu       sync.RWMutex
	values   map[string]*list.Element
	lru      *list.List
	capacity uint64
	timerCh  chan time.Duration
	stopCh   chan struct{}

	eventsMu       sync.RWMutex
	nextID         uint64
	insertionFns   map[uint64]func(*StringBytesItem)
	evictionFns    map[uint64]func(EvictionReason, *StringBytesItem)
	stopOnce       sync.Once
	nextExpiration time.Time
}

// NewStringBytes creates a new cache instance bounded to the given capacity,
// 0 means unbounded. It replaces New(WithCapacity[string, []byte](capacity)).
func NewStringBytes(capacity uint64) *StringBytesCache {
	return &StringBytesCache{
		values:       make(map[string]*list.Element),
		lru:          list.New(),
		capacity:     capacity,
		timerCh:      make(chan time.Duration, 1), // buffer is important
		stopCh:       make(chan struct{}),
		insertionFns: make(map[uint64]func(*StringBytesItem)),
		evictionFns:  make(map[uint64]func(EvictionReason, *StringBytesItem)),
	}
}

// notifyExpiration wakes the auto cleaner up when an item expires sooner
// than the one it currently waits for.
// Not safe for concurrent use by multiple goroutines without additional
// locking.
func (c *StringBytesCache) notifyExpiration(item *StringBytesItem) {
	if item.ttl <= 0 {
		return
	}

	if !c.nextExpiration.IsZero() && !item.expiresAt.Before(c.nextExpiration) {
		return
	}

	c.nextExpiration = item.expiresAt
	d := time.Until(item.expiresAt)
	if d <= 0 {
		d = time.Microsecond
	}

	select {
	case c.timerCh <- d:
	default:
	}
}

func (c *StringBytesCache) set(key string, value []byte, ttl time.Duration) *StringBytesItem {
	if ttl == PreviousOrDefaultTTL {
		ttl = DefaultTTL
	}

	elem := c.get(key, false, true)
	if elem != nil {
		item := elem.Value.(*StringBytesItem)
		item.update(value, ttl)
		c.notifyExpiration(item)

		return item
	}

	if c.capacity != 0 && uint64(len(c.values)) >= c.capacity {
		c.evict(EvictionReasonCapacityReached, c.lru.Back())
	}

	item := newItem(key, value, ttl)
	c.values[key] = c.lru.PushFront(item)
	c.notifyExpiration(item)

	c.eventsMu.RLock()
	for _, fn := range c.insertionFns {
		fn(item)
	}
	c.eventsMu.RUnlock()

	return item
}

func (c *StringBytesCache) get(key string, touch bool, includeExpired bool) *list.Element {
	elem := c.values[key]
	if elem == nil {
		return nil
	}

	item := elem.Value.(*StringBytesItem)
	if !includeExpired && item.isExpiredUnsafe() {
		return nil
	}

	c.lru.MoveToFront(elem)

	if touch && item.ttl > 0 {
		item.touch()
		c.notifyExpiration(item)
	}

	return elem
}

// evict deletes the given elements, or every element when none is given,
// and calls the eviction hooks.
// Not safe for concurrent use by multiple goroutines without additional
// locking.
func (c *StringBytesCache) evict(reason EvictionReason, elems ...*list.Element) {
	c.eventsMu.RLock()
	defer c.eventsMu.RUnlock()

	if len(elems) == 0 {
		for _, elem := range c.values {
			item := elem.Value.(*StringBytesItem)
			for _, fn := range c.evictionFns {
				fn(reason, item)
			}
		}

		c.values = make(map[string]*list.Element)
		c.lru.Init()

		return
	}

	for _, elem := range elems {
		if elem == nil {
			continue
		}

		item := elem.Value.(*StringBytesItem)
		delete(c.values, item.key)
		c.lru.Remove(elem)

		for _, fn := range c.evictionFns {
			fn(reason, item)
		}
	}
}

// Set creates a new item from the provided key and value, adds
// it to the cache and then returns it. If an item associated with the
// provided key already exists, the new item overwrites the existing one.
// NoTTL constant or -1 can be used to indicate that the item should never
// expire.
func (c *StringBytesCache) Set(key string, value []byte, ttl time.Duration) *StringBytesItem {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.set(key, value, ttl)
}

// Get retrieves an item from the cache by the provided key.
// The item's expiration is extended (touch on hit) like upstream.
// It returns nil if the item is not found or is expired.
func (c *StringBytesCache) Get(key string) *StringBytesItem {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem := c.get(key, true, false)
	if elem == nil {
		return nil
	}

	return elem.Value.(*StringBytesItem)
}

// Delete deletes an item from the cache. If the item associated with
// the key is not found, the method is no-op.
func (c *StringBytesCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem := c.values[key]; elem != nil {
		c.evict(EvictionReasonDeleted, elem)
	}
}

// DeleteAll deletes all items from the cache.
func (c *StringBytesCache) DeleteAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.evict(EvictionReasonDeleted)
}

// DeleteExpired deletes all expired items from the cache.
func (c *StringBytesCache) DeleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextExpiration = time.Time{}

	var expired []*list.Element
	for elem := c.lru.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*StringBytesItem)
		if item.isExpiredUnsafe() {
			expired = append(expired, elem)
		} else if item.ttl > 0 && (c.nextExpiration.IsZero() || item.expiresAt.Before(c.nextExpiration)) {
			c.nextExpiration = item.expiresAt
		}
	}

	if len(expired) > 0 {
		c.evict(EvictionReasonExpired, expired...)
	}
}

// Keys returns all unexpired keys in the cache.
func (c *StringBytesCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]string, 0, len(c.values))
	for k, elem := range c.values {
		if !elem.Value.(*StringBytesItem).isExpiredUnsafe() {
			res = append(res, k)
		}
	}

	return res
}

// Len returns the number of unexpired items in the cache.
func (c *StringBytesCache) Len() int {
	return len(c.Keys())
}

// Range calls fn for each unexpired item in the cache, from the most to the
// least recently used one. If fn returns false, Range stops the iteration.
func (c *StringBytesCache) Range(fn func(item *StringBytesItem) bool) {
	for _, item := range c.snapshot(false) {
		if !item.IsExpired() && !fn(item) {
			return
		}
	}
}

// RangeBackwards calls fn for each unexpired item in the cache, from the
// least to the most recently used one. If fn returns false, RangeBackwards
// stops the iteration.
func (c *StringBytesCache) RangeBackwards(fn func(item *StringBytesItem) bool) {
	for _, item := range c.snapshot(true) {
		if !item.IsExpired() && !fn(item) {
			return
		}
	}
}

// snapshot copies the items so fn can call the cache without deadlocking,
// upstream releases its lock around each fn call for the same reason.
func (c *StringBytesCache) snapshot(backwards bool) []*StringBytesItem {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make([]*StringBytesItem, 0, c.lru.Len())
	if backwards {
		for elem := c.lru.Back(); elem != nil; elem = elem.Prev() {
			items = append(items, elem.Value.(*StringBytesItem))
		}
	} else {
		for elem := c.lru.Front(); elem != nil; elem = elem.Next() {
			items = append(items, elem.Value.(*StringBytesItem))
		}
	}

	return items
}

// Start starts an automatic cleanup process that periodically deletes
// expired items.
// It blocks until Stop is called.
func (c *StringBytesCache) Start() {
	waitDur := func() time.Duration {
		c.mu.RLock()
		defer c.mu.RUnlock()

		if !c.nextExpiration.IsZero() {
			if d := time.Until(c.nextExpiration); d > 0 {
				return d
			}

			return time.Microsecond
		}

		return time.Hour
	}

	timer := time.NewTimer(waitDur())
	stop := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	defer stop()

	for {
		select {
		case <-c.stopCh:
			return
		case d := <-c.timerCh:
			stop()
			timer.Reset(d)
		case <-timer.C:
			c.DeleteExpired()
			stop()
			timer.Reset(waitDur())
		}
	}
}

// Stop stops the automatic cleanup process.
// It blocks until the cleanup process exits.
func (c *StringBytesCache) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

// OnInsertion adds the provided function to be executed when
// a new item is inserted into the cache. The function is executed
// on a separate goroutine and does not block the flow of the cache
// manager.
// The returned function may be called to delete the subscription function
// from the list of insertion subscribers.
// When the returned function is called, it blocks until all instances of
// the same subscription function return. A context is used to notify the
// subscription function when the returned/deletion function is called.
func (c *StringBytesCache) OnInsertion(fn func(context.Context, *StringBytesItem)) func() {
	var (
		wg          sync.WaitGroup
		ctx, cancel = context.WithCancel(context.Background())
	)

	c.eventsMu.Lock()
	id := c.nextID
	c.insertionFns[id] = func(item *StringBytesItem) {
		wg.Add(1)
		go func() {
			fn(ctx, item)
			wg.Done()
		}()
	}
	c.nextID++
	c.eventsMu.Unlock()

	return func() {
		cancel()

		c.eventsMu.Lock()
		delete(c.insertionFns, id)
		c.eventsMu.Unlock()

		wg.Wait()
	}
}

// OnEviction adds the provided function to be executed when
// an item is evicted/deleted from the cache. The function is executed
// on a separate goroutine and does not block the flow of the cache
// manager.
// The returned function may be called to delete the subscription function
// from the list of eviction subscribers.
// When the returned function is called, it blocks until all instances of
// the same subscription function return. A context is used to notify the
// subscription function when the returned/deletion function is called.
func (c *StringBytesCache) OnEviction(fn func(context.Context, EvictionReason, *StringBytesItem)) func() {
	var (
		wg          sync.WaitGroup
		ctx, cancel = context.WithCancel(context.Background())
	)

	c.eventsMu.Lock()
	id := c.nextID
	c.evictionFns[id] = func(r EvictionReason, item *StringBytesItem) {
		wg.Add(1)
		go func() {
			fn(ctx, r, item)
			wg.Done()
		}()
	}
	c.nextID++
	c.eventsMu.Unlock()

	return func() {
		cancel()

		c.eventsMu.Lock()
		delete(c.evictionFns, id)
		c.eventsMu.Unlock()

		wg.Wait()
	}
}
