package nats

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/darkweak/storages/core"
	nats "github.com/nats-io/nats.go"
)

// Nats provider type.
type Nats struct {
	// keyvalue is the bucket handle. Looking it up costs a JetStream API round
	// trip, so it is resolved once when the storer is created.
	keyvalue nats.KeyValue
	bucket   string
	stale    time.Duration
	logger   core.Logger
}

func sanitizeProperties(configMap map[string]interface{}) map[string]interface{} {
	for _, property := range []string{"MaxReconnect", "MaxPingsOut", "ReconnectBufSize", "SubChanLen"} {
		if v := configMap[property]; v != nil {
			configMap[property], _ = v.(int)
		}
	}

	for _, property := range []string{
		"Url",
		"Name",
		"Nkey",
		"User",
		"Password",
		"Token",
		"ProxyPath",
		"InboxPrefix",
	} {
		if v := configMap[property]; v != nil {
			configMap[property], _ = v.(string)
		}
	}

	for _, property := range []string{
		"ReconnectWait",
		"ReconnectJitter",
		"ReconnectJitterTLS",
		"Timeout",
		"DrainTimeout",
		"FlusherTimeout",
		"PingInterval",
	} {
		if v := configMap[property]; v != nil {
			if s, ok := v.(string); ok {
				configMap[property], _ = time.ParseDuration(s)
			} else if d, ok := v.(time.Duration); ok {
				configMap[property] = d
			}
		}
	}

	for _, property := range []string{
		"NoRandomize",
		"NoEcho",
		"Verbose",
		"Pedantic",
		"Secure",
		"TLSHandshakeFirst",
		"AllowReconnect",
		"UseOldRequestStyle",
		"NoCallbacksAfterClientClose",
		"RetryOnFailedConnect",
		"Compression",
		"IgnoreAuthErrorAbort",
		"SkipHostLookup",
	} {
		if v := configMap[property]; v != nil {
			if b, ok := v.(bool); ok {
				configMap[property] = b
			} else if s, ok := v.(string); ok {
				configMap[property], _ = strconv.ParseBool(s)
			}
		}
	}

	if v := configMap["Servers"]; v != nil {
		if s, ok := v.([]string); ok {
			configMap["Servers"] = s
		} else if s, ok := v.(string); ok {
			configMap["Servers"] = strings.Split(s, ",")
		}
	}

	return configMap
}

// Factory function create new Nats instance.
func Factory(natsConfiguration core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	natsOptions := nats.GetDefaultOptions()
	bucketName := "souin-bucket"

	if natsConfiguration.Configuration != nil {
		var parsedNats nats.Options

		if bucket, ok := natsConfiguration.Configuration.(map[string]interface{})["keyvalue"]; ok {
			bucketName, _ = bucket.(string)
		}

		natsConfiguration.Configuration = sanitizeProperties(natsConfiguration.Configuration.(map[string]interface{}))
		if b, e := json.Marshal(natsConfiguration.Configuration); e == nil {
			if e = json.Unmarshal(b, &parsedNats); e != nil {
				logger.Error("Impossible to parse the configuration for the Nuts provider", e)
			}
		}

		if err := mergo.Merge(&natsOptions, parsedNats, mergo.WithOverride); err != nil {
			logger.Error("An error occurred during the nutsOptions merge from the default options with your configuration.")
		}
	} else {
		natsOptions.Servers = strings.Split(natsConfiguration.URL, ",")
	}

	if len(natsOptions.Servers) == 0 {
		natsOptions.Servers = []string{nats.DefaultURL}
	}

	natsConn, err := natsOptions.Connect()
	if err != nil {
		logger.Error("Impossible to connect to the Nats DB.", err)

		return nil, err
	}

	stream, err := natsConn.JetStream()
	if err != nil {
		logger.Error("Impossible to instantiate the Nats DB.", err)

		return nil, err
	}

	//nolint:exhaustruct_v5
	keyvalue, err := stream.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: bucketName,
	})
	if err != nil {
		logger.Error("Impossible to create the Nats bucket %s.", err, bucketName)

		return nil, err
	}

	return &Nats{keyvalue: keyvalue, bucket: bucketName, logger: logger, stale: stale}, nil
}

// Name returns the storer name.
func (provider *Nats) Name() string {
	return "NATS"
}

// Uuid returns an unique identifier.
func (provider *Nats) Uuid() string {
	return fmt.Sprintf("%s-%s", provider.bucket, provider.stale)
}

// MapKeys method returns a map with the key and value.
func (provider *Nats) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	keyvalue := provider.keyvalue

	keysList, err := keyvalue.Keys()
	if err != nil {
		return keys
	}

	for _, key := range keysList {
		if strings.HasPrefix(key, prefix) {
			val, _ := keyvalue.Get(key)
			keys[strings.TrimPrefix(key, prefix)] = string(val.Value())
		}
	}

	return keys
}

// ListKeys method returns the list of existing keys.
func (provider *Nats) ListKeys() []string {
	keyvalue := provider.keyvalue

	keys, _ := keyvalue.Keys()

	return keys
}

// Get method returns the populated response if exists, empty response then.
func (provider *Nats) Get(key string) []byte {
	keyvalue := provider.keyvalue

	value, err := keyvalue.Get(key)
	if err != nil && !errors.Is(err, nats.ErrKeyNotFound) {
		provider.logger.Errorf("Impossible to get the key %s in Nats: %v", key, err)

		return nil
	} else if err != nil {
		return nil
	}

	return value.Value()
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Nats) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	keyvalue := provider.keyvalue

	value, err := keyvalue.Get(core.MappingKeyPrefix + key)
	if err != nil {
		provider.logger.Debugf("Impossible to get the mapping key %s in Nats", core.MappingKeyPrefix+key)

		return
	}

	fresh, stale, _ = core.MappingElection(provider, value.Value(), req, validator, provider.logger)

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Nats) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	compressed, err := core.CompressValue(value)
	if err != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Nats: %v", variedKey, err)

		return err
	}

	// A NATS key/value bucket has no per key lifetime: the response is stored
	// as is, and the mapping fresh and stale deadlines keep it from being
	// served once expired.
	if _, err = provider.keyvalue.Put(variedKey, compressed); err != nil {
		provider.logger.Errorf("Impossible to set value into Nats for the key %s, %v", variedKey, err)

		return err
	}

	mappingKey := core.MappingKeyPrefix + baseKey
	r := provider.Get(mappingKey)

	val, err := core.MappingUpdater(variedKey, r, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if err != nil {
		provider.logger.Errorf("Impossible to update the mapping key %s in Nats: %v", mappingKey, err)

		return err
	}

	return provider.Set(mappingKey, val, duration+provider.stale)
}

// Set method will store the response in Nats provider.
func (provider *Nats) Set(key string, value []byte, _ time.Duration) error {
	keyvalue := provider.keyvalue

	_, err := keyvalue.Put(key, value)
	if err != nil {
		provider.logger.Errorf("Impossible to set value into Nats, %v", err)
	}

	return err
}

// Delete method will delete the response in Nats provider if exists corresponding to key param.
func (provider *Nats) Delete(key string) {
	keyvalue := provider.keyvalue

	_ = keyvalue.Purge(key)
}

// DeleteMany method will delete the responses in Nats provider if exists corresponding to the regex key param.
func (provider *Nats) DeleteMany(key string) {
	rgKey, err := regexp.Compile(key)
	if err != nil {
		return
	}

	keyvalue := provider.keyvalue

	keys, err := keyvalue.Keys()
	if err != nil {
		return
	}

	for _, key := range keys {
		if rgKey.MatchString(key) {
			_ = keyvalue.Purge(key)
		}
	}
}

// Init method will.
func (provider *Nats) Init() error {
	return nil
}

// Reset method will reset or close provider.
func (provider *Nats) Reset() error {
	return nil
}
