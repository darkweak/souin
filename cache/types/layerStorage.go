package types

import (
	"github.com/dgraph-io/ristretto"
)

// VaryLayerStorage is the layer for Vary support storage
type VaryLayerStorage struct {
	*ristretto.Cache
}

// InitializeVaryLayerStorage initialize the storage
func InitializeVaryLayerStorage() *VaryLayerStorage {
	storage, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	return &VaryLayerStorage{Cache: storage}
}

// Get method returns the varied headers list if exists, empty array then
func (provider *VaryLayerStorage) Get(key string) []string {
	val, found := provider.Cache.Get(key)
	if !found {
		return []string{}
	}
	return val.([]string)
}

// Set method will store the response in Ristretto provider
func (provider *VaryLayerStorage) Set(key string, headers []string) {
	isSet := provider.Cache.Set(key, headers, 1)
	if !isSet {
		panic("Impossible to set value into Ristretto")
	}
}

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
		provider.Del(key)
	}()
}
