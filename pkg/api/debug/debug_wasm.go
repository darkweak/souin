//go:build wasi || wasm

package debug

import (
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

// DebugAPI object contains informations related to the endpoints
type DebugAPI struct {
	basePath string
	enabled  bool
}

type DefaultHandler struct{}

func (d *DefaultHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

// InitializeDebug initialize the debug endpoints
func InitializeDebug(_ configurationtypes.AbstractConfigurationInterface) *DebugAPI {
	return &DebugAPI{}
}

// GetBasePath will return the basepath for this resource
func (p *DebugAPI) GetBasePath() string {
	return p.basePath
}

// IsEnabled will return enabled status
func (p *DebugAPI) IsEnabled() bool {
	return false
}

// HandleRequest will handle the request
func (p *DebugAPI) HandleRequest(_ http.ResponseWriter, _ *http.Request) {}
