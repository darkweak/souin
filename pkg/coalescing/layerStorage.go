package coalescing

import (
	"github.com/dgraph-io/ristretto"
)

// CoalescingLayerStorage is the layer to be able to not coalesce uncoalesceable request
type CoalescingLayerStorage struct {
	*ristretto.Cache
}

// InitializeCoalescingLayerStorage initialize the storage
func InitializeCoalescingLayerStorage() *CoalescingLayerStorage {
	storage, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	return &CoalescingLayerStorage{Cache: storage}
}

// Exists method returns if the key should coalesce
func (provider *CoalescingLayerStorage) Exists(key string) bool {
	_, found := provider.Cache.Get(key)
	return !found
}

// Set method will store the response in Ristretto provider
func (provider *CoalescingLayerStorage) Set(key string) {
	isSet := provider.Cache.Set(key, nil, 1)
	if !isSet {
		panic("Impossible to set value into Ristretto")
	}
}

// Delete method will delete the response in Ristretto provider if exists corresponding to key param
func (provider *CoalescingLayerStorage) Delete(key string) {
	go func() {
		provider.Cache.Del(key)
	}()
}

// Destruct method will shutdown properly the provider
func (provider *CoalescingLayerStorage) Destruct() error {
	provider.Cache.Close()

	return nil
}
