package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	redis "github.com/redis/rueidis"
	"go.uber.org/zap"
)

// Redis provider type
type Redis struct {
	inClient      redis.Client
	stale         time.Duration
	ctx           context.Context
	logger        *zap.Logger
	configuration redis.ClientOption
	close         func()
	hashtags      string
}

// RedisConnectionFactory function create new Redis instance
func RedisConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	dc := c.GetDefaultCache()
	bc, _ := json.Marshal(dc.GetRedis().Configuration)

	var options redis.ClientOption
	var hashtags string
	if dc.GetRedis().Configuration != nil {
		if err := json.Unmarshal(bc, &options); err != nil {
			c.GetLogger().Sugar().Infof("Cannot parse your redis configuration: %+v", err)
		}
		if redisConfig, ok := dc.GetRedis().Configuration.(map[string]interface{}); ok && redisConfig != nil {
			if value, ok := redisConfig["HashTag"]; ok {
				if v, ok := value.(string); ok {
					hashtags = v
				}
			}
		}
	} else {
		options = redis.ClientOption{
			InitAddress: strings.Split(dc.GetRedis().URL, ","),
			Dialer: net.Dialer{
				Timeout: dc.GetTimeout().Cache.Duration,
			},
			SelectDB:   0,
			ClientName: "souin-redis",
		}
	}

	if options.Dialer.Timeout == 0 {
		options.Dialer.Timeout = time.Second
	}

	cli, err := redis.NewClient(options)
	if err != nil {
		return nil, err
	}

	return &Redis{
		inClient:      cli,
		ctx:           context.Background(),
		stale:         dc.GetStale(),
		configuration: options,
		logger:        c.GetLogger(),
		close:         cli.Close,
		hashtags:      hashtags,
	}, err
}

// Name returns the storer name
func (provider *Redis) Name() string {
	return "REDIS"
}

// ListKeys method returns the list of existing keys
func (provider *Redis) ListKeys() []string {
	provider.logger.Sugar().Debugf("Call the ListKeys function in redis")
	var scan redis.ScanEntry
	var err error
	elements := []string{}
	for more := true; more; more = scan.Cursor != 0 {
		if scan, err = provider.inClient.Do(context.Background(), provider.inClient.B().Scan().Cursor(scan.Cursor).Match(provider.hashtags+MappingKeyPrefix+"*").Build()).AsScanEntry(); err != nil {
			provider.logger.Sugar().Errorf("Cannot scan: %v", err)
		}
		for _, element := range scan.Elements {
			value := provider.Get(element)
			mapping, err := decodeMapping(value)
			if err == nil {
				for _, v := range mapping.Mapping {
					if !v.FreshTime.Before(time.Now()) || !v.StaleTime.Before(time.Now()) {
						elements = append(elements, v.RealKey)
					}
				}
			}
		}
	}

	return elements
}

// MapKeys method returns the list of existing keys
func (provider *Redis) MapKeys(prefix string) map[string]string {
	m := map[string]string{}
	provider.logger.Sugar().Debugf("Call the MapKeys in redis with the prefix %s", prefix)
	var scan redis.ScanEntry
	var err error
	elements := []string{}
	for more := true; more; more = scan.Cursor != 0 {
		if scan, err = provider.inClient.Do(context.Background(), provider.inClient.B().Scan().Cursor(scan.Cursor).Match(prefix+"*").Build()).AsScanEntry(); err != nil {
			provider.logger.Sugar().Errorf("Cannot scan: %v", err)
		}
		elements = append(elements, scan.Elements...)
	}
	for _, key := range elements {
		k, _ := strings.CutPrefix(key, prefix)
		m[k] = string(provider.Get(key))
	}

	return m
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Redis) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	b, e := provider.inClient.Do(provider.ctx, provider.inClient.B().Get().Key(provider.hashtags+MappingKeyPrefix+key).Build()).AsBytes()
	if e != nil {
		return fresh, stale
	}

	fresh, stale, _ = mappingElection(provider, b, req, validator, provider.logger)

	return fresh, stale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Redis) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()
	if err := provider.inClient.Do(provider.ctx, provider.inClient.B().Set().Key(provider.hashtags+variedKey).Value(string(value)).Ex(duration+provider.stale).Build()).Error(); err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
		return err
	}

	mappingKey := provider.hashtags + MappingKeyPrefix + baseKey
	v, e := provider.inClient.Do(provider.ctx, provider.inClient.B().Get().Key(mappingKey).Build()).AsBytes()
	if e != nil && !errors.Is(e, redis.Nil) {
		return e
	}

	val, e := mappingUpdater(provider.hashtags+variedKey, v, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	if e = provider.inClient.Do(provider.ctx, provider.inClient.B().Set().Key(mappingKey).Value(string(val)).Build()).Error(); e != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", e)
	}

	return e
}

// Get method returns the populated response if exists, empty response then
func (provider *Redis) Get(key string) []byte {
	r, e := provider.inClient.Do(provider.ctx, provider.inClient.B().Get().Key(key).Build()).AsBytes()
	if e != nil && e != redis.Nil {
		return nil
	}

	return r
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Redis) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	in := make(chan *http.Response)
	out := make(chan bool)

	keys, _ := provider.inClient.Do(provider.ctx, provider.inClient.B().Keys().Pattern(key+"*").Build()).AsStrSlice()
	go func(ks []string) {
		for _, k := range ks {
			select {
			case <-out:
				return
			case <-time.After(1 * time.Nanosecond):
				if varyVoter(key, req, k) {
					if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(provider.Get(k))), req); err == nil {
						rfc.ValidateETag(res, validator)
						if validator.Matched {
							provider.logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v", k, validator)
							in <- res
							return
						}
						provider.logger.Sugar().Errorf("The stored key %s didn't match the current iteration key ETag %+v", k, validator)
					}
				}
			}
		}
		in <- nil
	}(keys)

	select {
	case <-time.After(provider.configuration.Dialer.Timeout):
		out <- true
		return nil
	case v := <-in:
		return v
	}
}

// Set method will store the response in Etcd provider
func (provider *Redis) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	err := provider.inClient.Do(provider.ctx, provider.inClient.B().Set().Key(key).Value(string(value)).Ex(duration+provider.stale).Build()).Error()
	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
	}

	return err
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Redis) Delete(key string) {
	_ = provider.inClient.Do(provider.ctx, provider.inClient.B().Del().Key(key).Build())
}

// DeleteMany method will delete the responses in Redis provider if exists corresponding to the regex key param
func (provider *Redis) DeleteMany(key string) {
	provider.logger.Sugar().Debugf("Call the DeleteMany function in redis")
	var scan redis.ScanEntry
	var err error
	elements := []string{}
	for more := true; more; more = scan.Cursor != 0 {
		if scan, err = provider.inClient.Do(context.Background(), provider.inClient.B().Scan().Cursor(scan.Cursor).Match(key).Build()).AsScanEntry(); err != nil {
			provider.logger.Sugar().Errorf("Cannot scan: %v", err)
		}
		elements = append(elements, scan.Elements...)
	}
	_ = provider.inClient.Do(provider.ctx, provider.inClient.B().Del().Key(elements...).Build())
}

// Init method will
func (provider *Redis) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Redis) Reset() error {
	_ = provider.inClient.Do(provider.ctx, provider.inClient.B().Flushdb().Build())

	return nil
}

func (provider *Redis) Reconnect() {
	provider.logger.Debug("Doing nothing on reconnect because rueidis handles it!")
}
