package olric

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/darkweak/storages/core"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Olric provider type.
type Olric struct {
	olric.Client

	dm            *sync.Pool
	stale         time.Duration
	logger        core.Logger
	addresses     []string
	reconnecting  bool
	configuration config.Client
}

func tryToLoadConfiguration(olricInstance *config.Config, olricConfiguration core.CacheProvider, logger core.Logger) (*config.Config, bool) {
	var err error

	isAlreadyLoaded := false

	if olricConfiguration.Configuration == nil && olricConfiguration.Path != "" {
		if olricInstance, err = config.Load(olricConfiguration.Path); err == nil {
			isAlreadyLoaded = true
		}
	} else if olricConfiguration.Configuration != nil {
		tmpFile := "/tmp/" + uuid.NewString() + ".yml"
		yamlConfig, _ := yaml.Marshal(olricConfiguration.Configuration)

		defer func() {
			if err = os.RemoveAll(tmpFile); err != nil {
				logger.Error("Impossible to remove the temporary file")
			}
		}()

		if err = os.WriteFile(tmpFile, yamlConfig, 0o600); err != nil {
			logger.Error("Impossible to create the embedded Olric config from the given one")
		}

		if olricInstance, err = config.Load(tmpFile); err == nil {
			isAlreadyLoaded = true
		} else {
			logger.Error("Impossible to create the embedded Olric config from the given one")
		}
	}

	return olricInstance, isAlreadyLoaded
}

func newEmbeddedOlric(olricConfiguration core.CacheProvider, logger core.Logger) (*olric.EmbeddedClient, error) {
	var olricInstance *config.Config

	var loaded bool

	if olricInstance, loaded = tryToLoadConfiguration(olricInstance, olricConfiguration, logger); !loaded {
		olricInstance = config.New("local")
		olricInstance.DMaps.MaxInuse = 512 << 20
	}

	started, cancel := context.WithCancel(context.Background())
	olricInstance.Started = func() {
		logger.Error("Embedded Olric is ready")

		defer cancel()
	}

	olricDB, err := olric.New(olricInstance)
	if err != nil {
		return nil, err
	}

	errCh := make(chan error, 1)

	defer func() {
		close(errCh)
	}()

	go func(cdb *olric.Olric) {
		if err = cdb.Start(); err != nil {
			errCh <- err
		}
	}(olricDB)

	select {
	case err = <-errCh:
	case <-started.Done():
	}

	dbClient := olricDB.NewEmbeddedClient()

	logger.Info("Embedded Olric is ready for this node.")

	return dbClient, nil
}

// Factory function create new Olric instance.
func Factory(olricConfiguration core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	if olricConfiguration.URL == "" && olricConfiguration.Configuration != nil {
		if olricCfg, ok := olricConfiguration.Configuration.(map[string]interface{}); ok {
			if mode, found := olricCfg["mode"]; found && mode.(string) == "local" {
				logger.Debug("Olric configuration URL is empty, trying to load olric in embedded mode")

				client, err := newEmbeddedOlric(olricConfiguration, logger)
				if err != nil {
					logger.Error("Impossible to setup Embedded Olric instance")

					return nil, err
				}

				return &Olric{
					Client:        client,
					dm:            nil,
					stale:         stale,
					logger:        logger,
					configuration: config.Client{},
					addresses:     strings.Split(olricConfiguration.URL, ","),
				}, nil
			}
		}
	}

	client, err := olric.NewClusterClient(strings.Split(olricConfiguration.URL, ","))
	if err != nil {
		logger.Errorf("Impossible to connect to Olric, %v", err)
	}

	return &Olric{
		Client:        client,
		dm:            nil,
		stale:         stale,
		logger:        logger,
		configuration: config.Client{},
		addresses:     strings.Split(olricConfiguration.URL, ","),
	}, nil
}

// Name returns the storer name.
func (provider *Olric) Name() string {
	return "OLRIC"
}

// Uuid returns an unique identifier.
func (provider *Olric) Uuid() string {
	return fmt.Sprintf("%s-%s", provider.addresses, provider.stale)
}

// ListKeys method returns the list of existing keys.
func (provider *Olric) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Error("Impossible to list the olric keys while reconnecting.")

		return []string{}
	}

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	records, err := dm.Scan(context.Background(), olric.Match("^"+core.MappingKeyPrefix))
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Error("An error occurred while trying to list keys in Olric: %s\n", err)

		return []string{}
	}

	keys := []string{}

	for records.Next() {
		mapping, err := core.DecodeMapping(provider.Get(records.Key()))
		if err == nil {
			for _, v := range mapping.GetMapping() {
				keys = append(keys, v.GetRealKey())
			}
		}
	}

	records.Close()

	return keys
}

// MapKeys method returns the map of existing keys.
func (provider *Olric) MapKeys(prefix string) map[string]string {
	if provider.reconnecting {
		provider.logger.Error("Impossible to list the olric keys while reconnecting.")

		return map[string]string{}
	}

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	records, err := dm.Scan(context.Background())
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Error("An error occurred while trying to list keys in Olric: %s\n", err)

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
func (provider *Olric) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	res, e := dm.Get(context.Background(), core.MappingKeyPrefix+key)
	if e != nil {
		return fresh, stale
	}

	val, _ := res.Byte()
	fresh, stale, _ = core.MappingElection(provider, val, req, validator, provider.logger)

	return fresh, stale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Olric) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	dmap := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dmap)

	compressed, err := core.CompressValue(value)
	if err != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Olric, %v", variedKey, err)

		return err
	}

	if err := dmap.Put(context.Background(), variedKey, compressed, olric.EX(duration)); err != nil {
		provider.logger.Errorf("Impossible to set value into Olric, %v", err)

		return err
	}

	mappingKey := core.MappingKeyPrefix + baseKey

	// The first variant of a base key finds no mapping yet: start from an
	// empty one.
	var val []byte

	res, err := dmap.Get(context.Background(), mappingKey)

	switch {
	case isOlricError(err, olric.ErrKeyNotFound):
	case err != nil:
		provider.logger.Errorf("Impossible to get the key %s Olric, %v", baseKey, err)

		return err
	default:
		if val, err = res.Byte(); err != nil {
			provider.logger.Errorf("Impossible to parse the key %s value as byte, %v", baseKey, err)

			return err
		}
	}

	val, err = core.MappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if err != nil {
		return err
	}

	// The mapping must outlive the response it now references, otherwise the
	// response becomes unreachable while still stored.
	return provider.Set(mappingKey, val, max(time.Hour, duration+provider.stale))
}

// Get method returns the populated response if exists, empty response then.
func (provider *Olric) Get(key string) []byte {
	if provider.reconnecting {
		provider.logger.Error("Impossible to get the olric key while reconnecting.")

		return []byte{}
	}

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	res, err := dm.Get(context.Background(), key)
	if err != nil {
		if !isOlricError(err, olric.ErrKeyNotFound) && !isOlricError(err, olric.ErrKeyTooLarge) && !provider.reconnecting {
			go provider.Reconnect()
		}

		return []byte{}
	}

	val, _ := res.Byte()

	return val
}

// Set method will store the response in Olric provider.
func (provider *Olric) Set(key string, value []byte, duration time.Duration) error {
	if provider.reconnecting {
		provider.logger.Error("Impossible to set the olric value while reconnecting.")

		return errors.New("reconnecting error")
	}

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	err := dm.Put(context.Background(), key, value, olric.EX(duration))
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Errorf("Impossible to set value into Olric, %v", err)

		return err
	}

	return err
}

// Delete method will delete the response in Olric provider if exists corresponding to key param.
func (provider *Olric) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Error("Impossible to delete the olric key while reconnecting.")

		return
	}

	dm := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dm)

	_, err := dm.Delete(context.Background(), key)
	if err != nil {
		provider.logger.Errorf("Impossible to delete value into Olric, %v", err)
	}
}

// DeleteMany method will delete the responses in Olric provider if exists corresponding to the regex key param.
func (provider *Olric) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Error("Impossible to delete the olric keys while reconnecting.")

		return
	}

	dmap := provider.dm.Get().(olric.DMap)
	defer provider.dm.Put(dmap)

	records, err := dmap.Scan(context.Background(), olric.Match(key))
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Error("An error occurred while trying to list keys in Olric: %s\n", err)

		return
	}

	keys := []string{}
	for records.Next() {
		keys = append(keys, records.Key())
	}

	records.Close()

	_, _ = dmap.Delete(context.Background(), keys...)
}

// Init method will initialize Olric provider if needed.
func (provider *Olric) Init() error {
	provider.dm = &sync.Pool{
		New: func() interface{} {
			dmap, _ := provider.NewDMap("souin-map")

			return dmap
		},
	}

	return nil
}

// Reset method will reset or close provider.
func (provider *Olric) Reset() error {
	return provider.Close(context.Background())
}

func (provider *Olric) Reconnect() {
	provider.reconnecting = true

	if c, err := olric.NewClusterClient(provider.addresses, olric.WithConfig(&provider.configuration)); err == nil && c != nil {
		provider.Client = c
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}

// isOlricError reports whether err is the given olric sentinel error. The
// olric cluster client rebuilds the errors the server replies with from their
// message, e.g. a missing key comes back as a new "key not found" error, so
// errors.Is alone never matches them: compare the messages as well. Treating
// every cache miss as a connection failure made Get start a reconnection on
// each miss, failing the writes and blanking the reads meanwhile.
func isOlricError(err, target error) bool {
	return errors.Is(err, target) || (err != nil && err.Error() == target.Error())
}
