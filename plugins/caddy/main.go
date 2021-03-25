package caddy

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/plugins"
	"go.uber.org/zap"
	"net/http"
)

func init() {
	caddy.RegisterModule(SouinCaddyPlugin{})
	httpcaddyfile.RegisterGlobalOption("souin", parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective("souin", parseCaddyfileHandlerDirective)
}

var staticConfig Configuration

// SouinCaddyPlugin declaration.
type SouinCaddyPlugin struct {
	plugins.SouinBasePlugin
	configuration     *Configuration
	logger            *zap.Logger
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
	coalescing.ServeResponse(rw, req, s.Retriever, plugins.DefaultSouinPluginCallback, s.RequestCoalescing, next.ServeHTTP)
	return nil
}

// Validate to validate configuration.
func (s *SouinCaddyPlugin) Validate() error {
	s.logger.Info("Keep in mind the existing keys are always stored with the previous configuration. Use the API to purge existing keys")
	return nil
}

// Provision to do the provisioning part.
func (s *SouinCaddyPlugin) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)
	if s.configuration == nil && &staticConfig != nil {
		s.configuration = &staticConfig
	}
	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(s.configuration)
	s.RequestCoalescing = coalescing.Initialize()
	return nil
}

func parseCaddyfileGlobalOption(d *caddyfile.Dispenser) (interface{}, error) {
	p := NewParser()

	for d.Next() {
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			v := d.Val()
			d.NextArg()
			v2 := d.Val()

			p.WriteLine(v, v2)
		}
	}

	var s SouinCaddyPlugin
	err := staticConfig.Parse([]byte(p.str))
	s.configuration = &staticConfig
	return nil, err
}

func parseCaddyfileHandlerDirective(_ httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	s := &SouinCaddyPlugin{}
	if &staticConfig != nil {
		s.configuration = &staticConfig
	}

	return s, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*SouinCaddyPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyPlugin)(nil)
	_ caddy.Validator             = (*SouinCaddyPlugin)(nil)
)
