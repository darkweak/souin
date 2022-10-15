package providers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/client"
	"github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/query"
	"github.com/darkweak/souin/cache/types"
	t "github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// Olric provider type
type Olric struct {
	*client.Client
	dm            *client.DMap
	stale         time.Duration
	logger        *zap.Logger
	reconnecting  bool
	configuration client.Config
}

// OlricConnectionFactory function create new Olric instance
func OlricConnectionFactory(configuration t.AbstractConfigurationInterface) (types.AbstractReconnectProvider, error) {
	config := client.Config{
		Servers: []string{configuration.GetDefaultCache().GetOlric().URL},
		Client: &config.Client{
			DialTimeout: time.Second,
			KeepAlive:   time.Second,
			MaxConn:     10,
		},
	}
	c, err := client.New(&config)
	if err != nil {
		configuration.GetLogger().Sugar().Errorf("Impossible to connect to Olric, %v", err)
	}

	return &Olric{
		Client:        c,
		dm:            nil,
		stale:         configuration.GetDefaultCache().GetStale(),
		logger:        configuration.GetLogger(),
		configuration: config,
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Olric) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the olric keys while reconnecting.")
		return []string{}
	}
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
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Error("An error occurred while trying to list keys in Olric: %s\n", err)
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
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the olric keys by prefix while reconnecting.")
		return []byte{}
	}
	c, err := provider.dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": "^" + key + "({|$)",
		},
	})
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("An error occurred while trying to retrieve data in Olric: %s\n", err)
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
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the olric key while reconnecting.")
		return []byte{}
	}
	val2, err := provider.dm.Get(key)

	if err != nil {
		if !errors.Is(err, olric.ErrKeyNotFound) && !errors.Is(err, olric.ErrKeyTooLarge) && !provider.reconnecting {
			go provider.Reconnect()
		}
		return []byte{}
	}

	return val2.([]byte)
}

// Set method will store the response in Olric provider
func (provider *Olric) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the olric value while reconnecting.")
		return fmt.Errorf("reconnecting error")
	}
	if duration == 0 {
		duration = url.TTL.Duration
	}

	if err := provider.dm.PutEx(key, value, duration); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Olric, %v", err)
		return err
	}

	if err := provider.dm.PutEx(stalePrefix+key, value, provider.stale+duration); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Olric, %v", err)
	}

	return nil
}

// Delete method will delete the response in Olric provider if exists corresponding to key param
func (provider *Olric) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the olric key while reconnecting.")
		return
	}
	go func() {
		err := provider.dm.Delete(key)
		if err != nil {
			provider.logger.Sugar().Errorf("Impossible to delete value into Olric, %v", err)
		}
	}()
}

// DeleteMany method will delete the responses in Olric provider if exists corresponding to the regex key param
func (provider *Olric) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the olric keys while reconnecting.")
		return
	}
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
			provider.logger.Sugar().Errorf("Impossible to delete values into Olric, %v", err)
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

func (provider *Olric) Reconnect() {
	provider.reconnecting = true

	if c, err := client.New(&provider.configuration); err == nil && c != nil {
		provider.Client = c
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
