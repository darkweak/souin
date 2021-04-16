package providers

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) map[string]types.AbstractProviderInterface {
	providers := make(map[string]types.AbstractProviderInterface)
	if len(configuration.GetDefaultCache().GetProviders()) == 0 || contains(configuration.GetDefaultCache().GetProviders(), "all") {
		redis, _ := RedisConnectionFactory(configuration)
		providers["redis"] = redis
		olric, _ := OlricConnectionFactory(configuration)
		providers["olric"] = olric
		ristretto, _ := RistrettoConnectionFactory(configuration)
		providers["ristretto"] = ristretto
	} else {
		if contains(configuration.GetDefaultCache().GetProviders(), "redis") {
			redis, _ := RedisConnectionFactory(configuration)
			providers["redis"] = redis
		}
		if contains(configuration.GetDefaultCache().GetProviders(), "olric") {
			olric, _ := OlricConnectionFactory(configuration)
			providers["olric"] = olric
		}
		if contains(configuration.GetDefaultCache().GetProviders(), "ristretto") {
			ristretto, _ := RistrettoConnectionFactory(configuration)
			providers["ristretto"] = ristretto
		}
	}

	for _, p := range providers {
		_ = p.Init()
	}
	return providers
}
