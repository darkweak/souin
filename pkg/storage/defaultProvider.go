package storage

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/storages/core"
	"github.com/pierrec/lz4/v4"
)

// Default provider type
type Default struct {
	m      *sync.Map
	stale  time.Duration
	logger core.Logger

	mu sync.Mutex
}

type item struct {
	invalidAt time.Time
	value     []byte
}

// Default keeps the surrogate keys as native sets.
var _ core.SetStorer = (*Default)(nil)

// Factory function create new Default instance
func Factory(c configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
	return &Default{m: &sync.Map{}, logger: c.GetLogger(), stale: c.GetDefaultCache().GetStale()}, nil
}

// Name returns the storer name
func (provider *Default) Name() string {
	return types.DefaultStorageName
}

// Uuid returns an unique identifier
func (provider *Default) Uuid() string {
	return ""
}

// MapKeys method returns a map with the key and value
func (provider *Default) MapKeys(prefix string) map[string]string {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	now := time.Now()
	keys := map[string]string{}

	provider.m.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), prefix) {
			k, _ := strings.CutPrefix(key.(string), prefix)
			if v, ok := value.(item); ok {
				if v.invalidAt.After(now) {
					keys[k] = string(v.value)
				}

				return true
			}

			if v, ok := value.(*core.StorageMapper); ok {
				for _, v := range v.Mapping {
					if v.StaleTime.AsTime().After(now) {
						keys[v.RealKey] = string(provider.Get(v.RealKey))
					}
				}
			}
		}

		return true
	})

	return keys
}

// ListKeys method returns the list of existing keys
func (provider *Default) ListKeys() []string {
	now := time.Now()
	keys := []string{}

	provider.m.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), core.MappingKeyPrefix) {
			mapping, err := core.DecodeMapping(value.([]byte))
			if err == nil {
				for _, v := range mapping.Mapping {
					if v.StaleTime.AsTime().After(now) {
						keys = append(keys, v.RealKey)
					} else {
						provider.m.Delete(v.RealKey)
					}
				}
			}
		}

		return true
	})

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Default) Get(key string) []byte {
	result, ok := provider.m.Load(key)
	if !ok || result == nil {
		return nil
	}

	res, ok := result.(item)
	if !ok {
		return nil
	}

	if res.invalidAt.After(time.Now()) {
		return res.value
	}

	provider.m.Delete(key)

	return nil
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Default) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	result, found := provider.m.Load(core.MappingKeyPrefix + key)
	if !found {
		return
	}

	fresh, stale, _ = core.MappingElection(provider, result.([]byte), req, validator, provider.logger)

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Default) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	var e error

	compressed := new(bytes.Buffer)
	writer := core.Lz4WriterPool.Get().(*lz4.Writer)
	writer.Reset(compressed)
	defer core.Lz4WriterPool.Put(writer)

	_, e = writer.Write(value)
	_ = writer.Close()
	if e != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Badger, %v", variedKey, e)
		return e
	}

	provider.m.Store(variedKey, item{
		invalidAt: now.Add(duration + provider.stale),
		value:     compressed.Bytes(),
	})

	mappingKey := core.MappingKeyPrefix + baseKey
	item, ok := provider.m.Load(mappingKey)
	var val []byte
	if ok {
		val = item.([]byte)
	}

	val, e = core.MappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	provider.logger.Debugf("Store the new mapping for the key %s in Default", variedKey)
	provider.m.Store(mappingKey, val)
	return nil
}

// Set method will store the response in Badger provider
func (provider *Default) Set(key string, value []byte, duration time.Duration) error {
	provider.m.Store(key, item{
		invalidAt: time.Now().Add(duration),
		value:     value,
	})

	return nil
}

// Delete method will delete the response in Badger provider if exists corresponding to key param
func (provider *Default) Delete(key string) {
	provider.m.Delete(key)
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param
func (provider *Default) DeleteMany(key string) {
	re, e := regexp.Compile(key)

	if e != nil {
		provider.logger.Infof("Failed to compile key %s, %v", key, e)
	}

	provider.m.Range(func(current, _ any) bool {
		if (re != nil && re.MatchString(current.(string))) || strings.HasPrefix(current.(string), key) {
			provider.m.Delete(current)
		}

		return true
	})
}

// Init method will
func (provider *Default) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Default) Reset() error {
	provider.mu.Lock()
	provider.m = new(sync.Map)
	provider.mu.Unlock()

	return nil
}

// set is the value stored under a key managed with AddToSet. Its members are
// kept in insertion order.
type set struct {
	mu        sync.Mutex
	seen      map[string]struct{}
	members   []string
	invalidAt time.Time
}

func newSet(members []string, invalidAt time.Time) *set {
	s := &set{seen: make(map[string]struct{}, len(members)), invalidAt: invalidAt}
	for _, member := range members {
		s.add(member)
	}

	return s
}

func (s *set) add(member string) {
	if member == "" {
		return
	}

	if _, found := s.seen[member]; !found {
		s.seen[member] = struct{}{}
		s.members = append(s.members, member)
	}
}

// snapshot returns a copy of the members, or false when the set expired.
func (s *set) snapshot(now time.Time) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.invalidAt.After(now) {
		return nil, false
	}

	return append([]string(nil), s.members...), true
}

// loadSet returns the set stored at key, converting a separator-joined value
// written with Set into a set, or creating an empty one.
func (provider *Default) loadSet(key string, now time.Time) *set {
	if value, ok := provider.m.Load(key); ok {
		if s, ok := value.(*set); ok {
			return s
		}
	}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	value, ok := provider.m.Load(key)
	if s, isSet := value.(*set); ok && isSet {
		return s
	}

	s := newSet(nil, now)
	if i, isItem := value.(item); ok && isItem && i.invalidAt.After(now) {
		s = newSet(strings.Split(string(i.value), ","), i.invalidAt)
	}

	provider.m.Store(key, s)

	return s
}

// AddToSet adds members to the set stored at key, extending its lifetime to
// at least the given duration.
func (provider *Default) AddToSet(key string, members []string, duration time.Duration) error {
	for {
		now := time.Now()
		s := provider.loadSet(key, now)

		s.mu.Lock()
		if !s.invalidAt.After(now) {
			s.seen = make(map[string]struct{})
			s.members = nil
		}
		if invalidAt := now.Add(duration); duration > 0 && invalidAt.After(s.invalidAt) {
			s.invalidAt = invalidAt
		}
		for _, member := range members {
			s.add(member)
		}
		s.mu.Unlock()

		// A concurrent Delete or Reset may have dropped the set while the
		// members were added, which would lose them.
		if current, ok := provider.m.Load(key); ok && current == any(s) {
			return nil
		}
	}
}

// GetSet returns all members of the set stored at key.
func (provider *Default) GetSet(key string) []string {
	value, ok := provider.m.Load(key)
	if !ok {
		return nil
	}

	now := time.Now()
	switch v := value.(type) {
	case *set:
		members, _ := v.snapshot(now)

		return members
	case item:
		if v.invalidAt.After(now) && len(v.value) > 0 {
			return strings.Split(string(v.value), ",")
		}
	}

	return nil
}

// WalkSets visits every set whose key matches the prefix, with the key
// stripped of it.
func (provider *Default) WalkSets(prefix string, fn func(key string, members []string) bool) error {
	now := time.Now()

	provider.m.Range(func(current, value any) bool {
		key, found := strings.CutPrefix(current.(string), prefix)
		if !found {
			return true
		}

		switch v := value.(type) {
		case *set:
			if members, alive := v.snapshot(now); alive {
				return fn(key, members)
			}
		case item:
			if v.invalidAt.After(now) {
				return fn(key, strings.Split(string(v.value), ","))
			}
		}

		return true
	})

	return nil
}
