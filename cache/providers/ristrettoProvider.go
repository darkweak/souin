package providers

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"github.com/dgraph-io/ristretto"
	"time"
	"strconv"
)

// Ristretto provider type
type Ristretto struct {
	*ristretto.Cache
}

// RistrettoConnectionFactory function create new Ristretto instance
func RistrettoConnectionFactory(_ configuration.AbstractConfigurationInterface) *Ristretto {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	if err != nil {
		panic(err)
	}

	return &Ristretto{cache}
}

// GetRequestInCache method returns the populated response if exists, empty response then
func (provider *Ristretto) GetRequestInCache(key string) types.ReverseResponse {
	val, found := provider.Get(key)

	if !found {
		return types.ReverseResponse{Response: "", Proxy: nil, Request: nil}
	}

	return types.ReverseResponse{Response:string(val.([]byte)), Proxy: nil, Request: nil}
}

// SetRequestInCache method will store the response in Ristretto provider
func (provider *Ristretto) SetRequestInCache(key string, value []byte, url configuration.URL) {
	ttl, _ := strconv.Atoi(url.TTL)
	isSet := provider.SetWithTTL(key, value, 1, time.Duration(ttl)*time.Second)
	if !isSet {
		panic("Impossible to set into Ristretto")
	}
}

// DeleteRequestInCache method will delete the response in Ristretto provider if exists corresponding to key param
func (provider *Ristretto) DeleteRequestInCache(key string) {
	provider.Del(key)
}

// Init method will
func (provider *Ristretto) Init() error {
	return nil
}
