package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	reconnecting  bool
	configuration redis.ClientOption
	close         func()
}

// RedisConnectionFactory function create new Nuts instance
func RedisConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	dc := c.GetDefaultCache()
	bc, _ := json.Marshal(dc.GetRedis().Configuration)

	var options redis.ClientOption
	if dc.GetRedis().Configuration != nil {
		if err := json.Unmarshal(bc, &options); err != nil {
			c.GetLogger().Sugar().Infof("Cannot parse your redis configuration: %+v", err)
		}
	} else {
		options = redis.ClientOption{
			InitAddress: strings.Split(dc.GetRedis().URL, ","),
			Dialer: net.Dialer{
				Timeout: dc.GetTimeout().Cache.Duration,
			},
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
	}, err
}

// Name returns the storer name
func (provider *Redis) Name() string {
	return "REDIS"
}

// ListKeys method returns the list of existing keys
func (provider *Redis) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the redis keys while reconnecting.")
		return []string{}
	}

	keys, _ := provider.inClient.Do(provider.ctx, provider.inClient.B().Keys().Pattern("*").Build()).AsStrSlice()

	return keys
}

// MapKeys method returns the list of existing keys
func (provider *Redis) MapKeys(prefix string) map[string]string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the redis keys while reconnecting.")
		return map[string]string{}
	}

	m := map[string]string{}
	keys, _ := provider.inClient.Do(provider.ctx, provider.inClient.B().Keys().Pattern("*").Build()).AsStrSlice()
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			k, _ := strings.CutPrefix(key, prefix)
			m[k] = string(provider.Get(key))
		}
	}

	return m
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Redis) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the redis key while reconnecting.")
		return
	}

	b, e := provider.inClient.Do(provider.ctx, provider.inClient.B().Get().Key(MappingKeyPrefix+key).Build()).AsBytes()
	if e != nil {
		if !errors.Is(e, redis.Nil) && !provider.reconnecting {
			go provider.Reconnect()
		}
		return fresh, stale
	}

	fresh, stale, _ = mappingElection(provider, b, req, validator, provider.logger)

	return fresh, stale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Redis) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the redis value while reconnecting.")
		return fmt.Errorf("reconnecting error")
	}

	now := time.Now()
	if err := provider.inClient.Do(provider.ctx, provider.inClient.B().Set().Key(variedKey).Value(string(value)).Ex(duration+provider.stale).Build()).Error(); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
		return err
	}

	mappingKey := MappingKeyPrefix + baseKey
	v, e := provider.inClient.Do(provider.ctx, provider.inClient.B().Get().Key(mappingKey).Build()).AsBytes()
	if e != nil && !errors.Is(e, redis.Nil) {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		return e
	}

	val, e := mappingUpdater(variedKey, v, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag)
	if e != nil {
		return e
	}

	if e = provider.inClient.Do(provider.ctx, provider.inClient.B().Set().Key(mappingKey).Value(string(val)).Build()).Error(); e != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", e)
	}

	return e
}

// Get method returns the populated response if exists, empty response then
func (provider *Redis) Get(key string) []byte {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the redis key while reconnecting.")
		return nil
	}

	r, e := provider.inClient.Do(provider.ctx, provider.inClient.B().Get().Key(key).Build()).AsBytes()
	if e != nil && e != redis.Nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		return nil
	}

	return r
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Redis) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the redis keys by prefix while reconnecting.")
		return nil
	}
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
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the redis value while reconnecting.")
		return fmt.Errorf("reconnecting error")
	}

	err := provider.inClient.Do(provider.ctx, provider.inClient.B().Set().Key(key).Value(string(value)).Ex(duration+provider.stale).Build()).Error()
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
	}

	return err
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Redis) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the redis key while reconnecting.")
		return
	}
	_ = provider.inClient.Do(provider.ctx, provider.inClient.B().Del().Key(key).Build())
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param
func (provider *Redis) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the redis keys while reconnecting.")
		return
	}

	keys, _ := provider.inClient.Do(provider.ctx, provider.inClient.B().Keys().Pattern(key).Build()).AsStrSlice()
	_ = provider.inClient.Do(provider.ctx, provider.inClient.B().Del().Key(keys...).Build())
}

// Init method will
func (provider *Redis) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Redis) Reset() error {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to reset the redis instance while reconnecting.")
		return nil
	}
	provider.close()

	return nil
}

func (provider *Redis) Reconnect() {
	provider.reconnecting = true

	cli, err := redis.NewClient(provider.configuration)
	if err != nil {
		time.Sleep(10 * time.Second)
		provider.Reconnect()

		return
	}

	provider.inClient = cli
	provider.close = cli.Close
	if provider.inClient != nil {
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
