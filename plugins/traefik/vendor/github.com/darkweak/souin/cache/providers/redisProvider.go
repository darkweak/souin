package providers

import (
	"context"
	"github.com/mediocregopher/radix/v4"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/darkweak/souin/cache/types"
	t "github.com/darkweak/souin/configurationtypes"

	"go.uber.org/zap"
)

// Redis provider type
type Redis struct {
	radix.Client
	stale        time.Duration
	poolTimeout  time.Duration
	ctx          context.Context
	logger       *zap.Logger
	reconnecting bool
	addr         string
}

// RedisConnectionFactory function create new Nuts instance
func RedisConnectionFactory(c t.AbstractConfigurationInterface) (types.AbstractReconnectProvider, error) {
	dc := c.GetDefaultCache()

	client, err := (radix.PoolConfig{}).New(context.Background(), "tcp", dc.GetRedis().URL)
	if err != nil {
		return nil, err
	}
	return &Redis{
		Client: client,
		ctx:    context.Background(),
		stale:  dc.GetStale(),
		logger: c.GetLogger(),
		addr:   dc.GetRedis().URL,
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Redis) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the redis keys while reconnecting.")
		return []string{}
	}
	keys := []string{}

	s := (radix.ScannerConfig{Key: "*"}).New(provider.Client)
	var key string
	for s.Next(provider.ctx, &key) {
		keys = append(keys, key)
	}
	if err := s.Close(); err != nil {
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
	var val []byte
	e := provider.Client.Do(provider.ctx, radix.Cmd(&val, "GET", key))
	if e != nil && !provider.reconnecting {
		go provider.Reconnect()
	}

	item = val

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
	s := (radix.ScannerConfig{Key: key + "*"}).New(provider.Client)
	go func(iterator radix.Scanner) {
		var sKey string
		for iterator.Next(provider.ctx, &sKey) {
			select {
			case <-out:
				return
			case <-time.After(1 * time.Nanosecond):
				if varyVoter(key, req, sKey) {
					var val []byte
					e := provider.Client.Do(provider.ctx, radix.Cmd(&val, "GET", sKey))
					if e != nil && !provider.reconnecting {
						go provider.Reconnect()
						in <- []byte{}
						return
					}
					in <- val
					return
				}
			}
		}
	}(s)

	select {
	case <-time.After(provider.poolTimeout):
		out <- true
		return []byte{}
	case v := <-in:
		return v
	}
}

// Set method will store the response in Etcd provider
func (provider *Redis) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the redis value while reconnecting.")
		return
	}
	if duration == 0 {
		duration = url.TTL.Duration
	}
	if err := provider.Client.Do(provider.ctx, radix.Cmd(nil, "SET", key, string(value), "ex", strconv.FormatInt(int64(duration/time.Second), 10))); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
			return
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
	}
	if err := provider.Client.Do(provider.ctx, radix.Cmd(nil, "SET", StalePrefix+key, string(value), "ex", strconv.FormatInt(int64((duration+provider.stale)/time.Second), 10))); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
			return
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Redis, %v", err)
	}
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Redis) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the redis key while reconnecting.")
		return
	}
	_ = provider.Client.Do(provider.ctx, radix.Cmd(nil, "DEL", key))
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

	s := (radix.ScannerConfig{Key: "*"}).New(provider.Client)
	var skey string
	for s.Next(provider.ctx, &skey) {
		if re.MatchString(skey) {
			keys = append(keys, skey)
		}
	}
	if err := s.Close(); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
			return
		}
	}
	_ = provider.Client.Do(provider.ctx, radix.Cmd(nil, "DEL", keys...))
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
	client, err := (radix.PoolConfig{}).New(provider.ctx, "tcp", provider.addr)
	if err != nil {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
	if provider.Client = client; provider.Client != nil {
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
