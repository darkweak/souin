package storage

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
)

const (
	encodedHeaderSemiColonSeparator = "%3B"
	encodedHeaderColonSeparator     = "%3A"
	StalePrefix                     = "STALE_"
)

type Storer interface {
	ListKeys() []string
	Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response
	Get(key string) []byte
	Set(key string, value []byte, url configurationtypes.URL, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Name() string
	Reset() error
}

type StorerInstanciator func(configurationtypes.AbstractConfigurationInterface) (Storer, error)

var storageMap = map[string]StorerInstanciator{
	"etcd":           EtcdConnectionFactory,
	"redis":          RedisConnectionFactory,
	"olric":          OlricConnectionFactory,
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
	} else if configuration.GetDefaultCache().GetNuts().Configuration != nil || configuration.GetDefaultCache().GetNuts().Path != "" {
		return "nuts"
	}

	return "badger"
}

func NewStorage(configuration configurationtypes.AbstractConfigurationInterface) (Storer, error) {
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

func NewStorages(configuration configurationtypes.AbstractConfigurationInterface) ([]Storer, error) {
	storers := []Storer{}
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
