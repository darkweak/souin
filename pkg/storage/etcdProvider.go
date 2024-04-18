package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	lz4 "github.com/pierrec/lz4/v4"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/connectivity"
)

// Etcd provider type
type Etcd struct {
	*clientv3.Client
	stale         time.Duration
	ctx           context.Context
	logger        *zap.Logger
	reconnecting  bool
	configuration clientv3.Config
}

// EtcdConnectionFactory function create new Etcd instance
func EtcdConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	dc := c.GetDefaultCache()
	bc, _ := json.Marshal(dc.GetEtcd().Configuration)
	etcdConfiguration := clientv3.Config{
		DialTimeout:      5 * time.Second,
		AutoSyncInterval: 1 * time.Second,
		Logger:           c.GetLogger(),
	}
	_ = json.Unmarshal(bc, &etcdConfiguration)

	etcdConfiguration.Endpoints = []string{"etcd:2379"}
	cli, err := clientv3.New(etcdConfiguration)

	if err != nil {
		c.GetLogger().Sugar().Error("Impossible to initialize the Etcd DB.", err)
		return nil, err
	}

	for {
		if cli.ActiveConnection().GetState() == connectivity.Ready {
			break
		}
	}

	return &Etcd{
		Client:        cli,
		ctx:           context.Background(),
		stale:         dc.GetStale(),
		logger:        c.GetLogger(),
		configuration: etcdConfiguration,
	}, nil
}

// Name returns the storer name
func (provider *Etcd) Name() string {
	return "ETCD"
}

// ListKeys method returns the list of existing keys
func (provider *Etcd) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the etcd keys while reconnecting.")
		return []string{}
	}
	keys := []string{}

	r, e := provider.Client.Get(provider.ctx, MappingKeyPrefix, clientv3.WithPrefix())

	if e != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		return []string{}
	}
	for _, k := range r.Kvs {
		mapping, err := decodeMapping(k.Value)
		if err == nil {
			for _, v := range mapping.Mapping {
				keys = append(keys, v.RealKey)
			}
		}
	}

	return keys
}

// MapKeys method returns the map of existing keys
func (provider *Etcd) MapKeys(prefix string) map[string]string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to list the etcd keys while reconnecting.")
		return map[string]string{}
	}

	keys := map[string]string{}
	r, e := provider.Client.Get(provider.ctx, "\x00", clientv3.WithFromKey())

	if e != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		return map[string]string{}
	}
	for _, k := range r.Kvs {
		key := string(k.Key)
		if strings.HasPrefix(key, prefix) {
			nk, _ := strings.CutPrefix(key, prefix)
			keys[nk] = string(k.Value)
		}
	}

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Etcd) Get(key string) (item []byte) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the etcd key while reconnecting.")
		return []byte{}
	}
	r, e := provider.Client.Get(provider.ctx, key)

	if e != nil && !provider.reconnecting {
		go provider.Reconnect()
		return
	}

	if e == nil && r != nil && len(r.Kvs) > 0 {
		item = r.Kvs[0].Value
	}

	return
}

// Prefix method returns the keys that match the prefix key
func (provider *Etcd) Prefix(key string) []string {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the etcd keys by prefix while reconnecting.")
		return nil
	}
	r, e := provider.Client.Get(provider.ctx, key, clientv3.WithPrefix())

	if e != nil && !provider.reconnecting {
		go provider.Reconnect()
		return nil
	}

	result := []string{}
	if e == nil && r != nil {
		for _, v := range r.Kvs {
			result = append(result, string(v.Key))
		}
	}

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Etcd) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to get the etcd key while reconnecting.")
		return
	}

	r, e := provider.Client.Get(provider.ctx, MappingKeyPrefix+key)
	if e != nil {
		go provider.Reconnect()
		return fresh, stale
	}

	if len(r.Kvs) > 0 {
		fresh, stale, _ = mappingElection(provider, r.Kvs[0].Value, req, validator, provider.logger)
	}

	return fresh, stale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Etcd) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the etcd value while reconnecting.")
		return fmt.Errorf("reconnecting error")
	}

	now := time.Now()
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the etcd value while reconnecting.")
		return fmt.Errorf("reconnecting error")
	}
	if provider.Client.ActiveConnection().GetState() != connectivity.Ready && provider.Client.ActiveConnection().GetState() != connectivity.Idle {
		return fmt.Errorf("the connection is not ready: %v", provider.Client.ActiveConnection().GetState())
	}

	compressed := new(bytes.Buffer)
	if _, err := lz4.NewWriter(compressed).ReadFrom(bytes.NewReader(value)); err != nil {
		provider.logger.Sugar().Errorf("Impossible to compress the key %s into Etcd, %v", variedKey, err)
		return err
	}
	rs, err := provider.Client.Grant(context.TODO(), int64(duration.Seconds()))
	if err == nil {
		_, err = provider.Client.Put(provider.ctx, variedKey, compressed.String(), clientv3.WithLease(rs.ID))
		fmt.Println("Put err =>", err)
	}

	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Etcd, %v", err)
		return err
	}

	mappingKey := MappingKeyPrefix + baseKey
	r := provider.Get(mappingKey)
	val, e := mappingUpdater(variedKey, []byte(r), provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	return provider.Set(mappingKey, val, t.URL{}, duration+provider.stale)
}

// Set method will store the response in Etcd provider
func (provider *Etcd) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to set the etcd value while reconnecting.")
		return fmt.Errorf("reconnecting error")
	}
	if provider.Client.ActiveConnection().GetState() != connectivity.Ready && provider.Client.ActiveConnection().GetState() != connectivity.Idle {
		return fmt.Errorf("the connection is not ready: %v", provider.Client.ActiveConnection().GetState())
	}
	if duration == 0 {
		duration = url.TTL.Duration
	}

	rs, err := provider.Client.Grant(context.TODO(), int64(duration.Seconds()))
	if err == nil {
		_, err = provider.Client.Put(provider.ctx, key, string(value), clientv3.WithLease(rs.ID))
	}

	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}
		provider.logger.Sugar().Errorf("Impossible to set value into Etcd, %v", err)
	}

	return err
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Etcd) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the etcd key while reconnecting.")
		return
	}
	_, _ = provider.Client.Delete(provider.ctx, key)
}

// DeleteMany method will delete the responses in Etcd provider if exists corresponding to the regex key param
func (provider *Etcd) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Sugar().Error("Impossible to delete the etcd keys while reconnecting.")
		return
	}
	re, e := regexp.Compile(key)

	if e != nil {
		return
	}

	if r, e := provider.Client.Get(provider.ctx, "\x00", clientv3.WithFromKey()); e == nil {
		for _, k := range r.Kvs {
			key := string(k.Key)
			if re.MatchString(key) {
				provider.Delete(key)
			}
		}
	}
}

// Init method will
func (provider *Etcd) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Etcd) Reset() error {
	return provider.Client.Close()
}

func (provider *Etcd) Reconnect() {
	provider.reconnecting = true

	if c, err := clientv3.New(provider.configuration); err == nil && c != nil {
		provider.Client = c
		provider.reconnecting = false
	} else {
		time.Sleep(10 * time.Second)
		provider.Reconnect()
	}
}
