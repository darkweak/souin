package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/storages/core"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/connectivity"
)

// Etcd provider type.
type Etcd struct {
	*clientv3.Client

	stale         time.Duration
	ctx           context.Context
	logger        core.Logger
	reconnecting  bool
	configuration clientv3.Config
}

// Factory function create new Etcd instance.
func Factory(etcdCfg core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	etcdConfiguration := clientv3.Config{
		DialTimeout:      5 * time.Second,
		AutoSyncInterval: 1 * time.Second,
	}

	zapLogger, ok := logger.(*zap.SugaredLogger)
	if ok {
		etcdConfiguration.Logger = zapLogger.Desugar()
	}

	if etcdCfg.URL != "" {
		etcdConfiguration.Endpoints = strings.Split(etcdCfg.URL, ",")
	} else {
		bc, err := json.Marshal(etcdCfg.Configuration)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(bc, &etcdConfiguration)
		if err != nil {
			return nil, err
		}
	}

	cli, err := clientv3.New(etcdConfiguration)
	if err != nil {
		logger.Error("Impossible to initialize the Etcd DB.", err)

		return nil, err
	}

	// Block until the connection is ready, as before, but without spinning a
	// CPU core: sleep until the connection state changes.
	conn := cli.ActiveConnection()
	for state := conn.GetState(); state != connectivity.Ready; state = conn.GetState() {
		conn.Connect()
		conn.WaitForStateChange(context.Background(), state)
	}

	return &Etcd{
		Client:        cli,
		ctx:           context.Background(),
		stale:         stale,
		logger:        logger,
		configuration: etcdConfiguration,
	}, nil
}

// Name returns the storer name.
func (provider *Etcd) Name() string {
	return "ETCD"
}

// Uuid returns an unique identifier.
func (provider *Etcd) Uuid() string {
	return fmt.Sprintf(
		"%s-%s-%s-%s",
		strings.Join(provider.Endpoints(), ","),
		provider.Username,
		provider.Password,
		provider.stale,
	)
}

// ListKeys method returns the list of existing keys.
func (provider *Etcd) ListKeys() []string {
	if provider.reconnecting {
		provider.logger.Error("Impossible to list the etcd keys while reconnecting.")

		return []string{}
	}

	keys := []string{}

	result, e := provider.Client.Get(provider.ctx, core.MappingKeyPrefix, clientv3.WithPrefix())
	if e != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		return []string{}
	}

	for _, k := range result.Kvs {
		mapping, err := core.DecodeMapping(k.Value)
		if err == nil {
			for _, v := range mapping.GetMapping() {
				keys = append(keys, v.GetRealKey())
			}
		}
	}

	return keys
}

// MapKeys method returns the map of existing keys.
func (provider *Etcd) MapKeys(prefix string) map[string]string {
	if provider.reconnecting {
		provider.logger.Error("Impossible to list the etcd keys while reconnecting.")

		return map[string]string{}
	}

	keys := map[string]string{}

	// Let etcd filter on the prefix instead of transferring the whole keyspace.
	result, err := provider.Client.Get(provider.ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		return map[string]string{}
	}

	for _, k := range result.Kvs {
		key := string(k.Key)
		if strings.HasPrefix(key, prefix) {
			nk, _ := strings.CutPrefix(key, prefix)
			keys[nk] = string(k.Value)
		}
	}

	return keys
}

// Get method returns the populated response if exists, empty response then.
func (provider *Etcd) Get(key string) (item []byte) {
	if provider.reconnecting {
		provider.logger.Error("Impossible to get the etcd key while reconnecting.")

		return []byte{}
	}

	result, err := provider.Client.Get(provider.ctx, key)
	if err != nil && !provider.reconnecting {
		go provider.Reconnect()

		return
	}

	if err == nil && result != nil && len(result.Kvs) > 0 {
		item = result.Kvs[0].Value
	}

	return
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Etcd) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	if provider.reconnecting {
		provider.logger.Error("Impossible to get the etcd key while reconnecting.")

		return
	}

	result, err := provider.Client.Get(provider.ctx, core.MappingKeyPrefix+key)
	if err != nil {
		go provider.Reconnect()

		return fresh, stale
	}

	if len(result.Kvs) > 0 {
		fresh, stale, _ = core.MappingElection(provider, result.Kvs[0].Value, req, validator, provider.logger)
	}

	return fresh, stale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Etcd) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	if provider.reconnecting {
		provider.logger.Error("Impossible to set the etcd value while reconnecting.")

		return errors.New("reconnecting error")
	}

	now := time.Now()

	if provider.reconnecting {
		provider.logger.Error("Impossible to set the etcd value while reconnecting.")

		return errors.New("reconnecting error")
	}

	if provider.Client.ActiveConnection().GetState() != connectivity.Ready && provider.Client.ActiveConnection().GetState() != connectivity.Idle {
		return fmt.Errorf("the connection is not ready: %v", provider.Client.ActiveConnection().GetState())
	}

	compressed, err := core.CompressValue(value)
	if err != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Etcd, %v", variedKey, err)

		return err
	}

	mappingKey := core.MappingKeyPrefix + baseKey

	// The leases and the current mapping are independent, fetch them at once
	// instead of one round trip after the other.
	var (
		group                     sync.WaitGroup
		variedLease, mappingLease *clientv3.LeaseGrantResponse
		variedErr, mappingErr     error
		mapping                   []byte
	)

	group.Go(func() {
		variedLease, variedErr = provider.Grant(provider.ctx, int64(duration.Seconds()))
	})
	group.Go(func() {
		mappingLease, mappingErr = provider.Grant(provider.ctx, int64((duration + provider.stale).Seconds()))
	})
	group.Go(func() {
		mapping = provider.Get(mappingKey)
	})
	group.Wait()

	if err = errors.Join(variedErr, mappingErr); err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Errorf("Impossible to set value into Etcd, %v", err)

		return err
	}

	val, err := core.MappingUpdater(variedKey, mapping, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if err != nil {
		return err
	}

	// Store the response and its mapping in a single transaction.
	_, err = provider.Txn(provider.ctx).Then(
		clientv3.OpPut(variedKey, string(compressed), clientv3.WithLease(variedLease.ID)),
		clientv3.OpPut(mappingKey, string(val), clientv3.WithLease(mappingLease.ID)),
	).Commit()
	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Errorf("Impossible to set value into Etcd, %v", err)
	}

	return err
}

// Set method will store the response in Etcd provider.
func (provider *Etcd) Set(key string, value []byte, duration time.Duration) error {
	if provider.reconnecting {
		provider.logger.Error("Impossible to set the etcd value while reconnecting.")

		return errors.New("reconnecting error")
	}

	if provider.Client.ActiveConnection().GetState() != connectivity.Ready && provider.Client.ActiveConnection().GetState() != connectivity.Idle {
		return fmt.Errorf("the connection is not ready: %v", provider.Client.ActiveConnection().GetState())
	}

	rs, err := provider.Grant(context.TODO(), int64(duration.Seconds()))
	if err == nil {
		_, err = provider.Put(provider.ctx, key, string(value), clientv3.WithLease(rs.ID))
	}

	if err != nil {
		if !provider.reconnecting {
			go provider.Reconnect()
		}

		provider.logger.Errorf("Impossible to set value into Etcd, %v", err)
	}

	return err
}

// Delete method will delete the response in Etcd provider if exists corresponding to key param.
func (provider *Etcd) Delete(key string) {
	if provider.reconnecting {
		provider.logger.Error("Impossible to delete the etcd key while reconnecting.")

		return
	}

	_, _ = provider.Client.Delete(provider.ctx, key)
}

// DeleteMany method will delete the responses in Etcd provider if exists corresponding to the regex key param.
func (provider *Etcd) DeleteMany(key string) {
	if provider.reconnecting {
		provider.logger.Error("Impossible to delete the etcd keys while reconnecting.")

		return
	}

	rgKey, err := regexp.Compile(key)
	if err != nil {
		return
	}

	// Only the keys are needed to match, don't transfer the values.
	response, err := provider.Client.Get(provider.ctx, "\x00", clientv3.WithFromKey(), clientv3.WithKeysOnly())
	if err != nil {
		return
	}

	// Delete the matching keys in transactions of deleteBatchSize operations
	// instead of one request per key.
	ops := make([]clientv3.Op, 0, deleteBatchSize)

	flush := func() {
		if len(ops) == 0 {
			return
		}

		if _, err := provider.Txn(provider.ctx).Then(ops...).Commit(); err != nil {
			provider.logger.Errorf("Impossible to delete the keys matching %s from Etcd, %v", key, err)
		}

		ops = ops[:0]
	}

	for _, k := range response.Kvs {
		if rgKey.Match(k.Key) {
			ops = append(ops, clientv3.OpDelete(string(k.Key)))

			if len(ops) == deleteBatchSize {
				flush()
			}
		}
	}

	flush()
}

// deleteBatchSize stays below the etcd --max-txn-ops default of 128.
const deleteBatchSize = 100

// Init method will.
func (provider *Etcd) Init() error {
	return nil
}

// Reset method will reset or close provider.
func (provider *Etcd) Reset() error {
	return provider.Close()
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
