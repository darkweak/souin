package storage

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/pierrec/lz4"
	"go.uber.org/zap"
)

const (
	encodedHeaderSemiColonSeparator = "%3B"
	encodedHeaderColonSeparator     = "%3A"
	StalePrefix                     = "STALE_"
	surrogatePrefix                 = "SURROGATE_"
	MappingKeyPrefix                = "IDX_"
)

type StorerInstanciator func(configurationtypes.AbstractConfigurationInterface) (types.Storer, error)

var storageMap = map[string]StorerInstanciator{
	"etcd":           EtcdConnectionFactory,
	"redis":          RedisConnectionFactory,
	"olric":          OlricConnectionFactory,
	"otter":          OtterConnectionFactory,
	"embedded_olric": EmbeddedOlricConnectionFactory,
	"nuts":           NutsConnectionFactory,
	"badger":         BadgerConnectionFactory,
}

func getStorageNameFromConfiguration(configuration configurationtypes.AbstractConfigurationInterface) string {
	if configuration.GetDefaultCache().GetDistributed() {
		if configuration.GetDefaultCache().GetEtcd().Configuration != nil {
			return "etcd"
		} else if configuration.GetDefaultCache().GetRedis().Configuration != nil || configuration.GetDefaultCache().GetRedis().URL != "" {
			return "redis"
		} else {
			if configuration.GetDefaultCache().GetOlric().URL != "" {
				return "olric"
			} else {
				return "embedded_olric"
			}
		}
	} else if configuration.GetDefaultCache().GetOtter().Configuration != nil || configuration.GetDefaultCache().GetOtter().Path != "" {
		return "otter"
	} else if configuration.GetDefaultCache().GetNuts().Configuration != nil || configuration.GetDefaultCache().GetNuts().Path != "" {
		return "nuts"
	}

	return "badger"
}

func NewStorageFromName(name string) (StorerInstanciator, error) {
	if newStorage, found := storageMap[name]; found {
		return newStorage, nil
	}

	return nil, errors.New("Storer with name" + name + " not found")
}

func NewStorage(configuration configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
	storerName := getStorageNameFromConfiguration(configuration)
	if newStorage, found := storageMap[storerName]; found {
		return newStorage(configuration)
	}
	return nil, errors.New("Storer with name" + storerName + " not found")
}

func uniqueStorers(storers []string) []string {
	storerPresent := make(map[string]bool)
	s := []string{}

	for _, current := range storers {
		if _, found := storerPresent[current]; !found {
			storerPresent[current] = true
			s = append(s, current)
		}
	}

	return s
}

func NewStorages(configuration configurationtypes.AbstractConfigurationInterface) ([]types.Storer, error) {
	storers := []types.Storer{}
	for _, storerName := range uniqueStorers(configuration.GetDefaultCache().GetStorers()) {
		if newStorage, found := storageMap[storerName]; found {
			instance, err := newStorage(configuration)
			if err != nil {
				configuration.GetLogger().Sugar().Debugf("Cannot load configuration for the chianed provider %s: %+v", storerName, err)
				continue
			}

			configuration.GetLogger().Sugar().Debugf("Append storer %s to the chain", storerName)
			storers = append(storers, instance)
		} else {
			configuration.GetLogger().Sugar().Debugf("Storer with name %s not found", storerName)
		}
	}

	if len(storers) == 0 {
		configuration.GetLogger().Debug("Not able to create storers chain from the storers slice, fallback to the default storer creation")
		instance, err := NewStorage(configuration)
		if err != nil || instance == nil {
			return nil, err
		}

		storers = append(storers, instance)
	}

	names := []string{}
	for _, s := range storers {
		names = append(names, s.Name())
	}
	configuration.GetLogger().Sugar().Debugf("Run with %d chained providers with the given order %s", len(storers), strings.Join(names, ", "))
	return storers, nil
}

func varyVoter(baseKey string, req *http.Request, currentKey string) bool {
	if currentKey == baseKey {
		return true
	}

	if strings.Contains(currentKey, rfc.VarySeparator) && strings.HasPrefix(currentKey, baseKey+rfc.VarySeparator) {
		list := currentKey[(strings.LastIndex(currentKey, rfc.VarySeparator) + len(rfc.VarySeparator)):]
		if len(list) == 0 {
			return false
		}

		for _, item := range strings.Split(list, ";") {
			index := strings.LastIndex(item, ":")
			if len(item) < index+1 {
				return false
			}

			hVal := item[index+1:]
			if strings.Contains(hVal, encodedHeaderSemiColonSeparator) || strings.Contains(hVal, encodedHeaderColonSeparator) {
				hVal, _ = url.QueryUnescape(hVal)
			}
			if req.Header.Get(item[:index]) != hVal {
				return false
			}
		}

		return true
	}

	return false
}

func decodeMapping(item []byte) (mapping types.StorageMapper, e error) {
	e = gob.NewDecoder(bytes.NewBuffer(item)).Decode(&mapping)

	return
}

func mappingElection(provider types.Storer, item []byte, req *http.Request, validator *rfc.Revalidator, logger *zap.Logger) (resultFresh *http.Response, resultStale *http.Response, e error) {
	var mapping types.StorageMapper
	if len(item) != 0 {
		mapping, e = decodeMapping(item)
		if e != nil {
			return resultFresh, resultStale, e
		}
	}

	for keyName, keyItem := range mapping.Mapping {
		valid := true
		for hname, hval := range keyItem.VariedHeaders {
			if req.Header.Get(hname) != strings.Join(hval, ", ") {
				valid = false
				break
			}
		}

		if !valid {
			continue
		}

		rfc.ValidateETagFromHeader(keyItem.Etag, validator)
		if validator.Matched {
			// If the key is fresh enough.
			if time.Since(keyItem.FreshTime) < 0 {
				response := provider.Get(keyName)
				if response != nil {
					var buf bytes.Buffer
					_, _ = lz4.NewReader(&buf).Read(response)
					if resultFresh, e = http.ReadResponse(bufio.NewReader(&buf), req); e != nil {
						logger.Sugar().Errorf("An error occured while reading response for the key %s: %v", string(keyName), e)
						return
					}

					logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v", string(keyName), validator)
					return
				}
			}

			// If the key is still stale.
			if time.Since(keyItem.StaleTime) < 0 {
				response := provider.Get(keyName)
				if response != nil {
					var buf bytes.Buffer
					_, _ = lz4.NewReader(&buf).Read(response)
					if resultStale, e = http.ReadResponse(bufio.NewReader(&buf), req); e != nil {
						logger.Sugar().Errorf("An error occured while reading response for the key %s: %v", string(keyName), e)
						return
					}

					logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v as stale", string(keyName), validator)
				}
			}
		} else {
			logger.Sugar().Debugf("The stored key %s didn't match the current iteration key ETag %+v", string(keyName), validator)
		}
	}

	return
}

func mappingUpdater(key string, item []byte, logger *zap.Logger, now, freshTime, staleTime time.Time, variedHeaders http.Header, etag, realKey string) (val []byte, e error) {
	var mapping types.StorageMapper
	if len(item) == 0 {
		mapping = types.StorageMapper{}
	} else {
		e = gob.NewDecoder(bytes.NewBuffer(item)).Decode(&mapping)
		if e != nil {
			logger.Sugar().Errorf("Impossible to decode the key %s, %v", key, e)
			return nil, e
		}
	}

	if mapping.Mapping == nil {
		mapping.Mapping = make(map[string]types.KeyIndex)
	}

	mapping.Mapping[key] = types.KeyIndex{
		StoredAt:      now,
		FreshTime:     freshTime,
		StaleTime:     staleTime,
		VariedHeaders: variedHeaders,
		Etag:          etag,
		RealKey:       realKey,
	}

	buf := new(bytes.Buffer)
	e = gob.NewEncoder(buf).Encode(mapping)
	if e != nil {
		logger.Sugar().Errorf("Impossible to encode the mapping value for the key %s, %v", key, e)
		return nil, e
	}

	val = buf.Bytes()

	compressed := make([]byte, len(val))
	_, e = lz4.CompressBlock(val, compressed, nil)

	return val, e
}
