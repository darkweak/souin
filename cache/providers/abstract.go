package providers

import (
	"regexp"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
)

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	GetRequestInCache(key string) types.ReverseResponse
	SetRequestInCache(key string, value []byte)
	DeleteRequestInCache(key string)
	Init() error
}

// PathnameNotInRegex check if pathname is in parameter regex var
func PathnameNotInRegex(pathname string, configuration configuration.Configuration) bool {
	b, _ := regexp.Match(configuration.Regex.Exclude, []byte(pathname))
	return !b
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// InitializeProviders allow to generate the providers array according to the configuration
func InitializeProviders(configuration configuration.Configuration) *[]AbstractProviderInterface {
	var providers []AbstractProviderInterface

	if len(configuration.Cache.Providers) == 0 || contains(configuration.Cache.Providers, "all") {
		providers = append(providers, MemoryConnectionFactory(configuration), RedisConnectionFactory(configuration))
	} else {
		if contains(configuration.Cache.Providers, "redis") {
			providers = append(providers, RedisConnectionFactory(configuration))
		}
		if contains(configuration.Cache.Providers, "memory") {
			providers = append(providers, MemoryConnectionFactory(configuration))
		}
	}

	return &providers
}
