package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v3"
)

// EmbeddedOlric provider type
type EmbeddedOlric struct {
	dm     olric.DMap
	db     *olric.Olric
	stale  time.Duration
	logger *zap.Logger
	ct     context.Context
}

func tryToLoadConfiguration(olricInstance *config.Config, olricConfiguration t.CacheProvider, logger *zap.Logger) (*config.Config, bool) {
	var e error
	isAlreadyLoaded := false
	if olricConfiguration.Configuration == nil && olricConfiguration.Path != "" {
		if olricInstance, e = config.Load(olricConfiguration.Path); e == nil {
			isAlreadyLoaded = true
		}
	} else if olricConfiguration.Configuration != nil {
		tmpFile := "/tmp/" + uuid.NewString() + ".yml"
		yamlConfig, e := yaml.Marshal(olricConfiguration.Configuration)
		defer func() {
			if e = os.RemoveAll(tmpFile); e != nil {
				logger.Error("Impossible to remove the temporary file")
			}
		}()
		if e = os.WriteFile(
			tmpFile,
			yamlConfig,
			0600,
		); e != nil {
			logger.Error("Impossible to create the embedded Olric config from the given one")
		}

		if olricInstance, e = config.Load(tmpFile); e == nil {
			isAlreadyLoaded = true
		} else {
			logger.Error("Impossible to create the embedded Olric config from the given one")
		}
	}

	return olricInstance, isAlreadyLoaded
}

// EmbeddedOlricConnectionFactory function create new EmbeddedOlric instance
func EmbeddedOlricConnectionFactory(configuration t.AbstractConfigurationInterface) (types.Storer, error) {
	var olricInstance *config.Config
	loaded := false

	if olricInstance, loaded = tryToLoadConfiguration(olricInstance, configuration.GetDefaultCache().GetOlric(), configuration.GetLogger()); !loaded {
		olricInstance = config.New("local")
		olricInstance.DMaps.MaxInuse = 512 << 20
	}

	started, cancel := context.WithCancel(context.Background())
	olricInstance.Started = func() {
		configuration.GetLogger().Sugar().Error("Embedded Olric is ready")
		defer cancel()
	}

	db, err := olric.New(olricInstance)
	if err != nil {
		return nil, err
	}

	ch := make(chan error, 1)
	defer func() {
		close(ch)
	}()

	go func(cdb *olric.Olric) {
		if err = cdb.Start(); err != nil {
			ch <- err
		}
	}(db)

	select {
	case err = <-ch:
	case <-started.Done():
	}
	dm, e := db.NewEmbeddedClient().NewDMap("souin-map")

	configuration.GetLogger().Sugar().Info("Embedded Olric is ready for this node.")

	return &EmbeddedOlric{
		dm:     dm,
		db:     db,
		stale:  configuration.GetDefaultCache().GetStale(),
		logger: configuration.GetLogger(),
		ct:     context.Background(),
	}, e
}

// Name returns the storer name
func (provider *EmbeddedOlric) Name() string {
	return "EMBEDDED_OLRIC"
}

// ListKeys method returns the list of existing keys
func (provider *EmbeddedOlric) ListKeys() []string {

	records, err := provider.dm.Scan(provider.ct)
	if err != nil {
		provider.logger.Sugar().Errorf("An error occurred while trying to list keys in Olric: %s\n", err)
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

// MapKeys method returns a map with the key and value
func (provider *EmbeddedOlric) MapKeys(prefix string) map[string]string {
	records, err := provider.dm.Scan(provider.ct)
	if err != nil {
		provider.logger.Sugar().Errorf("An error occurred while trying to map keys in Olric: %s\n", err)
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

// Prefix method returns the populated response if exists, empty response then
func (provider *EmbeddedOlric) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	records, err := provider.dm.Scan(provider.ct, olric.Match("^"+key+"({|$)"))
	if err != nil {
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

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *EmbeddedOlric) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	var resultFresh *http.Response
	var resultStale *http.Response

	res, e := provider.dm.Get(provider.ct, key)

	if e != nil {
		return resultFresh, resultStale
	}

	val, _ := res.Byte()
	resultFresh, resultStale, _ = mappingElection(provider, val, req, validator, provider.logger)

	return resultFresh, resultStale
}

// SetMultiLevel tries to store the keywith the given value and update the mapping key to store metadata.
func (provider *EmbeddedOlric) SetMultiLevel(baseKey, key string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error {
	now := time.Now()

	if err := provider.dm.Put(provider.ct, key, value, olric.EX(duration)); err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into EmbeddedOlric, %v", err)
		return err
	}

	mappingKey := mappingKeyPrefix + baseKey
	res, e := provider.dm.Get(provider.ct, mappingKey)
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

// Get method returns the populated response if exists, empty response then
func (provider *EmbeddedOlric) Get(key string) []byte {
	res, err := provider.dm.Get(provider.ct, key)

	if err != nil {
		return []byte{}
	}

	val, _ := res.Byte()
	return val
}

// Set method will store the response in EmbeddedOlric provider
func (provider *EmbeddedOlric) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	if err := provider.dm.Put(provider.ct, key, value, olric.EX(duration)); err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into EmbeddedOlric, %v", err)
		return err
	}

	if err := provider.dm.Put(provider.ct, StalePrefix+key, value, olric.EX(provider.stale+duration)); err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into EmbeddedOlric, %v", err)
	}

	return nil
}

// Delete method will delete the response in EmbeddedOlric provider if exists corresponding to key param
func (provider *EmbeddedOlric) Delete(key string) {
	_, err := provider.dm.Delete(provider.ct, key)
	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to delete value into Olric, %v", err)
	}
}

// DeleteMany method will delete the responses in EmbeddedOlric provider if exists corresponding to the regex key param
func (provider *EmbeddedOlric) DeleteMany(key string) {
	records, err := provider.dm.Scan(provider.ct, olric.Match(key))
	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to delete values into EmbeddedOlric, %v", err)
		return
	}

	keys := []string{}
	for records.Next() {
		keys = append(keys, records.Key())
	}
	records.Close()

	_, _ = provider.dm.Delete(provider.ct, keys...)
}

// Init method will initialize EmbeddedOlric provider if needed
func (provider *EmbeddedOlric) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *EmbeddedOlric) Reset() error {
	return provider.db.Shutdown(provider.ct)
}

// Destruct method will reset or close provider
func (provider *EmbeddedOlric) Destruct() error {
	provider.logger.Sugar().Debug("Destruct current embedded olric...")
	return provider.Reset()
}

// GetDM method returns the embbeded instance dm property
func (provider *EmbeddedOlric) GetDM() olric.DMap {
	return provider.dm
}
