package api

import (
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/surrogate/providers"
)

// MapHandler is a map to store the available http Handlers
type MapHandler struct {
	Handlers *map[string]http.HandlerFunc
}

// GenerateHandlerMap generate the MapHandler
func GenerateHandlerMap(
	configuration configurationtypes.AbstractConfigurationInterface,
	storers []storage.Storer,
	surrogateStorage providers.SurrogateInterface,
) *MapHandler {
	hm := make(map[string]http.HandlerFunc)
	shouldEnable := false

	souinAPI := configuration.GetAPI()
	basePathAPIS := souinAPI.BasePath
	if basePathAPIS == "" {
		basePathAPIS = "/souin-api"
	}

	for _, endpoint := range Initialize(configuration, storers, surrogateStorage) {
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
func Initialize(c configurationtypes.AbstractConfigurationInterface, storers []storage.Storer, surrogateStorage providers.SurrogateInterface) []EndpointInterface {
	return []EndpointInterface{initializeSouin(c, storers, surrogateStorage)}
}
