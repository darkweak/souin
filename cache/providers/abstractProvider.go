package providers

import (
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/cache/types"
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
func InitializeProvider(configuration configuration.AbstractConfigurationInterface) types.AbstractProviderInterface {
	return RistrettoConnectionFactory(configuration)
}
