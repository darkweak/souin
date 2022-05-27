package providers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/buraksezer/olric/client"
	"github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/query"
	t "github.com/darkweak/souin/configurationtypes"
)

// Olric provider type
type Olric struct {
	*client.Client
	dm    *client.DMap
	stale time.Duration
}

// OlricConnectionFactory function create new Olric instance
func OlricConnectionFactory(configuration t.AbstractConfigurationInterface) (*Olric, error) {
	c, err := client.New(&client.Config{
		Servers: []string{configuration.GetDefaultCache().GetOlric().URL},
		Client: &config.Client{
			DialTimeout: time.Second,
			KeepAlive:   time.Second,
			MaxConn:     10,
		},
	})
	if err != nil {
		panic(err)
	}

	return &Olric{
		Client: c,
		dm:     nil,
		stale:  configuration.GetDefaultCache().GetStale(),
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Olric) ListKeys() []string {
	c, err := provider.dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": "",
			"$options": query.M{
				"$onValue": query.M{
					"$ignore": true,
				},
			},
		},
	})
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		fmt.Printf("An error occurred while trying to list keys in Olric: %s\n", err)
		return []string{}
	}

	keys := []string{}
	_ = c.Range(func(key string, _ interface{}) bool {
		keys = append(keys, key)
		return true
	})

	return keys
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Olric) Prefix(key string, req *http.Request) []byte {
	c, err := provider.dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": "^" + key,
		},
	})
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		fmt.Printf("An error occurred while trying to retrieve data in Olric: %s\n", err)
		return []byte{}
	}

	res := []byte{}
	_ = c.Range(func(k string, v interface{}) bool {
		if varyVoter(key, req, k) {
			res = v.([]byte)
			return false
		}

		return true
	})

	return res
}

// Get method returns the populated response if exists, empty response then
func (provider *Olric) Get(key string) []byte {
	val2, err := provider.dm.Get(key)

	if err != nil {
		return []byte{}
	}

	return val2.([]byte)
}

// Set method will store the response in Olric provider
func (provider *Olric) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	if err := provider.dm.PutEx(key, value, duration); err != nil {
		panic(err)
	}

	if err := provider.dm.PutEx(stalePrefix+key, value, provider.stale+duration); err != nil {
		panic(err)
	}
}

// Delete method will delete the response in Olric provider if exists corresponding to key param
func (provider *Olric) Delete(key string) {
	go func() {
		err := provider.dm.Delete(key)
		if err != nil {
			panic(err)
		}
	}()
}

// DeleteMany method will delete the responses in Olric provider if exists corresponding to the regex key param
func (provider *Olric) DeleteMany(key string) {
	go func() {
		c, err := provider.dm.Query(query.M{
			"$onKey": query.M{
				"$regexMatch": key,
				"$options": query.M{
					"$onValue": query.M{
						"$ignore": true,
					},
				},
			},
		})

		if c == nil || err != nil {
			return
		}

		err = c.Range(func(key string, _ interface{}) bool {
			provider.Delete(key)
			return true
		})

		if err != nil {
			panic(err)
		}
	}()
}

// Init method will initialize Olric provider if needed
func (provider *Olric) Init() error {
	dm := provider.Client.NewDMap("souin-map")

	provider.dm = dm
	return nil
}

// Reset method will reset or close provider
func (provider *Olric) Reset() error {
	provider.Client.Close()

	return nil
}
