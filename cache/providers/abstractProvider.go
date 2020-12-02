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
	if len(configuration.GetDefaultCache().Providers) == 0 || contains(configuration.GetDefaultCache().Providers, "all") {
		redis, _ := RedisConnectionFactory(configuration)
		providers["redis"] = redis
		ristretto, _ := RistrettoConnectionFactory(configuration)
		providers["ristretto"] = ristretto
	} else {
		if contains(configuration.GetDefaultCache().Providers, "redis") {
			redis, _ := RedisConnectionFactory(configuration)
			providers["redis"] = redis
		}
		if contains(configuration.GetDefaultCache().Providers, "ristretto") {
			ristretto, _ := RistrettoConnectionFactory(configuration)
			providers["ristretto"] = ristretto
		}
	}
	return providers
}
