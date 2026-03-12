//go:build wasi || wasm

package core

import (
	"bufio"
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/pierrec/lz4/v4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

const MappingKeyPrefix = "IDX_"

func DecodeMapping(item []byte) (*StorageMapper, error) {
	mapping := &StorageMapper{}
	e := proto.Unmarshal(item, mapping)

	return mapping, e
}

func MappingElection(provider Storer, item []byte, req *http.Request, validator *Revalidator, logger Logger) (resultFresh *http.Response, resultStale *http.Response, e error) {
	mapping := &StorageMapper{}

	if len(item) != 0 {
		mapping, e = DecodeMapping(item)
		if e != nil {
			return resultFresh, resultStale, e
		}
	}

	for keyName, keyItem := range mapping.GetMapping() {
		valid := true

		for hname, hval := range keyItem.GetVariedHeaders() {
			if req.Header.Get(hname) != strings.Join(hval.GetHeaderValue(), ", ") {
				valid = false

				break
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
					bufW := new(bytes.Buffer)
					reader := lz4.NewReader(bytes.NewBuffer(response))
					_, _ = reader.WriteTo(bufW)

					if resultFresh, e = http.ReadResponse(bufio.NewReader(bufW), req); e != nil {
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
					bufW := new(bytes.Buffer)
					reader := lz4.NewReader(bytes.NewBuffer(response))
					_, _ = reader.WriteTo(bufW)

					if resultStale, e = http.ReadResponse(bufio.NewReader(bufW), req); e != nil {
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
	mapping := &StorageMapper{}
	if len(item) != 0 {
		e = proto.Unmarshal(item, mapping)
		if e != nil {
			logger.Errorf("Impossible to decode the key %s, %v", key, e)

			return nil, e
		}
	}

	if mapping.GetMapping() == nil {
		mapping.Mapping = make(map[string]*KeyIndex)
	}

	var pbvariedeheader map[string]*KeyIndexStringList
	if variedHeaders != nil {
		pbvariedeheader = make(map[string]*KeyIndexStringList)
	}

	for k, v := range variedHeaders {
		pbvariedeheader[k] = &KeyIndexStringList{HeaderValue: v}
	}

	mapping.Mapping[key] = &KeyIndex{
		StoredAt:      timestamppb.New(now),
		FreshTime:     timestamppb.New(freshTime),
		StaleTime:     timestamppb.New(staleTime),
		VariedHeaders: pbvariedeheader,
		Etag:          etag,
		RealKey:       realKey,
	}

	val, e = proto.Marshal(mapping)
	if e != nil {
		logger.Errorf("Impossible to encode the mapping value for the key %s, %v", key, e)

		return nil, e
	}

	return val, e
}
