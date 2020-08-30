package providers

import (
	"regexp"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
)

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	GetRequestInCache(key string) types.ReverseResponse
	SetRequestInCache(key string, value []byte, url configuration.URL)
	DeleteRequestInCache(key string)
	Init() error
}

// PathnameNotInExcludeRegex check if pathname is in parameter regex var
func PathnameNotInExcludeRegex(pathname string, configuration configuration.Configuration) bool {
	b, _ := regexp.Match(configuration.DefaultCache.Regex.Exclude, []byte(pathname))
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
func InitializeProviders(configuration configuration.Configuration) map[string]AbstractProviderInterface {
	providers := make(map[string]AbstractProviderInterface)

	if len(configuration.DefaultCache.Providers) == 0 || contains(configuration.DefaultCache.Providers, "all") {
		providers["memory"] = MemoryConnectionFactory(configuration)
		providers["redis"] = RedisConnectionFactory(configuration)
	} else {
		if contains(configuration.DefaultCache.Providers, "redis") {
			providers["redis"] = RedisConnectionFactory(configuration)
		}
		if contains(configuration.DefaultCache.Providers, "memory") {
			providers["memory"] = MemoryConnectionFactory(configuration)
		}
	}

	return providers
}
