package providers

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) types.AbstractProviderInterface {
	r, _ := RistrettoConnectionFactory(configuration)
	return r
}
