package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/go-redis/redis/v9"
)

// Redis provider type
type Redis struct {
	*redis.Client
	stale     time.Duration
	ctx       context.Context
	stopAfter time.Duration
}

// RedisConnectionFactory function create new Nuts instance
func RedisConnectionFactory(c t.AbstractConfigurationInterface) (*Redis, error) {
	dc := c.GetDefaultCache()
	bc, _ := json.Marshal(dc.GetRedis().Configuration)

	var options redis.Options
	if dc.GetRedis().Configuration != nil {
		_ = json.Unmarshal(bc, &options)
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
		Client:    cli,
		ctx:       context.Background(),
		stale:     dc.GetStale(),
		stopAfter: dc.GetTimeout().Cache.Duration,
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Redis) ListKeys() []string {
	keys := []string{}

	iter := provider.Client.Scan(provider.ctx, 0, "*", 0).Iterator()
	for iter.Next(provider.ctx) {
		keys = append(keys, string(iter.Val()))
	}
	if err := iter.Err(); err != nil {
		fmt.Println(err)
		return []string{}
	}

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Redis) Get(key string) (item []byte) {
	r, e := provider.Client.Get(provider.ctx, key).Result()
	if e != nil {
		return
	}

	item = []byte(r)

	if e != nil {
		return
	}

	return
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Redis) Prefix(key string, req *http.Request) []byte {
	in := make(chan []byte)
	out := make(chan bool)

	iter := provider.Client.Scan(provider.ctx, 0, key+"*", 0).Iterator()
	go func(iterator *redis.ScanIterator) {
		for iterator.Next(provider.ctx) {
			select {
			case <-out:
				return
			case <-time.After(1 * time.Nanosecond):
				fmt.Println(iterator.Val())
				if varyVoter(key, req, iter.Val()) {
					v, _ := provider.Client.Get(provider.ctx, iter.Val()).Result()
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
func (provider *Redis) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	if err := provider.Client.Set(provider.ctx, key, value, duration).Err(); err != nil {
		panic(fmt.Sprintf("Impossible to set value into Redis, %s", err))
	}

	if err := provider.Client.Set(provider.ctx, stalePrefix+key, value, duration+provider.stale).Err(); err != nil {
		panic(fmt.Sprintf("Impossible to set value into Redis, %s", err))
	}
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Redis) Delete(key string) {
	_ = provider.Client.Del(provider.ctx, key)
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param
func (provider *Redis) DeleteMany(key string) {
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

	provider.Client.Del(provider.ctx, keys...)
}

// Init method will
func (provider *Redis) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Redis) Reset() error {
	return provider.Client.Close()
}
