package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/darkweak/souin/cache/types"
	t "github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// Olric provider type
type Olric struct {
	*olric.ClusterClient
	dm            *sync.Pool
	stale         time.Duration
	logger        *zap.Logger
	addresses     []string
	reconnecting  bool
	configuration config.Client
}

// OlricConnectionFactory function create new Olric instance
func OlricConnectionFactory(configuration t.AbstractConfigurationInterface) (types.AbstractReconnectProvider, error) {
	c, err := olric.NewClusterClient([]string{configuration.GetDefaultCache().GetOlric().URL})
	if err != nil {
		configuration.GetLogger().Sugar().Errorf("Impossible to connect to Olric, %v", err)
	}

	return &Olric{
		ClusterClient: c,
		dm:            nil,
		stale:         configuration.GetDefaultCache().GetStale(),
		logger:        configuration.GetLogger(),
		configuration: config.Client{},
		addresses:     []string{configuration.GetDefaultCache().GetOlric().URL},
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Olric) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the olric keys while reconnecting.")
		return []string{}
	}
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	records, err := dm.Scan(context.Background())
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Error("An error occurred while trying to list keys in Olric: %s\n", err)
		return []string{}
	}

	keys := []string{}
	for records.Next() {
		keys = append(keys, records.Key())
	}
	records.Close()

	return keys
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Olric) Prefix(key string, req *http.Request) []byte {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the olric keys by prefix while reconnecting.")
		return []byte{}
	}
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	records, err := dm.Scan(context.Background(), olric.Match("^"+key+"({|$)"))
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("An error occurred while trying to retrieve data in Olric: %s\n", err)
		return []byte{}
	}

	for records.Next() {
		if varyVoter(key, req, records.Key()) {
			return provider.Get(records.Key())
		}
	}
	records.Close()

	return []byte{}
}

// Get method returns the populated response if exists, empty response then
func (provider *Olric) Get(key string) []byte {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the olric key while reconnecting.")
		return []byte{}
	}
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	res, err := dm.Get(context.Background(), key)

	if err != nil {
		if !errors.Is(err, olric.ErrKeyNotFound) && !errors.Is(err, olric.ErrKeyTooLarge) && !provider.reconnecting {
			go provider.Reconnect()
		}
		return []byte{}
	}

	val, _ := res.Byte()
	return val
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

	fmt.Println("STORE IN OLRIC")
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	if err := dm.Put(context.Background(), key, value, olric.EX(duration)); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Olric, %v", err)
		return err
	}

	if err := dm.Put(context.Background(), stalePrefix+key, value, olric.EX(provider.stale+duration)); err != nil {
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
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	_, err := dm.Delete(context.Background(), key)
	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to delete value into Olric, %v", err)
	}
}

// DeleteMany method will delete the responses in Olric provider if exists corresponding to the regex key param
func (provider *Olric) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the olric keys while reconnecting.")
		return
	}

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	records, err := dm.Scan(context.Background(), olric.Match(key))
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Error("An error occurred while trying to list keys in Olric: %s\n", err)
		return
	}

	keys := []string{}
	for records.Next() {
		keys = append(keys, records.Key())
	}
	records.Close()

	_, _ = dm.Delete(context.Background(), keys...)
}

// Init method will initialize Olric provider if needed
func (provider *Olric) Init() error {
	dm := sync.Pool{
		New: func() interface{} {
			dmap, _ := provider.ClusterClient.NewDMap("souin-map")
			return dmap
		},
	}

	provider.dm = &dm
	return nil
}

// Reset method will reset or close provider
func (provider *Olric) Reset() error {
	provider.ClusterClient.Close(context.Background())

	return nil
}

func (provider *Olric) Reconnect() {
	provider.reconnecting = true

	if c, err := olric.NewClusterClient(provider.addresses, olric.WithConfig(&provider.configuration)); err == nil && c != nil {
		provider.ClusterClient = c
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
