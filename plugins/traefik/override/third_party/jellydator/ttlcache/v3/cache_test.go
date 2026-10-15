package ttlcache

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"
)

type evictions struct {
	mu      sync.Mutex
	reasons map[string]EvictionReason
	wg      sync.WaitGroup
}

func watchEvictions(c *StringBytesCache, expected int) *evictions {
	e := &evictions{reasons: map[string]EvictionReason{}}
	e.wg.Add(expected)
	c.OnEviction(func(_ context.Context, reason EvictionReason, item *StringBytesItem) {
		e.mu.Lock()
		e.reasons[item.Key()] = reason
		e.mu.Unlock()
		e.wg.Done()
	})

	return e
}

func Test_Cache_SetGet(t *testing.T) {
	c := NewStringBytes(0)
	c.Set("a", []byte("1"), NoTTL)

	if item := c.Get("a"); item == nil || string(item.Value()) != "1" || item.Key() != "a" {
		t.Errorf("unexpected item %+v", item)
	}

	if c.Get("missing") != nil {
		t.Error("a missing key must return nil")
	}
}

func Test_Cache_TouchOnHit(t *testing.T) {
	c := NewStringBytes(0)
	c.Set("a", []byte("1"), 50*time.Millisecond)

	time.Sleep(30 * time.Millisecond)
	if c.Get("a") == nil {
		t.Fatal("the item must still be fresh")
	}

	// Like upstream, the hit extended the expiration.
	time.Sleep(30 * time.Millisecond)
	if c.Get("a") == nil {
		t.Error("the hit must have extended the expiration")
	}

	time.Sleep(60 * time.Millisecond)
	if c.Get("a") != nil {
		t.Error("the item must be expired")
	}
}

func Test_Cache_CapacityEvictsLeastRecentlyUsed(t *testing.T) {
	c := NewStringBytes(2)
	e := watchEvictions(c, 1)

	c.Set("a", []byte("1"), NoTTL)
	c.Set("b", []byte("2"), NoTTL)
	_ = c.Get("a")
	c.Set("c", []byte("3"), NoTTL)
	e.wg.Wait()

	if e.reasons["b"] != EvictionReasonCapacityReached {
		t.Errorf("the least recently used item must be evicted, got %v", e.reasons)
	}

	keys := c.Keys()
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "c" {
		t.Errorf("unexpected keys %v", keys)
	}
}

func Test_Cache_StartEvictsExpired(t *testing.T) {
	c := NewStringBytes(0)
	e := watchEvictions(c, 1)
	go c.Start()
	defer c.Stop()

	c.Set("a", []byte("1"), 20*time.Millisecond)
	c.Set("b", []byte("2"), NoTTL)
	e.wg.Wait()

	if e.reasons["a"] != EvictionReasonExpired {
		t.Errorf("the expired item must be evicted, got %v", e.reasons)
	}

	if len(c.Keys()) != 1 {
		t.Errorf("unexpected keys %v", c.Keys())
	}
}

func Test_Cache_DeleteAndRangeOrder(t *testing.T) {
	c := NewStringBytes(0)
	e := watchEvictions(c, 3)
	c.Set("a", []byte("1"), NoTTL)
	c.Set("b", []byte("2"), NoTTL)
	c.Set("c", []byte("3"), NoTTL)

	forward := []string{}
	c.Range(func(item *StringBytesItem) bool {
		forward = append(forward, item.Key())

		return true
	})

	backward := []string{}
	c.RangeBackwards(func(item *StringBytesItem) bool {
		backward = append(backward, item.Key())

		return true
	})

	if len(forward) != 3 || forward[0] != "c" || backward[0] != "a" {
		t.Errorf("unexpected order forward %v backward %v", forward, backward)
	}

	c.Delete("a")
	c.DeleteAll()
	e.wg.Wait()

	if len(c.Keys()) != 0 || len(e.reasons) != 3 || e.reasons["a"] != EvictionReasonDeleted {
		t.Errorf("unexpected state keys %v evictions %v", c.Keys(), e.reasons)
	}
}
