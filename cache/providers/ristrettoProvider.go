package providers

import (
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
	"strconv"
	"time"
)

// Ristretto provider type
type Ristretto struct {
	*ristretto.Cache
}

// RistrettoConnectionFactory function create new Ristretto instance
func RistrettoConnectionFactory(_ t.AbstractConfigurationInterface) (*Ristretto, error) {
	cache, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})

	return &Ristretto{cache}, nil
}

// Get method returns the populated response if exists, empty response then
func (provider *Ristretto) Get(key string) []byte {
	val, found := provider.Cache.Get(key)
	if !found {
		return []byte{}
	}
	return val.([]byte)
}

// Set method will store the response in Ristretto provider
func (provider *Ristretto) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		ttl, _ := strconv.Atoi(url.TTL)
		duration = time.Duration(ttl)*time.Second
	}
	isSet := provider.SetWithTTL(key, value, 1, duration)
	if !isSet {
		panic("Impossible to set value into Ristretto")
	}
}

// Delete method will delete the response in Ristretto provider if exists corresponding to key param
func (provider *Ristretto) Delete(key string) {
	provider.Del(key)
}

// Init method will
func (provider *Ristretto) Init() error {
	return nil
}
