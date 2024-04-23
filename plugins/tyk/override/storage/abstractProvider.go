package storage

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
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

func NewStorageFromName(_ string) (StorerInstanciator, error) {
	return CacheConnectionFactory, nil
}

func NewStorages(configuration configurationtypes.AbstractConfigurationInterface) ([]types.Storer, error) {
	s, err := CacheConnectionFactory(configuration)
	return []types.Storer{s}, err
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
					if resultFresh, e = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(response)), req); e != nil {
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
					if resultStale, e = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(response)), req); e != nil {
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

	return val, e
}
