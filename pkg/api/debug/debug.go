//go:build !wasi && !wasm

package debug

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/darkweak/souin/configurationtypes"
)

// DebugAPI object contains informations related to the endpoints
type DebugAPI struct {
	basePath string
	enabled  bool
}

type DefaultHandler struct{}

func (d *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pprof.Index(w, r)
}

// InitializeDebug initialize the debug endpoints
func InitializeDebug(configuration configurationtypes.AbstractConfigurationInterface) *DebugAPI {
	basePath := configuration.GetAPI().Debug.BasePath
	enabled := configuration.GetAPI().Debug.Enable
	if basePath == "" {
		basePath = "/debug/"
	}

	return &DebugAPI{
		basePath,
		enabled,
	}
}

// GetBasePath will return the basepath for this resource
func (p *DebugAPI) GetBasePath() string {
	return p.basePath
}

// IsEnabled will return enabled status
func (p *DebugAPI) IsEnabled() bool {
	return p.enabled
}

// HandleRequest will handle the request
func (p *DebugAPI) HandleRequest(w http.ResponseWriter, r *http.Request) {
	var executor http.Handler
	executor = &DefaultHandler{}

	if strings.Contains(r.RequestURI, "allocs") {
		executor = pprof.Handler("allocs")
	}
	if strings.Contains(r.RequestURI, "cmdline") {
		executor = pprof.Handler("cmdline")
	}
	if strings.Contains(r.RequestURI, "profile") {
		executor = pprof.Handler("profile")
	}
	if strings.Contains(r.RequestURI, "symbol") {
		executor = pprof.Handler("symbol")
	}
	if strings.Contains(r.RequestURI, "trace") {
		executor = pprof.Handler("trace")
	}
	if strings.Contains(r.RequestURI, "goroutine") {
		executor = pprof.Handler("goroutine")
	}
	if strings.Contains(r.RequestURI, "heap") {
		executor = pprof.Handler("heap")
	}
	if strings.Contains(r.RequestURI, "block") {
		executor = pprof.Handler("block")
	}
	if strings.Contains(r.RequestURI, "heap") {
		executor = pprof.Handler("heap")
	}
	if strings.Contains(r.RequestURI, "mutex") {
		executor = pprof.Handler("mutex")
	}
	if strings.Contains(r.RequestURI, "threadcreate") {
		executor = pprof.Handler("threadcreate")
	}

	executor.ServeHTTP(w, r)
}
