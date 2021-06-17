package api

import (
	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
)

// Initialize contains all apis that should be enabled
func Initialize(provider types.AbstractProviderInterface, c configurationtypes.AbstractConfigurationInterface, ykeyStorage *ykeys.YKeyStorage) []EndpointInterface {
	security := auth.InitializeSecurity(c)
	return []EndpointInterface{security, initializeSouin(provider, c, security, ykeyStorage)}
}
