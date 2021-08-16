package providers

import (
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/patrickmn/go-cache"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Badger provider type
type Badger struct {
	*cache.Cache
}

// BadgerConnectionFactory function create new Badger instance
func BadgerConnectionFactory(_ t.AbstractConfigurationInterface) (*Badger, error) {
	c := cache.New(1*time.Second, 2*time.Second)
	return &Badger{c}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Badger) ListKeys() []string {
	keys := []string{}

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Badger) Get(key string) []byte {
	result, found := provider.Cache.Get(key)

	if !found {
		return []byte{}
	}

	return result.([]byte)
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Badger) Prefix(key string, req *http.Request) []byte {
	var result []byte

	for k, v := range provider.Items() {
		if k == key {
			return v.Object.([]byte)
		}

		if !strings.HasPrefix(key, k) {
			continue
		}

		if varyVoter(key, req, k) {
			result = v.Object.([]byte)
		}
	}

	return result
}

// Set method will store the response in Badger provider
func (provider *Badger) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	provider.Cache.Set(key, value, duration)
}

// Delete method will delete the response in Badger provider if exists corresponding to key param
func (provider *Badger) Delete(key string) {
	provider.Cache.Delete(key)
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param
func (provider *Badger) DeleteMany(key string) {
	re, e := regexp.Compile(key)

	if e != nil {
		return
	}

	for k := range provider.Items() {
		if re.MatchString(k) {
			provider.Delete(k)
		}
	}
}

// Init method will
func (provider *Badger) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Badger) Reset() {
	provider.Cache.Flush()
}
