package keysaver

import (
	"github.com/dgraph-io/ristretto/z"
	"sync"
)

// ClearKey contains an array referencing the keys saved in the cache provider
// and implements Mutex to ensure the array didn't get updated during
// inspection or update. It ensure deadlocks or any crash won't happen.
type ClearKey struct {
	keys map[uint64]string
	mu sync.RWMutex
}

// NewClearKey generate a new ClearKey object and returns it
func NewClearKey() *ClearKey {
	ck := &ClearKey{}
	ck.keys = make(map[uint64]string)
	return ck
}

// ListKeys returns the saved keys list to the client.
func (c *ClearKey) ListKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	list := []string{}

	for _, v := range c.keys {
		list = append(list, v)
	}
	return list
}

// AddKey is called to update keys array with non-existing key or replacing
// the existing one.
func (c *ClearKey) AddKey(clearKey string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	hashedKey, _ := z.KeyToHash(clearKey)
	c.keys[hashedKey] = clearKey
}

// DelKey is called to delete a key if exists, it does nothing otherwise
func (c *ClearKey) DelKey(clearKey string, hashedKey uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if clearKey != "" {
		hashedKey, _ = z.KeyToHash(clearKey)
	}
	delete(c.keys, hashedKey)
}
