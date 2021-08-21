package caddy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
)

type key string

const getterContextCtxKey key = "getter_context"
const moduleName = "cache"
const moduleID caddy.ModuleID = "http.handlers." + moduleName

func init() {
	caddy.RegisterModule(SouinCaddyPlugin{})
	httpcaddyfile.RegisterGlobalOption(moduleName, parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective(moduleName, parseCaddyfileHandlerDirective)
}

// SouinCaddyPlugin declaration.
type SouinCaddyPlugin struct {
	plugins.SouinBasePlugin
	Configuration *Configuration
	logger        *zap.Logger
	LogLevel      string `json:"log_level,omitempty"`
	bufPool       *sync.Pool
	Headers       []string                           `json:"headers,omitempty"`
	Badger        configurationtypes.CacheProvider   `json:"badger,omitempty"`
	Olric         configurationtypes.CacheProvider   `json:"olric,omitempty"`
	TTL           time.Duration                      `json:"ttl,omitempty"`
	YKeys         map[string]configurationtypes.YKey `json:"ykeys,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (SouinCaddyPlugin) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  moduleID,
		New: func() caddy.Module { return new(SouinCaddyPlugin) },
	}
}

type getterContext struct {
	rw   http.ResponseWriter
	req  *http.Request
	next caddyhttp.Handler
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (s *SouinCaddyPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	getterCtx := getterContext{rw, req, next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	buf := s.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer s.bufPool.Put(buf)
	combo := ctx.Value(getterContextCtxKey).(getterContext)
	plugins.DefaultSouinPluginCallback(rw, req, s.Retriever, s.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
		recorder := httptest.NewRecorder()
		e := combo.next.ServeHTTP(recorder, combo.req)
		if e != nil {
			return e
		}

		response := recorder.Result()
		req.Response = response
		response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(req)
		if e != nil {
			return e
		}

		_, e = io.Copy(rw, response.Body)

		return e
	})

	return nil
}

func (s *SouinCaddyPlugin) configurationPropertyMapper() error {
	if s.Configuration != nil {
		return nil
	}
	defaultCache := &DefaultCache{
		Badger:      s.Badger,
		Distributed: s.Olric.URL != "" || s.Olric.Path != "" || s.Olric.Configuration != nil,
		Headers:     s.Headers,
		Olric:       s.Olric,
		TTL:         s.TTL,
	}
	if s.Configuration == nil {
		s.Configuration = &Configuration{
			DefaultCache: defaultCache,
			LogLevel:     s.LogLevel,
		}
	}
	s.Configuration.Ykeys = s.YKeys
	s.Configuration.DefaultCache = defaultCache
	return nil
}

// Validate to validate configuration.
func (s *SouinCaddyPlugin) Validate() error {
	return nil
}

// FromApp to initialize configuration from App structure.
func (s *SouinCaddyPlugin) FromApp(app *SouinApp) error {
	if s.Configuration == nil {
		s.Configuration = &Configuration{
			URLs: make(map[string]configurationtypes.URL),
		}
	}

	if app.DefaultCache == nil {
		return nil
	}

	if s.Configuration.DefaultCache == nil {
		s.Configuration.DefaultCache = &DefaultCache{
			Headers: app.Headers,
			TTL:     app.TTL,
		}
	} else {
		dc := s.Configuration.DefaultCache
		appDc := app.DefaultCache
		if dc.Headers == nil {
			s.Configuration.DefaultCache.Headers = appDc.Headers
		}
		if dc.TTL == 0 {
			s.Configuration.DefaultCache.TTL = appDc.TTL
		}
		if dc.Olric.URL == "" || dc.Olric.Path == "" || dc.Olric.Configuration == nil {
			s.Configuration.DefaultCache.Distributed = appDc.Distributed
			s.Configuration.DefaultCache.Olric = appDc.Olric
		}
		if dc.Badger.Path == "" || dc.Badger.Configuration == nil {
			s.Configuration.DefaultCache.Badger = appDc.Badger
		}
	}

	return nil
}

// Provision to do the provisioning part.
func (s *SouinCaddyPlugin) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)

	if err := s.configurationPropertyMapper(); err != nil {
		return err
	}

	app, _ := ctx.App(moduleName)

	if err := s.FromApp(app.(*SouinApp)); err != nil {
		return err
	}

	s.bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(s.Configuration)
	// s.RequestCoalescing = coalescing.Initialize()
	return nil
}

func parseCaddyfileRecursively(h *caddyfile.Dispenser) interface{} {
	input := make(map[string]interface{})
	for nesting := h.Nesting(); h.NextBlock(nesting); {
		val := h.Val()
		if val == "}" || val == "{" {
			continue
		}
		args := h.RemainingArgs()
		if len(args) > 0 {
			input[val] = args[0]
		} else {
			input[val] = parseCaddyfileRecursively(h)
		}
	}

	return input
}

func parseCaddyfileGlobalOption(h *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
	souinApp := new(SouinApp)
	cfg := &Configuration{
		DefaultCache: &DefaultCache{
			Distributed: false,
			Headers:     []string{},
		},
		URLs: make(map[string]configurationtypes.URL),
	}

	for h.Next() {
		for nesting := h.Nesting(); h.NextBlock(nesting); {
			rootOption := h.Val()
			switch rootOption {
			case "headers":
				args := h.RemainingArgs()
				cfg.DefaultCache.Headers = append(cfg.DefaultCache.Headers, args...)
			case "log_level":
				args := h.RemainingArgs()
				cfg.LogLevel = args[0]
			case "badger":
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					}
				}
				cfg.DefaultCache.Badger = provider
			case "olric":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "url":
						urlArgs := h.RemainingArgs()
						provider.URL = urlArgs[0]
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					}
				}
				cfg.DefaultCache.Olric = provider
			case "ttl":
				args := h.RemainingArgs()
				ttl, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.TTL = ttl
				}
			default:
				return nil, h.Errf("unsupported root directive: %s", rootOption)
			}
		}
	}

	souinApp.DefaultCache = cfg.DefaultCache

	return httpcaddyfile.App{
		Name:  moduleName,
		Value: caddyconfig.JSON(souinApp, nil),
	}, nil
}

func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	dc := DefaultCache{}
	sc := Configuration{
		DefaultCache: &dc,
	}

	for h.Next() {
		directive := h.Val()
		switch directive {
		case "headers":
			sc.DefaultCache.Headers = h.RemainingArgs()
		case "ttl":
			ttl, err := time.ParseDuration(h.RemainingArgs()[0])
			if err == nil {
				sc.DefaultCache.TTL = ttl
			}
		}
	}

	return &SouinCaddyPlugin{
		Configuration: &sc,
	}, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*SouinCaddyPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyPlugin)(nil)
	_ caddy.Validator             = (*SouinCaddyPlugin)(nil)
)
