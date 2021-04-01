package types

import (
	"github.com/dgraph-io/ristretto"
)

type VaryLayerStorage struct {
	*ristretto.Cache
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
