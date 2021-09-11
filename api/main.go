package api

import (
	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
)

type MapHandler struct {
	Handlers *map[string]http.HandlerFunc
}

func GenerateHandlerMap(
	configuration configurationtypes.AbstractConfigurationInterface,
	provider types.AbstractProviderInterface,
	ykeyStorage *ykeys.YKeyStorage,
) *MapHandler {
	hm := make(map[string]http.HandlerFunc)
	shouldEnable := false

	souinAPI := configuration.GetAPI()
	basePathAPIS := souinAPI.BasePath
	if basePathAPIS == "" {
		basePathAPIS = "/souin-api"
	}

	for _, endpoint := range Initialize(provider, configuration, ykeyStorage) {
		if endpoint.IsEnabled() {
			shouldEnable = true
			hm[basePathAPIS+endpoint.GetBasePath()] = endpoint.(*SouinAPI).HandleRequest
		}
	}

	if shouldEnable {
		return &MapHandler{Handlers: &hm}
	}

	return nil
}

// Initialize contains all apis that should be enabled
func Initialize(provider types.AbstractProviderInterface, c configurationtypes.AbstractConfigurationInterface, ykeyStorage *ykeys.YKeyStorage) []EndpointInterface {
	security := auth.InitializeSecurity(c)
	return []EndpointInterface{security, initializeSouin(provider, c, security, ykeyStorage)}
}
