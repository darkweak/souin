package simplefs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/darkweak/storages/core"
	"github.com/dustin/go-humanize"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pierrec/lz4/v4"
)

// Simplefs provider type.
type Simplefs struct {
	cache         *ttlcache.StringBytesCache
	stale         time.Duration
	size          int
	path          string
	logger        core.Logger
	actualSize    atomic.Int64
	directorySize int64
	mu            sync.Mutex
}

func onEvict(path string) error {
	return os.Remove(path)
}

// indexDirectory holds the persisted mapping metadata, relative to the storage
// path. It is a directory so it can never collide with an escaped cache key,
// and the stored responses stay alone at the root of the storage path.
const indexDirectory = ".simplefs-index"

// lz4FrameMagic is the magic number every lz4 frame starts with. Simplefs only
// ever writes lz4 frames in its directory, so it tells the files it owns from
// the ones that were already sitting there.
var lz4FrameMagic = []byte{0x04, 0x22, 0x4d, 0x18}

// isCacheFile reports whether the given file looks like a response simplefs
// stored. The storage path is not necessarily a directory simplefs owns, and it
// even defaults to the working directory when no path is configured, so a file
// is only ever removed once it has been recognized.
func isCacheFile(path string) bool {
	//nolint:gosec
	file, err := os.Open(path)
	if err != nil {
		return false
	}

	defer func() {
		_ = file.Close()
	}()

	header := make([]byte, len(lz4FrameMagic))
	if _, err := io.ReadFull(file, header); err != nil {
		return false
	}

	return bytes.Equal(header, lz4FrameMagic)
}

// Factory function create new Simplefs instance.
func Factory(simplefsCfg core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	var directorySize int64

	storagePath := simplefsCfg.Path
	size := 0
	directorySize = -1

	simplefsConfiguration := simplefsCfg.Configuration
	if simplefsConfiguration != nil {
		if sfsconfig, ok := simplefsConfiguration.(map[string]interface{}); ok {
			if v, found := sfsconfig["size"]; found && v != nil {
				if val, ok := v.(int); ok && val > 0 {
					size = val
				} else if val, ok := v.(float64); ok && val > 0 {
					size = int(val)
				} else if val, ok := v.(string); ok {
					size, _ = strconv.Atoi(val)
				}
			}

			if v, found := sfsconfig["path"]; found && v != nil {
				if val, ok := v.(string); ok {
					storagePath = val
				}
			}

			if v, found := sfsconfig["directory_size"]; found && v != nil {
				if val, ok := v.(int64); ok && val > 0 {
					directorySize = val
				} else if val, ok := v.(float64); ok && val > 0 {
					directorySize = int64(val)
				} else if val, ok := v.(string); ok && val != "" {
					s, _ := humanize.ParseBytes(val)
					//nolint:gosec
					directorySize = int64(s)
				}
			}
		}
	}

	var err error

	if storagePath == "" {
		logger.Info("No configuration path given, fallback to the current working directory.")

		storagePath, err = os.Getwd()
		if err != nil {
			logger.Errorf("Impossible to init the storage path in this working: %#v", err)

			return nil, err
		}
	}

	cache := ttlcache.NewStringBytes(uint64(size))

	if cache == nil {
		err = errors.New("impossible to initialize the simplefs storage")
		logger.Error(err)

		return nil, err
	}

	if err := os.MkdirAll(storagePath, 0o750); err != nil {
		logger.Errorf("impossible to create the storage directory: %#v", err)

		return nil, err
	}

	if err := os.MkdirAll(filepath.Join(storagePath, indexDirectory), 0o750); err != nil {
		logger.Errorf("impossible to create the index directory: %#v", err)

		return nil, err
	}

	logger.Infof("Created the storage directory %s if needed", storagePath)

	store := &Simplefs{cache: cache, directorySize: directorySize, logger: logger, path: storagePath, size: size, stale: stale}

	defer func() {
		go store.cache.Start()
	}()

	return store, nil
}

// Name returns the storer name.
func (provider *Simplefs) Name() string {
	return "SIMPLEFS"
}

// Uuid returns an unique identifier.
func (provider *Simplefs) Uuid() string {
	return fmt.Sprintf("%s-%d", provider.path, provider.size)
}

// MapKeys method returns a map with the key and value.
func (provider *Simplefs) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	provider.cache.Range(func(item *ttlcache.StringBytesItem) bool {
		if strings.HasPrefix(item.Key(), prefix) {
			k, _ := strings.CutPrefix(item.Key(), prefix)
			keys[k] = string(item.Value())
		}

		return true
	})

	return keys
}

// ListKeys method returns the list of existing keys.
func (provider *Simplefs) ListKeys() []string {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	return provider.cache.Keys()
}

// Get method returns the populated response if exists, empty response then.
func (provider *Simplefs) Get(key string) []byte {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	result := provider.cache.Get(key)
	if result == nil {
		provider.logger.Warnf("Impossible to get the key %s in Simplefs", key)

		return nil
	}

	if strings.HasPrefix(key, core.SurrogateKeyPrefix) {
		return result.Value()
	}

	byteValue, err := os.ReadFile(strings.Trim(string(result.Value()), ","))
	if err != nil {
		provider.logger.Errorf("Impossible to read the file %s from Simplefs: %#v", result.Value(), err)

		return result.Value()
	}

	return byteValue
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Simplefs) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	provider.mu.Lock()

	val := provider.cache.Get(core.MappingKeyPrefix + key)

	provider.mu.Unlock()

	if val == nil {
		provider.logger.Debugf("Impossible to get the mapping key %s in Simplefs", core.MappingKeyPrefix+key)

		return fresh, stale
	}

	fresh, stale, _ = core.MappingElection(provider, val.Value(), req, validator, provider.logger)

	return fresh, stale
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Simplefs) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	compressed := new(bytes.Buffer)
	writer := lz4.NewWriter(compressed)

	if _, err := writer.Write(value); err != nil {
		_ = writer.Close()

		provider.logger.Errorf("Impossible to compress the key %s into Simplefs, %v", variedKey, err)

		return err
	}

	if err := writer.Close(); err != nil {
		provider.logger.Errorf("Impossible to close the compressor for key %s into Simplefs, %v", variedKey, err)

		return err
	}

	provider.recoverEnoughSpaceIfNeeded(int64(compressed.Len()))

	joinedFP := filepath.Join(provider.path, url.PathEscape(variedKey))
	//nolint:gosec
	if err := os.WriteFile(joinedFP, compressed.Bytes(), 0o644); err != nil {
		provider.logger.Errorf("Impossible to write the file %s from Simplefs: %#v", variedKey, err)

		return nil
	}

	provider.mu.Lock()
	defer provider.mu.Unlock()

	_ = provider.cache.Set(variedKey, []byte(joinedFP), duration)

	mappingKey := core.MappingKeyPrefix + baseKey
	item := provider.cache.Get(mappingKey)

	if item == nil {
		provider.logger.Debugf("Impossible to get the mapping key %s in Simplefs", mappingKey)

		item = &ttlcache.StringBytesItem{}
	}

	val, e := core.MappingUpdater(variedKey, item.Value(), provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	provider.logger.Debugf("Store the new mapping for the key %s in Simplefs", variedKey)
	// Used to calculate -(now * 2)
	negativeNow, err := time.ParseDuration(fmt.Sprintf("-%ds", time.Now().Nanosecond()*2))
	if err != nil {
		return fmt.Errorf("impossible to generate the duration: %w", err)
	}

	_ = provider.cache.Set(mappingKey, val, negativeNow)
	provider.writeIndex(baseKey, val)

	return nil
}

// Set method will store the response in Simplefs provider.
func (provider *Simplefs) Set(key string, value []byte, duration time.Duration) error {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	_ = provider.cache.Set(key, value, duration)

	return nil
}

// Delete method will delete the response in Simplefs provider if exists corresponding to key param.
func (provider *Simplefs) Delete(key string) {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	provider.cache.Delete(key)
}

// DeleteMany method will delete the responses in Simplefs provider if exists corresponding to the regex key param.
func (provider *Simplefs) DeleteMany(key string) {
	rgKey, e := regexp.Compile(key)
	if e != nil {
		return
	}

	provider.cache.Range(func(item *ttlcache.StringBytesItem) bool {
		if rgKey.MatchString(item.Key()) {
			provider.Delete(item.Key())
		}

		return true
	})
}

// Init method will.
func (provider *Simplefs) Init() error {
	provider.cache.OnInsertion(func(_ context.Context, item *ttlcache.StringBytesItem) {
		if strings.Contains(item.Key(), core.MappingKeyPrefix) || strings.Contains(item.Key(), core.SurrogateKeyPrefix) {
			return
		}

		info, err := os.Stat(string(item.Value()))
		if err != nil {
			provider.logger.Errorf("impossible to get the file size %s: %#v", item.Key(), err)

			return
		}

		provider.logger.Debugf("Actual size add: %d, new: %d", provider.actualSize.Add(info.Size()), info.Size())
	})

	provider.cache.OnEviction(func(_ context.Context, _ ttlcache.EvictionReason, item *ttlcache.StringBytesItem) {
		if strings.HasPrefix(item.Key(), core.MappingKeyPrefix) {
			// The mapping left the index, its persisted copy has to go as well.
			provider.removeIndex(strings.TrimPrefix(item.Key(), core.MappingKeyPrefix))

			return
		}

		// The surrogate keys keep their value in memory, they are backed by no file.
		if strings.HasPrefix(item.Key(), core.SurrogateKeyPrefix) {
			return
		}

		info, err := os.Stat(string(item.Value()))
		if err != nil {
			provider.logger.Errorf("impossible to get the file size %s: %#v", item.Key(), err)

			return
		}

		provider.logger.Debugf("Actual size remove: %d, new: %d", provider.actualSize.Add(-info.Size()), info.Size())

		if err := onEvict(string(item.Value())); err != nil {
			provider.logger.Errorf("impossible to remove the file %s: %#v", item.Key(), err)
		}
	})

	provider.loadIndex()
	provider.purgeUnindexedFiles()

	return nil
}

// Reset method will reset or close provider.
func (provider *Simplefs) Reset() error {
	provider.mu.Lock()
	defer provider.mu.Unlock()

	provider.cache.DeleteAll()

	return nil
}

// recoverEnoughSpaceIfNeeded evicts the least recently used entries until the
// given size can be stored without exceeding the configured directory size.
//
// The eviction plan is computed in a single pass because the ttlcache eviction
// callback, which is in charge of decrementing the actual size, runs in its own
// goroutine: re-reading provider.actualSize after each deletion would observe a
// stale value and evict far more entries than necessary.
func (provider *Simplefs) recoverEnoughSpaceIfNeeded(size int64) {
	if provider.directorySize <= -1 {
		return
	}

	toFree := provider.actualSize.Load() + size - provider.directorySize
	if toFree <= 0 {
		return
	}

	// The eviction callback removes the file from the disk and updates the
	// actual size, so only the keys have to be collected here.
	keys := make([]string, 0)

	provider.mu.Lock()

	provider.cache.RangeBackwards(func(item *ttlcache.StringBytesItem) bool {
		// The mapping and the surrogate keys are stored in memory only, evicting
		// them wouldn't free any space on the disk.
		if strings.HasPrefix(item.Key(), core.MappingKeyPrefix) || strings.HasPrefix(item.Key(), core.SurrogateKeyPrefix) {
			return true
		}

		info, err := os.Stat(string(item.Value()))
		if err != nil {
			provider.logger.Errorf("impossible to get the file size %s: %#v", item.Key(), err)

			return true
		}

		keys = append(keys, item.Key())
		toFree -= info.Size()

		// Stop as soon as enough space has been planned to be recovered.
		return toFree > 0
	})

	provider.mu.Unlock()

	if toFree > 0 {
		provider.logger.Warnf("Impossible to recover enough space in %s to store %d bytes, the directory size limit may be exceeded", provider.path, size)
	}

	for _, key := range keys {
		provider.logger.Debugf("Evict the key %s from Simplefs to recover space", key)
		provider.Delete(key)
	}
}

// purgeUnindexedFiles removes the stored responses no cache entry points to.
//
// The simplefs index lives in memory only: the mapping metadata that makes a
// response reachable, its freshness and its varied headers, is never written on
// the disk. Whatever a previous process stored is therefore unreachable, and
// keeping it would leak the directory forever because the eviction can only
// free the entries it knows about, which in turn makes the directory_size limit
// impossible to enforce.
// See https://github.com/darkweak/storages/issues/60.
func (provider *Simplefs) purgeUnindexedFiles() {
	indexed := map[string]bool{}

	provider.mu.Lock()

	provider.cache.Range(func(item *ttlcache.StringBytesItem) bool {
		// Only the stored responses are backed by a file, the mapping and the
		// surrogate keys keep their value in memory.
		if strings.HasPrefix(item.Key(), core.MappingKeyPrefix) || strings.HasPrefix(item.Key(), core.SurrogateKeyPrefix) {
			return true
		}

		indexed[string(item.Value())] = true

		return true
	})

	provider.mu.Unlock()

	entries, err := os.ReadDir(provider.path)
	if err != nil {
		provider.logger.Errorf("impossible to read the storage directory %s: %#v", provider.path, err)

		return
	}

	var purged, purgedSize int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(provider.path, entry.Name())

		info, err := entry.Info()
		if err != nil {
			provider.logger.Errorf("impossible to get the file size %s: %#v", path, err)

			continue
		}

		// A file the index already points to is reachable, and its size was
		// accounted for when it got stored.
		if indexed[path] {
			continue
		}

		// A file simplefs did not write is none of its business, but it still
		// occupies the directory the limit is about.
		if !isCacheFile(path) {
			provider.actualSize.Add(info.Size())

			continue
		}

		if err := onEvict(path); err != nil {
			provider.logger.Errorf("impossible to remove the unindexed file %s: %#v", path, err)
			provider.actualSize.Add(info.Size())

			continue
		}

		purged++
		purgedSize += info.Size()
	}

	if purged > 0 {
		provider.logger.Infof(
			"Removed %d unreachable file(s), %d bytes, from %s: the simplefs index is kept in memory, so a response stored by a previous process can neither be served nor evicted.",
			purged, purgedSize, provider.path,
		)
	}
}

// indexPath returns the directory holding the persisted mapping metadata.
func (provider *Simplefs) indexPath() string {
	return filepath.Join(provider.path, indexDirectory)
}

// writeIndex persists the mapping of a base key, so the responses it references
// can be served again after a restart.
func (provider *Simplefs) writeIndex(baseKey string, mapping []byte) {
	directory := provider.indexPath()

	if err := os.MkdirAll(directory, 0o750); err != nil {
		provider.logger.Errorf("impossible to create the index directory %s: %#v", directory, err)

		return
	}

	// Write to a temporary file then rename it: the rename is atomic, so a
	// reader always sees either the previous mapping or the new one. A half
	// written index file would make every response it references unreachable.
	temporary, err := os.CreateTemp(directory, "tmp-")
	if err != nil {
		provider.logger.Errorf("impossible to create the temporary index file in %s: %#v", directory, err)

		return
	}

	if _, err := temporary.Write(mapping); err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporary.Name())

		provider.logger.Errorf("impossible to write the index of the key %s: %#v", baseKey, err)

		return
	}

	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporary.Name())

		provider.logger.Errorf("impossible to close the index of the key %s: %#v", baseKey, err)

		return
	}

	if err := os.Rename(temporary.Name(), filepath.Join(directory, url.PathEscape(baseKey))); err != nil {
		_ = os.Remove(temporary.Name())

		provider.logger.Errorf("impossible to store the index of the key %s: %#v", baseKey, err)
	}
}

// removeIndex drops the persisted mapping of a base key.
func (provider *Simplefs) removeIndex(baseKey string) {
	if err := os.Remove(filepath.Join(provider.indexPath(), url.PathEscape(baseKey))); err != nil && !errors.Is(err, os.ErrNotExist) {
		provider.logger.Errorf("impossible to remove the index of the key %s: %#v", baseKey, err)
	}
}

// restoreMapping puts back in the cache every response of the given mapping
// that can still be served, and returns the mapping stripped from the entries
// it had to drop, along with the number of restored responses.
func (provider *Simplefs) restoreMapping(mapping *core.StorageMapper) (pruned []byte, restored int64) {
	for variedKey, keyIndex := range mapping.GetMapping() {
		payload := filepath.Join(provider.path, url.PathEscape(variedKey))

		// Reproduce the state a process that never stopped would be in: the
		// entry lives until its fresh deadline, when the eviction removes the
		// file it points to.
		remaining := time.Until(keyIndex.GetFreshTime().AsTime())
		if remaining <= 0 {
			if err := onEvict(payload); err != nil && !errors.Is(err, os.ErrNotExist) {
				provider.logger.Errorf("impossible to remove the expired response %s: %#v", payload, err)
			}

			continue
		}

		if _, err := os.Stat(payload); err != nil {
			// The response is gone, the entry cannot be served anymore.
			continue
		}

		variedHeaders := http.Header{}
		for name, values := range keyIndex.GetVariedHeaders() {
			variedHeaders[name] = values.GetHeaderValue()
		}

		// core exposes no plain encoder, the kept entries are stacked back one
		// by one on top of an empty mapping.
		updated, err := core.MappingUpdater(
			variedKey, pruned, provider.logger,
			keyIndex.GetStoredAt().AsTime(), keyIndex.GetFreshTime().AsTime(), keyIndex.GetStaleTime().AsTime(),
			variedHeaders, keyIndex.GetEtag(), keyIndex.GetRealKey(),
		)
		if err != nil {
			continue
		}

		pruned = updated

		_ = provider.cache.Set(variedKey, []byte(payload), remaining)

		restored++
	}

	return pruned, restored
}

// loadIndex rebuilds the in memory index from the mappings a previous process
// persisted, so the stored responses survive a restart.
// See https://github.com/darkweak/storages/issues/60.
func (provider *Simplefs) loadIndex() {
	directory := provider.indexPath()

	entries, err := os.ReadDir(directory)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			provider.logger.Errorf("impossible to read the index directory %s: %#v", directory, err)
		}

		return
	}

	var restored, dropped int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		file := filepath.Join(directory, entry.Name())

		baseKey, err := url.PathUnescape(entry.Name())
		if err != nil {
			provider.logger.Errorf("impossible to decode the index file name %s, dropping it: %#v", file, err)
			provider.dropIndexFile(file)

			dropped++

			continue
		}

		//nolint:gosec
		content, err := os.ReadFile(file)
		if err != nil {
			provider.logger.Errorf("impossible to read the index file %s: %#v", file, err)

			continue
		}

		mapping, err := core.DecodeMapping(content)
		if err != nil {
			provider.logger.Errorf("impossible to decode the index file %s, dropping it: %#v", file, err)
			provider.dropIndexFile(file)

			dropped++

			continue
		}

		pruned, count := provider.restoreMapping(mapping)
		if count == 0 {
			// Not a single response of that base key can be served anymore.
			provider.dropIndexFile(file)

			dropped++

			continue
		}

		_ = provider.cache.Set(core.MappingKeyPrefix+baseKey, pruned, ttlcache.NoTTL)

		restored += count

		// Persist the pruned mapping so the dead entries do not pile up across
		// the restarts.
		if count != int64(len(mapping.GetMapping())) {
			provider.writeIndex(baseKey, pruned)
		}
	}

	if restored > 0 || dropped > 0 {
		provider.logger.Infof("Restored %d response(s) from the simplefs index in %s, dropped %d obsolete mapping(s).", restored, provider.path, dropped)
	}
}

// dropIndexFile removes an index file that cannot be used anymore.
func (provider *Simplefs) dropIndexFile(file string) {
	if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		provider.logger.Errorf("impossible to remove the index file %s: %#v", file, err)
	}
}
