package keysaver

import (
	"github.com/dgraph-io/ristretto/z"
	"reflect"
	"sync"
	"time"
)

// ADD to add
const ADD = "add"

// DELETE to delete
const DELETE = "delete"

type payload struct {
	updateType string
	clearKey   string
	hashedKey  uint64
}

// ClearKey contains an array referencing the keys saved in the cache provider
// and implements Mutex to ensure the array didn't get updated during
// inspection or update. It ensure deadlocks or any crash won't happen.
type ClearKey struct {
	keys           map[uint64]string
	mu             sync.RWMutex
	payloadChannel chan payload
}

func (c *ClearKey) startKeysUpdater() {
	for {
		payload, _ := <-c.payloadChannel
		if reflect.ValueOf(&c.mu).Elem().FieldByName("readerCount").Int() > 0 {
			time.Sleep(20 * time.Millisecond)
			c.payloadChannel <- payload
		}
		c.mu.RLock()
		switch payload.updateType {
		case ADD:
			c.keys[payload.hashedKey] = payload.clearKey
			break
		case DELETE:
			delete(c.keys, payload.hashedKey)
			break
		}
		c.mu.RUnlock()
	}
}

// NewClearKey generate a new ClearKey object and returns it
func NewClearKey() *ClearKey {
	ck := &ClearKey{}
	ck.keys = make(map[uint64]string)
	ck.payloadChannel = make(chan payload)
	go ck.startKeysUpdater()
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
	hashedKey, _ := z.KeyToHash(clearKey)
	c.payloadChannel <- payload{
		ADD,
		clearKey,
		hashedKey,
	}
}

// DelKey is called to delete a key if exists, it does nothing otherwise
func (c *ClearKey) DelKey(clearKey string, hashedKey uint64) {
	if clearKey != "" {
		hashedKey, _ = z.KeyToHash(clearKey)
	}
	c.payloadChannel <- payload{
		DELETE,
		clearKey,
		hashedKey,
	}
}
