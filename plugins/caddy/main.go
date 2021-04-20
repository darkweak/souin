package caddy

import (
	"bytes"
	"context"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
)

type key string

const getterContextCtxKey key = "getter_context"
const moduleName = "souin_cache"
const moduleID caddy.ModuleID = "http.handlers." + moduleName

func init() {
	caddy.RegisterModule(SouinCaddyPlugin{})
	httpcaddyfile.RegisterGlobalOption(moduleName, parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective(moduleName, parseCaddyfileHandlerDirective)
}

var staticConfig Configuration

// SouinCaddyPlugin declaration.
type SouinCaddyPlugin struct {
	plugins.SouinBasePlugin
	Configuration *Configuration
	logger        *zap.Logger
	bufPool       sync.Pool
	Distributed   bool
	Headers       []string
	Olric         configurationtypes.CacheProvider
	TTL           string
	Rules         map[string]configurationtypes.URL
}

// CaddyModule returns the Caddy module information.
func (s SouinCaddyPlugin) CaddyModule() caddy.ModuleInfo {
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
func (s SouinCaddyPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	getterCtx := getterContext{rw, req, next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	buf := s.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer s.bufPool.Put(buf)
	combo := ctx.Value(getterContextCtxKey).(getterContext)
	coalescing.ServeResponse(rw, req, s.Retriever, plugins.DefaultSouinPluginCallback, s.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
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

// Validate to validate configuration.
func (s *SouinCaddyPlugin) Validate() error {
	s.logger.Info("Keep in mind the existing keys are always stored with the previous configuration. Use the API to purge existing keys")
	return nil
}

func (s *SouinCaddyPlugin) configurationPropertyMapper() error {
	if val, ok := s.Rules["*"]; ok {
		delete(s.Rules, "*")
		defaultCache := &DefaultCache{
			Distributed: s.Olric.URL != "",
			Headers:     val.Headers,
			Olric:       s.Olric,
			TTL:         val.TTL,
		}
		if s.Configuration == nil {
			s.Configuration = &Configuration{
				DefaultCache: defaultCache,
				URLs:         s.Rules,
			}
		}
		s.Configuration.DefaultCache = defaultCache
		return nil
	}

	for _, _ = range s.Configuration.URLs {
		return nil
	}

	return new(defaultCacheError)
}

// Provision to do the provisioning part.
func (s *SouinCaddyPlugin) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)
	if err := s.configurationPropertyMapper(); err != nil {
		return err
	}

	s.bufPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	if s.Configuration == nil && &staticConfig != nil {
		s.Configuration = &staticConfig
	}
	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(s.Configuration)
	s.RequestCoalescing = coalescing.Initialize()
	return nil
}

func parseCaddyfileGlobalOption(h *caddyfile.Dispenser) (interface{}, error) {
	var souin SouinCaddyPlugin
	cfg := &Configuration{
		DefaultCache: &DefaultCache{
			Distributed: false,
			Headers: []string{},
		},
		URLs: make(map[string]configurationtypes.URL),
	}

	for h.Next() {
		for nesting := h.Nesting(); h.NextBlock(nesting); {
			rootOption := h.Val()
			switch rootOption {
			case "distributed":
				args := h.RemainingArgs()
				distributed, _ := strconv.ParseBool(args[0])
				cfg.DefaultCache.Distributed = distributed
			case "headers":
				args := h.RemainingArgs()
				cfg.DefaultCache.Headers = append(cfg.DefaultCache.Headers, args...)
			case "log_level":
				args := h.RemainingArgs()
				cfg.LogLevel = args[0]
			case "olric":
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "url":
						urlArgs := h.RemainingArgs()
						provider.URL = urlArgs[0]
					}
				}
				cfg.DefaultCache.Olric = provider
			case "ttl":
				args := h.RemainingArgs()
				cfg.DefaultCache.TTL = args[0]
			default:
				return nil, h.Errf("unsupported root directive: %s", rootOption)
			}
		}
	}

	souin.Configuration = cfg
	staticConfig = *cfg
	return nil, nil
}

func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	s := &SouinCaddyPlugin{}
	if &staticConfig != nil {
		s.Configuration = &staticConfig
	}

	for h.Next() {
		for nesting := h.Nesting(); h.NextBlock(nesting); {
			switch h.Val() {
			case "route":
				args := h.RemainingArgs()
				url := configurationtypes.URL{}

				if v, ok := s.Configuration.URLs[args[0]]; ok {
					url = v
				}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "headers":
						headers := h.RemainingArgs()
						if args[0] == "*" {
							s.Configuration.DefaultCache.Headers = headers
						} else {
							url.Headers = append(s.Configuration.URLs[args[0]].Headers, headers...)
						}
					case "ttl":
						if args[0] == "*" {
							s.Configuration.DefaultCache.TTL = h.RemainingArgs()[0]
						} else {
							url.TTL = h.RemainingArgs()[0]
						}
					}
				}

				if args[0] != "*" {
					s.Configuration.URLs[args[0]] = url
				}
			}
		}
	}

	return s, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*SouinCaddyPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyPlugin)(nil)
	_ caddy.Validator             = (*SouinCaddyPlugin)(nil)
)
