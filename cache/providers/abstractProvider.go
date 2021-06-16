package providers

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) types.AbstractProviderInterface {
	var r types.AbstractProviderInterface
	if configuration.GetDefaultCache().GetDistributed() {
		if configuration.GetDefaultCache().GetOlric().URL != "" {
			r, _ = OlricConnectionFactory(configuration)
		} else {
			r, _ = EmbeddedOlricConnectionFactory(configuration)
		}
	} else {
		r, _ = BadgerConnectionFactory(configuration)
	}
	e := r.Init()
	if e != nil {
		panic(e)
	}
	return r
}
