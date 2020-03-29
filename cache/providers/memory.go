package providers

import (
	"github.com/allegro/bigcache"
	"github.com/darkweak/souin/cache/types"
	"os"
	"time"
)

// Memory provider type
type Memory struct {
	*bigcache.BigCache
}

// MemoryConnectionFactory function create new Memory instance
func MemoryConnectionFactory() *Memory {
	t, _ := time.ParseDuration(os.Getenv("TTL"))
	bc, _ := bigcache.NewBigCache(bigcache.DefaultConfig(t * time.Second))
	return &Memory{
		bc,
	}
}

// GetRequestInCache method returns the populated response if exists, empty response then
func (provider *Memory) GetRequestInCache(key string) types.ReverseResponse {
	val2, err := provider.Get(key)

	if err != nil {
		return types.ReverseResponse{Response: "", Proxy: nil, Request: nil}
	}

	return types.ReverseResponse{Response: string(val2), Proxy: nil, Request: nil}
}

// SetRequestInCache method will store the response in Memory provider
func (provider *Memory) SetRequestInCache(key string, value []byte) {
	err := provider.Set(key, value)
	if err != nil {
		panic(err)
	}
}

// DeleteRequestInCache method will delete the response in Memory provider if exists corresponding to key param
func (provider *Memory) DeleteRequestInCache(key string) {
	provider.Delete(key)
}

// DeleteManyRequestInCache method will delete the response in Memory provider if exists corresponding to regex param
func (provider *Memory) DeleteManyRequestInCache(regex string) {
	provider.Delete(regex)
}

// Init method will
func (provider *Memory) Init() {}
