package providers

import (
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"strconv"
	"github.com/allegro/bigcache"
)

// Memory provider type
type Memory struct {
	*bigcache.BigCache
}

// MemoryConnectionFactory function create new Memory instance
func MemoryConnectionFactory(configuration configuration.Configuration) Memory {
	t, _ := strconv.Atoi(configuration.DefaultCache.TTL)
	bc, _ := bigcache.NewBigCache(bigcache.DefaultConfig(time.Second * time.Duration(t)))
	return Memory{
		bc,
	}
}

// GetRequestInCache method returns the populated response if exists, empty response then
func (provider Memory) GetRequestInCache(key string) types.ReverseResponse {
	val2, err := provider.Get(key)

	if err != nil {
		return types.ReverseResponse{Response: "", Proxy: nil, Request: nil}
	}

	return types.ReverseResponse{Response: string(val2), Proxy: nil, Request: nil}
}

// SetRequestInCache method will store the response in Memory provider
func (provider Memory) SetRequestInCache(key string, value []byte, _ configuration.URL) {
	err := provider.Set(key, value)
	if err != nil {
		panic(err)
	}
}

// DeleteRequestInCache method will delete the response in Memory provider if exists corresponding to key param
func (provider Memory) DeleteRequestInCache(key string) {
	provider.Delete(key)
}

// Init method will
func (provider Memory) Init() error {
	return nil
}
