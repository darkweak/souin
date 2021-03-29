package providers

import (
	"github.com/darkweak/souin/cache/keysaver"
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
	"strconv"
	"time"
)

// Ristretto provider type
type Ristretto struct {
	*ristretto.Cache
	keySaver *keysaver.ClearKey
}

// RistrettoConnectionFactory function create new Ristretto instance
func RistrettoConnectionFactory(c t.AbstractConfigurationInterface) (*Ristretto, error) {
	ristrettoConfig := &ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	}

	var keySaver *keysaver.ClearKey
	if c.GetAPI().Souin.Enable {
		keySaver = keysaver.NewClearKey()
		ristrettoConfig.OnEvict = func(key uint64, u2 uint64, i interface{}, i2 int64) {
			go func() {
				keySaver.DelKey("", key)
			}()
		}
	}
	cache, _ := ristretto.NewCache(ristrettoConfig)

	return &Ristretto{cache, keySaver}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Ristretto) ListKeys() []string {
	if nil != provider.keySaver {
		return provider.keySaver.ListKeys()
	}
	return []string{}
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
		duration = time.Duration(ttl) * time.Second
	}
	isSet := provider.SetWithTTL(key, value, 1, duration)
	if !isSet {
		panic("Impossible to set value into Ristretto")
	} else {
		go func() {
			if nil != provider.keySaver {
				provider.keySaver.AddKey(key)
			}
		}()
	}
}

// Delete method will delete the response in Ristretto provider if exists corresponding to key param
func (provider *Ristretto) Delete(key string) {
	go func() {
		provider.Del(key)
		provider.keySaver.DelKey(key, 0)
	}()
}

// Init method will
func (provider *Ristretto) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Ristretto) Reset() {}
