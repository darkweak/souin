package httpcache

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/api"
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/storages/core"
)

func init() {
	caddy.RegisterModule(new(adminAPI))
}

// adminAPI is a module that serves PKI endpoints to retrieve
// information about the CAs being managed by Caddy.
type adminAPI struct {
	ctx                      caddy.Context
	logger                   core.Logger
	app                      *SouinApp
	InternalEndpointHandlers *api.MapHandler
}

// CaddyModule returns the Caddy module information.
func (adminAPI) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "admin.api.souin",
		New: func() caddy.Module { return new(adminAPI) },
	}
}

func (a *adminAPI) handleAPIEndpoints(writer http.ResponseWriter, request *http.Request) error {
	if a.InternalEndpointHandlers != nil {
		for k, handler := range *a.InternalEndpointHandlers.Handlers {
			if strings.Contains(request.RequestURI, k) {
				handler(writer, request)
				return nil
			}
		}
	}

	return caddy.APIError{
		HTTPStatus: http.StatusNotFound,
		Err:        fmt.Errorf("resource not found: %v", request.URL.Path),
	}
}

// Provision sets up the adminAPI module.
func (a *adminAPI) Provision(ctx caddy.Context) error {
	a.ctx = ctx
	a.logger = ctx.Logger(a).Sugar()

	app, err := ctx.App(moduleName)
	if err != nil {
		return err
	}

	a.app = app.(*SouinApp)
	config := Configuration{
		API: a.app.API,
		DefaultCache: DefaultCache{
			TTL: configurationtypes.Duration{
				Duration: 120 * time.Second,
			},
		},
	}
	a.InternalEndpointHandlers = api.GenerateHandlerMap(&config, a.app.Storers, a.app.SurrogateStorage)

	return nil
}

// Routes returns the admin routes for the PKI app.
func (a *adminAPI) Routes() []caddy.AdminRoute {
	basepath := "/souin-api"
	if a.app != nil && a.app.API.BasePath != "" {
		basepath = a.app.API.BasePath
	}

	return []caddy.AdminRoute{
		{
			Pattern: basepath + "/{params...}",
			Handler: caddy.AdminHandlerFunc(a.handleAPIEndpoints),
		},
	}
}
