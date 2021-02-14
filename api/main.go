package api

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// Initialize contains all apis that should be enabled
func Initialize(provider types.AbstractProviderInterface, c configurationtypes.AbstractConfigurationInterface) []EndpointInterface {
	return []EndpointInterface{initializeSouin(provider, c)}
}
