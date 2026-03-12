package httpcache

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/api"

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

	go func() {
		currentApp := app.(*SouinApp)

		item := <-currentApp.onMiddlewareLoaded()

		config := Configuration{
			API: item.API,
			DefaultCache: DefaultCache{
				TTL: configurationtypes.Duration{
					Duration: 120 * time.Second,
				},
				MappingEvictionInterval: configurationtypes.Duration{
					Duration: time.Hour,
				},
			},
		}
		a.InternalEndpointHandlers = api.GenerateHandlerMap(&config, currentApp.Storers, item.SurrogateStorage)
	}()

	return nil
}

// Routes returns the admin routes.
func (a *adminAPI) Routes() []caddy.AdminRoute {
	return []caddy.AdminRoute{
		{
			Pattern: "/{params...}",
			Handler: caddy.AdminHandlerFunc(a.handleAPIEndpoints),
		},
	}
}
