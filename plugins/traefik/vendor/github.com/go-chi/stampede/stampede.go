package stampede

import (
	"context"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-chi/stampede/singleflight"
	lru "github.com/hashicorp/golang-lru"
)

// Prevents cache stampede https://en.wikipedia.org/wiki/Cache_stampede by only running a
// single data fetch operation per expired / missing key regardless of number of requests to that key.

func NewCache(size int, freshFor, ttl time.Duration) *Cache {
	values, _ := lru.New(size)
	return &Cache{
		freshFor: freshFor,
		ttl:      ttl,
		values:   values,
	}
}

type Cache struct {
	values *lru.Cache

	freshFor time.Duration
	ttl      time.Duration

	mu        sync.RWMutex
	callGroup singleflight.Group
}

func (c *Cache) Get(ctx context.Context, key interface{}, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	return c.get(ctx, key, false, fn)
}

func (c *Cache) GetFresh(ctx context.Context, key interface{}, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	return c.get(ctx, key, true, fn)
}

func (c *Cache) Set(ctx context.Context, key interface{}, fn func(ctx context.Context) (interface{}, error)) (interface{}, bool, error) {
	return c.callGroup.Do(ctx, key, c.set(key, fn))
}

func (c *Cache) get(ctx context.Context, key interface{}, freshOnly bool, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	c.mu.RLock()
	v, ok := c.values.Get(key)
	c.mu.RUnlock()

	var val value
	if ok {
		if val, ok = v.(value); !ok {
			panic("stampede: invalid cache value")
		}
	}

	// value exists and is fresh - just return
	if val.IsFresh() {
		return val.Value(), nil
	}

	// value exists and is stale, and we're OK with serving it stale while updating in the background
	if !freshOnly && !val.IsExpired() {
		go c.Set(ctx, key, fn)
		return val.Value(), nil
	}

	// value doesn't exist or is expired, or is stale and we need it fresh - sync update
	v, _, err := c.Set(ctx, key, fn)
	return v, err
}

func (c *Cache) set(key interface{}, fn singleflight.DoFunc) singleflight.DoFunc {
	return singleflight.DoFunc(func(ctx context.Context) (interface{}, error) {
		val, err := fn(ctx)
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.values.Add(key, value{
			v:          val,
			expiry:     time.Now().Add(c.ttl),
			bestBefore: time.Now().Add(c.freshFor),
		})
		c.mu.Unlock()

		return val, nil
	})
}

type value struct {
	v interface{}

	bestBefore time.Time // cache entry freshness cutoff
	expiry     time.Time // cache entry time to live cutoff
}

func (v *value) IsFresh() bool {
	if v == nil {
		return false
	}
	return v.bestBefore.After(time.Now())
}

func (v *value) IsExpired() bool {
	if v == nil {
		return true
	}
	return v.expiry.Before(time.Now())
}

func (v *value) Value() interface{} {
	return v.v
}

func BytesToHash(b ...[]byte) uint64 {
	d := xxhash.New()
	for _, v := range b {
		d.Write(v)
	}
	return d.Sum64()
}

func StringToHash(s ...string) uint64 {
	d := xxhash.New()
	for _, v := range s {
		d.WriteString(v)
	}
	return d.Sum64()
}
