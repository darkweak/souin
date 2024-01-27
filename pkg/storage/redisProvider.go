package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	redis "github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidiscompat"
	"go.uber.org/zap"
)

// Redis provider type
type Redis struct {
	Client        rueidiscompat.Cmdable
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

	// TODO remove this adapter to use directly rueidis
	compat := rueidiscompat.NewAdapter(cli)

	return &Redis{
		Client:        compat,
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

	keys, _ := provider.Client.Scan(provider.ctx, 0, "*", 0).Val()

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Redis) Get(key string) (item []byte) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the redis key while reconnecting.")
		return
	}
	r, e := provider.Client.Get(provider.ctx, key).Result()
	if e != nil {
		if e != redis.Nil && !provider.reconnecting {
			go provider.Reconnect()
		}
		return
	}

	item = []byte(r)

	return
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Redis) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the redis keys by prefix while reconnecting.")
		return nil
	}
	in := make(chan *http.Response)
	out := make(chan bool)

	keys, _, _ := provider.Client.Scan(provider.ctx, 0, key+"*", 0).Result()
	go func(ks []string) {
		for _, k := range ks {
			select {
			case <-out:
				return
			case <-time.After(1 * time.Nanosecond):
				if varyVoter(key, req, k) {
					v, e := provider.Client.Get(provider.ctx, k).Result()
					if e != nil && e != redis.Nil && !provider.reconnecting {
						go provider.Reconnect()
						in <- nil
						return
					}
					if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer([]byte(v))), req); err == nil {
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
	if duration == 0 {
		duration = url.TTL.Duration
	}

	if err := provider.Client.Set(provider.ctx, key, value, duration).Err(); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
		return err
	}

	if err := provider.Client.Set(provider.ctx, StalePrefix+key, value, duration+provider.stale).Err(); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
	}

	return nil
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Redis) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the redis key while reconnecting.")
		return
	}
	_ = provider.Client.Del(provider.ctx, key)
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param
func (provider *Redis) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the redis keys while reconnecting.")
		return
	}
	re, e := regexp.Compile(key)

	if e != nil {
		return
	}

	keys := []string{}
	rKeys, _ := provider.Client.Scan(provider.ctx, 0, "*", 0).Val()
	for _, rkey := range rKeys {
		if re.MatchString(rkey) {
			keys = append(keys, rkey)
		}
	}

	provider.Client.Del(provider.ctx, keys...)
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

	// TODO remove this adapter to use directly rueidis
	provider.Client = rueidiscompat.NewAdapter(cli)
	provider.close = cli.Close
	if provider.Client != nil {
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
