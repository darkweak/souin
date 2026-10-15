package core

// Yaegi compatible replacement of github.com/darkweak/storages/core.
// It keeps the upstream API but drops protobuf, which crashes the Yaegi
// compiler: mappings are encoded as JSON instead.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pierrec/lz4/v4"
)

type Storer interface {
	MapKeys(prefix string) map[string]string
	ListKeys() []string
	Get(key string) []byte
	Set(key string, value []byte, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Name() string
	Uuid() string
	Reset() error

	// Multi level storer to handle fresh/stale at once
	GetMultiLevel(key string, req *http.Request, validator *Revalidator) (fresh *http.Response, stale *http.Response)
	SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error
}

// CacheProvider config.
type CacheProvider struct {
	// URL to connect to the storage system.
	URL string `json:"url" yaml:"url"`
	// Path to the configuration file.
	Path string `json:"path" yaml:"path"`
	// Declare the cache provider directly in the Souin configuration.
	Configuration any `json:"configuration" yaml:"configuration"`
}

const (
	DISABLE_VARY_CTX   = "storages_bypass_vary"
	MappingKeyPrefix   = "IDX_"
	SurrogateKeyPrefix = "SURROGATE_"
)

func DecodeMapping(item []byte) (*StorageMapper, error) {
	mapping := new(StorageMapper)
	e := json.Unmarshal(item, mapping)

	return mapping, e
}

func readResponse(data []byte, req *http.Request) (*http.Response, error) {
	reader := lz4.NewReader(bytes.NewReader(data))

	return http.ReadResponse(bufio.NewReader(reader), req)
}

func newKeyIndex(now, freshTime, staleTime time.Time, variedHeaders http.Header, etag, realKey string) *KeyIndex {
	var variedHeadersList map[string]*KeyIndexStringList
	if variedHeaders != nil {
		variedHeadersList = make(map[string]*KeyIndexStringList, len(variedHeaders))
	}

	for headerName, headerValues := range variedHeaders {
		variedHeadersList[headerName] = &KeyIndexStringList{HeaderValue: headerValues}
	}

	return &KeyIndex{
		StoredAt:      NewTimestamp(now),
		FreshTime:     NewTimestamp(freshTime),
		StaleTime:     NewTimestamp(staleTime),
		VariedHeaders: variedHeadersList,
		Etag:          etag,
		RealKey:       realKey,
	}
}

func MappingElection(provider Storer, item []byte, req *http.Request, validator *Revalidator, logger Logger) (resultFresh *http.Response, resultStale *http.Response, e error) {
	mapping := new(StorageMapper)

	if len(item) != 0 {
		mapping, e = DecodeMapping(item)
		if e != nil {
			return resultFresh, resultStale, e
		}
	}

	return electMapping(provider, mapping.GetMapping(), req, validator, logger)
}

// MappingElectionEntries elects a fresh/stale response from per-variant
// entries, where each value is one marshalled KeyIndex as stored by storers
// keeping one field per varied key instead of one blob per base key.
func MappingElectionEntries(provider Storer, entries map[string][]byte, req *http.Request, validator *Revalidator, logger Logger) (*http.Response, *http.Response, error) {
	mapping := make(map[string]*KeyIndex, len(entries))

	for keyName, raw := range entries {
		keyItem := &KeyIndex{}
		if err := json.Unmarshal(raw, keyItem); err != nil {
			logger.Errorf("Impossible to decode the mapping entry %s: %v", keyName, err)

			continue
		}

		mapping[keyName] = keyItem
	}

	return electMapping(provider, mapping, req, validator, logger)
}

func electMapping(provider Storer, mapping map[string]*KeyIndex, req *http.Request, validator *Revalidator, logger Logger) (resultFresh *http.Response, resultStale *http.Response, e error) {
	for keyName, keyItem := range mapping {
		valid := true

		if req.Context().Value(DISABLE_VARY_CTX) == nil || !req.Context().Value(DISABLE_VARY_CTX).(bool) {
			for hname, hval := range keyItem.GetVariedHeaders() {
				if req.Header.Get(hname) != strings.Join(hval.GetHeaderValue(), ", ") {
					valid = false

					break
				}
			}
		}

		if !valid {
			continue
		}

		ValidateETagFromHeader(keyItem.GetEtag(), validator)

		if validator.Matched {
			// If the key is fresh enough.
			if time.Since(keyItem.GetFreshTime().AsTime()) < 0 {
				response := provider.Get(keyName)
				if response != nil {
					if resultFresh, e = readResponse(response, req); e != nil {
						logger.Errorf("An error occurred while reading response for the key %s: %v", keyName, e)

						return resultFresh, resultStale, e
					}

					logger.Debugf("The stored key %s matched the current iteration key ETag %+v", keyName, validator)

					return resultFresh, resultStale, e
				}
			}

			// If the key is still stale.
			if time.Since(keyItem.GetStaleTime().AsTime()) < 0 {
				response := provider.Get(keyName)
				if response != nil {
					if resultStale, e = readResponse(response, req); e != nil {
						logger.Errorf("An error occurred while reading response for the key %s: %v", keyName, e)

						return resultFresh, resultStale, e
					}

					logger.Debugf("The stored key %s matched the current iteration key ETag %+v as stale", keyName, validator)
				}
			}
		} else {
			logger.Debugf("The stored key %s didn't match the current iteration key ETag %+v", keyName, validator)
		}
	}

	return resultFresh, resultStale, e
}

func MappingUpdater(key string, item []byte, logger Logger, now, freshTime, staleTime time.Time, variedHeaders http.Header, etag, realKey string) (val []byte, e error) {
	mapping := new(StorageMapper)
	if len(item) != 0 {
		e = json.Unmarshal(item, mapping)
		if e != nil {
			logger.Errorf("Impossible to decode the key %s, %v", key, e)

			return nil, e
		}
	}

	if mapping.GetMapping() == nil {
		mapping.Mapping = make(map[string]*KeyIndex)
	}

	mapping.Mapping[key] = newKeyIndex(now, freshTime, staleTime, variedHeaders, etag, realKey)

	val, e = json.Marshal(mapping)
	if e != nil {
		logger.Errorf("Impossible to encode the mapping value for the key %s, %v", key, e)

		return nil, e
	}

	return val, e
}
