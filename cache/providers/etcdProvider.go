package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Etcd provider type
type Etcd struct {
	*clientv3.Client
	stale time.Duration
	ctx   context.Context
}

// EtcdConnectionFactory function create new Nuts instance
func EtcdConnectionFactory(c t.AbstractConfigurationInterface) (*Etcd, error) {
	dc := c.GetDefaultCache()
	bc, _ := json.Marshal(dc.GetEtcd().Configuration)
	etcdConfiguration := clientv3.Config{
		DialTimeout:      5 * time.Second,
		AutoSyncInterval: 1 * time.Second,
		Logger:           c.GetLogger(),
	}
	_ = json.Unmarshal(bc, &etcdConfiguration)

	cli, err := clientv3.New(etcdConfiguration)

	if err != nil {
		c.GetLogger().Sugar().Error("Impossible to initialize the Etcd DB.", err)
		return nil, err
	}

	return &Etcd{
		Client: cli,
		ctx:    context.Background(),
		stale:  dc.GetStale(),
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Etcd) ListKeys() []string {
	keys := []string{}

	r, e := provider.Client.Get(provider.ctx, "\x00", clientv3.WithFromKey())

	if e != nil {
		return []string{}
	}
	for _, k := range r.Kvs {
		keys = append(keys, string(k.Key))
	}

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Etcd) Get(key string) (item []byte) {
	r, e := provider.Client.Get(provider.ctx, key)

	if e == nil && r != nil && len(r.Kvs) > 0 {
		item = r.Kvs[0].Value
	}

	return
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Etcd) Prefix(key string, req *http.Request) []byte {
	r, e := provider.Client.Get(provider.ctx, key, clientv3.WithPrefix())

	if e == nil && r != nil {
		for _, v := range r.Kvs {
			if varyVoter(key, req, string(v.Key)) {
				return v.Value
			}
		}
	}

	return []byte{}
}

// Set method will store the response in Etcd provider
func (provider *Etcd) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	rs, _ := provider.Client.Grant(context.TODO(), int64(duration.Seconds()))
	_, err := provider.Client.Put(provider.ctx, key, string(value), clientv3.WithLease(rs.ID))

	if err != nil {
		panic(fmt.Sprintf("Impossible to set value into Etcd, %s", err))
	}

	_, err = provider.Client.Put(provider.ctx, stalePrefix+key, string(value), clientv3.WithLease(rs.ID))

	if err != nil {
		panic(fmt.Sprintf("Impossible to set value into Etcd, %s", err))
	}
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param
func (provider *Etcd) Delete(key string) {
	_, _ = provider.Client.Delete(provider.ctx, key)
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param
func (provider *Etcd) DeleteMany(key string) {
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
