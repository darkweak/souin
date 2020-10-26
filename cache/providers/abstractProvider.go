package providers

import (
	"github.com/darkweak/souin/configuration_types"
	"github.com/darkweak/souin/cache/types"
)

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configuration_types.AbstractConfigurationInterface) types.AbstractProviderInterface {
	r, _ := RistrettoConnectionFactory(configuration)
	return r
}
