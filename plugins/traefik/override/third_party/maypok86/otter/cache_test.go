package otter

import (
	"testing"
	"time"
)

func Test_StringBytesCache_TTL(t *testing.T) {
	c, err := NewStringBytesCache(10)
	if err != nil {
		t.Fatal(err)
	}

	c.Set("fresh", []byte("value"), time.Minute)
	if v, ok := c.Get("fresh"); !ok || string(v) != "value" {
		t.Errorf("unexpected fresh entry %q %v", v, ok)
	}

	// Like upstream, a zero TTL is already expired.
	c.Set("zero", []byte("value"), 0)
	if _, ok := c.Get("zero"); ok {
		t.Error("the zero TTL entry must be expired")
	}

	// The otter storage relies on a negative TTL wrapping around to never expire.
	c.Set("negative", []byte("value"), -2*time.Hour)
	if _, ok := c.Get("negative"); !ok {
		t.Error("the negative TTL entry must never expire")
	}
}

func Test_StringBytesCache_Capacity(t *testing.T) {
	c, _ := NewStringBytesCache(2)
	c.Set("a", []byte("a"), time.Minute)
	c.Set("b", []byte("b"), time.Minute)
	c.Set("a", []byte("a2"), time.Minute)
	c.Set("c", []byte("c"), time.Minute)

	if c.Size() != 2 {
		t.Errorf("the size must be bounded by the capacity, got %d", c.Size())
	}

	if _, ok := c.Get("b"); ok {
		t.Error("the oldest inserted entry must be evicted first")
	}

	if v, _ := c.Get("a"); string(v) != "a2" {
		t.Errorf("unexpected updated value %q", v)
	}
}

func Test_StringBytesCache_SharedCopies(t *testing.T) {
	c, _ := NewStringBytesCache(10)
	cp := c
	cp.Set("a", []byte("a"), time.Minute)

	if _, ok := c.Get("a"); !ok {
		t.Error("copies must share the same storage")
	}
}

func Test_StringBytesCache_RangeAndDelete(t *testing.T) {
	c, _ := NewStringBytesCache(10)
	c.Set("keep", []byte("1"), time.Minute)
	c.Set("drop-1", []byte("2"), time.Minute)
	c.Set("drop-2", []byte("3"), time.Minute)
	c.Set("expired", []byte("4"), 0)

	c.DeleteByFunc(func(key string, _ []byte) bool {
		return key != "keep"
	})

	keys := []string{}
	c.Range(func(key string, _ []byte) bool {
		keys = append(keys, key)

		return true
	})

	if len(keys) != 1 || keys[0] != "keep" {
		t.Errorf("unexpected keys %v", keys)
	}

	c.Clear()
	if c.Size() != 0 {
		t.Error("the cache must be empty after Clear")
	}
}
