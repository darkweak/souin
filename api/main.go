package api

import (
	"net/http"

	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/api/prometheus"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// MapHandler is a map to store the available http Handlers
type MapHandler struct {
	Handlers *map[string]http.HandlerFunc
}

// GenerateHandlerMap generate the MapHandler
func GenerateHandlerMap(
	configuration configurationtypes.AbstractConfigurationInterface,
	transport types.TransportInterface,
) *MapHandler {
	hm := make(map[string]http.HandlerFunc)
	shouldEnable := false

	souinAPI := configuration.GetAPI()
	basePathAPIS := souinAPI.BasePath
	if basePathAPIS == "" {
		basePathAPIS = "/souin-api"
	}

	for _, endpoint := range Initialize(transport, configuration) {
		if endpoint.IsEnabled() {
			shouldEnable = true
			hm[basePathAPIS+endpoint.GetBasePath()] = endpoint.HandleRequest
		}
	}

	if shouldEnable {
		return &MapHandler{Handlers: &hm}
	}

	return nil
}

// Initialize contains all apis that should be enabled
func Initialize(transport types.TransportInterface, c configurationtypes.AbstractConfigurationInterface) []EndpointInterface {
	security := auth.InitializeSecurity(c)
	return []EndpointInterface{security, initializeSouin(c, security, transport), prometheus.InitializePrometheus(c, security)}
}
