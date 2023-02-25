package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"
)

// Redis provider type
type Redis struct {
	*redis.Client
	stale         time.Duration
	ctx           context.Context
	logger        *zap.Logger
	reconnecting  bool
	configuration redis.Options
}

// RedisConnectionFactory function create new Nuts instance
func RedisConnectionFactory(c t.AbstractConfigurationInterface) (Storer, error) {
	dc := c.GetDefaultCache()
	bc, _ := json.Marshal(dc.GetRedis().Configuration)

	var options redis.Options
	if dc.GetRedis().Configuration != nil {
		if err := json.Unmarshal(bc, &options); err != nil {
			c.GetLogger().Sugar().Infof("Cannot parse your redis configuration: %+v", err)
		}
	} else {
		options = redis.Options{
			Addr:        dc.GetRedis().URL,
			Password:    "",
			DB:          0,
			PoolSize:    1000,
			PoolTimeout: dc.GetTimeout().Cache.Duration,
		}
	}

	cli := redis.NewClient(&options)

	return &Redis{
		Client:        cli,
		ctx:           context.Background(),
		stale:         dc.GetStale(),
		configuration: options,
		logger:        c.GetLogger(),
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Redis) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the redis keys while reconnecting.")
		return []string{}
	}
	keys := []string{}

	iter := provider.Client.Scan(provider.ctx, 0, "*", 0).Iterator()
	for iter.Next(provider.ctx) {
		keys = append(keys, string(iter.Val()))
	}
	if err := iter.Err(); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Error(err)
		return []string{}
	}

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
func (provider *Redis) Prefix(key string, req *http.Request) []byte {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the redis keys by prefix while reconnecting.")
		return []byte{}
	}
	in := make(chan []byte)
	out := make(chan bool)

	iter := provider.Client.Scan(provider.ctx, 0, key+"*", 0).Iterator()
	go func(iterator *redis.ScanIterator) {
		for iterator.Next(provider.ctx) {
			select {
			case <-out:
				return
			case <-time.After(1 * time.Nanosecond):
				if varyVoter(key, req, iter.Val()) {
					v, e := provider.Client.Get(provider.ctx, iter.Val()).Result()
					if e != nil && e != redis.Nil && !provider.reconnecting {
						go provider.Reconnect()
						in <- []byte{}
						return
					}
					in <- []byte(v)
					return
				}
			}
		}
	}(iter)

	select {
	case <-time.After(provider.Client.Options().PoolTimeout):
		out <- true
		return []byte{}
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
	iter := provider.Client.Scan(provider.ctx, 0, "*", 0).Iterator()
	for iter.Next(provider.ctx) {
		if re.MatchString(iter.Val()) {
			keys = append(keys, iter.Val())
		}
	}

	if iter.Err() != nil && !provider.reconnecting {
		go provider.Reconnect()
		return
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
	return provider.Client.Close()
}

func (provider *Redis) Reconnect() {
	provider.reconnecting = true

	if provider.Client = redis.NewClient(&provider.configuration); provider.Client != nil {
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
