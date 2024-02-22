package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
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
func OlricConnectionFactory(configuration t.AbstractConfigurationInterface) (types.Storer, error) {
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

// Name returns the storer name
func (provider *Olric) Name() string {
	return "OLRIC"
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
		if !strings.Contains(records.Key(), surrogatePrefix) {
			keys = append(keys, records.Key())
		}
	}
	records.Close()

	return keys
}

// MapKeys method returns the map of existing keys
func (provider *Olric) MapKeys(prefix string) map[string]string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the olric keys while reconnecting.")
		return map[string]string{}
	}
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	records, err := dm.Scan(context.Background())
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Error("An error occurred while trying to list keys in Olric: %s\n", err)
		return map[string]string{}
	}

	keys := map[string]string{}
	for records.Next() {
		if strings.HasPrefix(records.Key(), prefix) {
			k, _ := strings.CutPrefix(records.Key(), prefix)
			keys[k] = string(provider.Get(records.Key()))
		}
	}
	records.Close()

	return keys
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Olric) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	var resultFresh *http.Response
	var resultStale *http.Response

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	res, e := dm.Get(context.Background(), key)

	if e != nil {
		return resultFresh, resultStale
	}

	val, _ := res.Byte()
	resultFresh, resultStale, _ = mappingElection(provider, val, req, validator, provider.logger)

	return resultFresh, resultStale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Olric) SetMultiLevel(baseKey, key string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error {
	now := time.Now()

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	if err := dm.Put(context.Background(), key, value, olric.EX(duration)); err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into EmbeddedOlric, %v", err)
		return err
	}

	mappingKey := mappingKeyPrefix + baseKey
	res, e := dm.Get(context.Background(), mappingKey)
	if e != nil && !errors.Is(e, olric.ErrKeyNotFound) {
		provider.logger.Sugar().Errorf("Impossible to get the key %s EmbeddedOlric, %v", baseKey, e)
		return nil
	}

	val, e := res.Byte()
	if e != nil {
		provider.logger.Sugar().Errorf("Impossible to parse the key %s value as byte, %v", baseKey, e)
		return e
	}

	val, e = mappingUpdater(key, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag)
	if e != nil {
		return e
	}

	return provider.Set(mappingKey, val, t.URL{}, time.Hour)
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Olric) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the olric keys by prefix while reconnecting.")
		return nil
	}
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	records, err := dm.Scan(context.Background(), olric.Match("^"+key+"({|$)"))
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("An error occurred while trying to retrieve data in Olric: %s\n", err)
		return nil
	}

	for records.Next() {
		if varyVoter(key, req, records.Key()) {
			if val := provider.Get(records.Key()); val != nil {
				if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(val)), req); err == nil {
					rfc.ValidateETag(res, validator)
					if validator.Matched {
						provider.logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v", records.Key(), validator)
						return res
					}

					provider.logger.Sugar().Debugf("The stored key %s didn't match the current iteration key ETag %+v", records.Key(), validator)
				} else {
					provider.logger.Sugar().Errorf("An error occured while reading response for the key %s: %v", records.Key(), err)
				}
			}
		}
	}
	records.Close()

	return nil
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

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)
	if err := dm.Put(context.Background(), key, value, olric.EX(duration)); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Olric, %v", err)
		return err
	}

	if err := dm.Put(context.Background(), StalePrefix+key, value, olric.EX(provider.stale+duration)); err != nil {
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
