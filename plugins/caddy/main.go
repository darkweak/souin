package caddy

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"go.uber.org/zap"
	"net/http"
)

func init() {
	caddy.RegisterModule(SouinCaddyPlugin{})
	httpcaddyfile.RegisterGlobalOption("souin", parseCaddyfileGlobalOption)
}

// SouinCaddyPlugin declaration.
type SouinCaddyPlugin struct {
	*plugins.SouinBasePlugin
	configuration Configuration
	logger        *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (s SouinCaddyPlugin) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.souin_system",
		New: func() caddy.Module { return new(SouinCaddyPlugin) },
	}
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (s SouinCaddyPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	coalescing.ServeResponse(rw, req, s.Retriever, plugins.DefaultSouinPluginCallback, s.RequestCoalescing)
	return next.ServeHTTP(rw, req)
}

func (s *SouinCaddyPlugin) Validate() error {
	s.logger.Info("Keep in mind the existing keys are always stored with the previous configuration. Use the API to purge existing keys")
	return nil
}

func (s *SouinCaddyPlugin) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)
	c := &Configuration{
		DefaultCache: configurationtypes.DefaultCache{
			TTL: "10000",
		},
	}
	s.RequestCoalescing = coalescing.Initialize()
	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(c)
	return nil
}

func parseCaddyfileGlobalOption(d *caddyfile.Dispenser) (interface{}, error) {
	for d.Next() {
		for d.NextBlock(0) {
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			fmt.Println(d.Val())
		}
	}

	return nil, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*SouinCaddyPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyPlugin)(nil)
	_ caddy.Validator             = (*SouinCaddyPlugin)(nil)
)
