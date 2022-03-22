package httpcache

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
)

type key string

const getterContextCtxKey key = "getter_context"
const moduleName = "cache"

func init() {
	caddy.RegisterModule(SouinCaddyPlugin{})
	httpcaddyfile.RegisterGlobalOption(moduleName, parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective(moduleName, parseCaddyfileHandlerDirective)
}

// SouinCaddyPlugin declaration.
type SouinCaddyPlugin struct {
	plugins.SouinBasePlugin
	Configuration       *Configuration
	logger              *zap.Logger
	LogLevel            string `json:"log_level,omitempty"`
	bufPool             *sync.Pool
	AllowedHTTPVerbs    []string                         `json:"allowed_http_verbs,omitempty"`
	Headers             []string                         `json:"headers,omitempty"`
	Badger              configurationtypes.CacheProvider `json:"badger,omitempty"`
	Olric               configurationtypes.CacheProvider `json:"olric,omitempty"`
	TTL                 configurationtypes.Duration      `json:"ttl,omitempty"`
	DefaultCacheControl string                           `json:"default_cache_control,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (SouinCaddyPlugin) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.cache",
		New: func() caddy.Module { return new(SouinCaddyPlugin) },
	}
}

type getterContext struct {
	rw   http.ResponseWriter
	req  *http.Request
	next caddyhttp.Handler
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (s *SouinCaddyPlugin) ServeHTTP(rw http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	req := s.Retriever.GetContext().Method.SetContext(r)
	if b, handler := s.HandleInternally(req); b {
		handler(rw, req)
		return nil
	}

	if !plugins.CanHandle(req, s.Retriever) {
		rw.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
		return next.ServeHTTP(rw, r)
	}

	req = s.Retriever.GetContext().SetContext(req)
	customWriter := &plugins.CustomWriter{
		Response: &http.Response{},
		Buf:      s.bufPool.Get().(*bytes.Buffer),
		Rw:       rw,
	}
	getterCtx := getterContext{customWriter, req, next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	if plugins.HasMutation(req, rw) {
		return next.ServeHTTP(rw, r)
	}
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	combo := ctx.Value(getterContextCtxKey).(getterContext)

	return plugins.DefaultSouinPluginCallback(customWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
		var e error
		if e = combo.next.ServeHTTP(customWriter, r); e != nil {
			return e
		}

		combo.req.Response = customWriter.Response
		if combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req); e != nil {
			return e
		}

		_, _ = customWriter.Send()
		return e
	})
}

func (s *SouinCaddyPlugin) configurationPropertyMapper() error {
	if s.Configuration != nil {
		return nil
	}
	defaultCache := &DefaultCache{
		Badger:              s.Badger,
		DefaultCacheControl: s.DefaultCacheControl,
		Distributed:         s.Olric.URL != "" || s.Olric.Path != "" || s.Olric.Configuration != nil,
		Headers:             s.Headers,
		Olric:               s.Olric,
		TTL:                 s.TTL,
	}
	if s.Configuration == nil {
		s.Configuration = &Configuration{
			DefaultCache: defaultCache,
			LogLevel:     s.LogLevel,
		}
	}
	s.Configuration.DefaultCache = defaultCache
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

	s.Configuration.API = app.API

	if s.Configuration.DefaultCache == nil {
		s.Configuration.DefaultCache = &DefaultCache{
			AllowedHTTPVerbs:    app.DefaultCache.AllowedHTTPVerbs,
			Headers:             app.Headers,
			TTL:                 app.TTL,
			DefaultCacheControl: app.DefaultCacheControl,
		}
		return nil
	}

	dc := s.Configuration.DefaultCache
	appDc := app.DefaultCache
	s.Configuration.DefaultCache.AllowedHTTPVerbs = append(s.Configuration.DefaultCache.AllowedHTTPVerbs, appDc.AllowedHTTPVerbs...)
	if dc.Headers == nil {
		s.Configuration.DefaultCache.Headers = appDc.Headers
	}
	if dc.TTL.Duration == 0 {
		s.Configuration.DefaultCache.TTL = appDc.TTL
	}
	if dc.DefaultCacheControl == "" {
		s.Configuration.DefaultCache.DefaultCacheControl = appDc.DefaultCacheControl
	}
	if dc.Olric.URL == "" || dc.Olric.Path == "" || dc.Olric.Configuration == nil {
		s.Configuration.DefaultCache.Distributed = appDc.Distributed
		s.Configuration.DefaultCache.Olric = appDc.Olric
	}
	if dc.Badger.Path == "" || dc.Badger.Configuration == nil {
		s.Configuration.DefaultCache.Badger = appDc.Badger
	}
	if dc.Regex.Exclude == "" {
		s.Configuration.DefaultCache.Regex.Exclude = appDc.Regex.Exclude
	}

	return nil
}

// Provision to do the provisioning part.
func (s *SouinCaddyPlugin) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)

	if err := s.configurationPropertyMapper(); err != nil {
		return err
	}

	ctxApp, _ := ctx.App(moduleName)
	app := ctxApp.(*SouinApp)

	if err := s.FromApp(app); err != nil {
		return err
	}

	s.bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(s.Configuration)
	if app.Provider == nil {
		app.Provider = s.Retriever.GetProvider()
	} else {
		s.Retriever.(*types.RetrieverResponseProperties).Provider = app.Provider
		s.Retriever.GetTransport().(*rfc.VaryTransport).Provider = app.Provider
	}

	if app.SurrogateStorage == nil {
		app.SurrogateStorage = s.Retriever.GetTransport().GetSurrogateKeys()
	} else {
		s.Retriever.GetTransport().SetSurrogateKeys(app.SurrogateStorage)
	}

	s.RequestCoalescing = coalescing.Initialize()
	s.MapHandler = api.GenerateHandlerMap(s.Configuration, s.Retriever.GetTransport())
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

func parseBadgerConfiguration(c map[string]interface{}) map[string]interface{} {
	for k, v := range c {
		switch k {
		case "Dir", "ValueDir":
			c[k] = v
		case "SyncWrites", "ReadOnly", "InMemory", "MetricsEnabled", "CompactL0OnClose", "LmaxCompaction", "VerifyValueChecksum", "BypassLockGuard", "DetectConflicts":
			c[k] = true
		case "NumVersionsToKeep", "NumGoroutines", "MemTableSize", "BaseTableSize", "BaseLevelSize", "LevelSizeMultiplier", "TableSizeMultiplier", "MaxLevels", "ValueThreshold", "NumMemtables", "BlockSize", "BlockCacheSize", "IndexCacheSize", "NumLevelZeroTables", "NumLevelZeroTablesStall", "ValueLogFileSize", "NumCompactors", "ZSTDCompressionLevel", "ChecksumVerificationMode", "NamespaceOffset":
			c[k], _ = strconv.Atoi(v.(string))
		case "Compression", "ValueLogMaxEntries":
			c[k], _ = strconv.ParseUint(v.(string), 10, 32)
		case "VLogPercentile", "BloomFalsePositive":
			c[k], _ = strconv.ParseFloat(v.(string), 64)
		case "EncryptionKey":
			c[k] = []byte(v.(string))
		case "EncryptionKeyRotationDuration":
			c[k], _ = time.ParseDuration(v.(string))
		}
	}

	return c
}

func parseCaddyfileGlobalOption(h *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
	souinApp := new(SouinApp)
	cfg := &Configuration{
		DefaultCache: &DefaultCache{
			AllowedHTTPVerbs: make([]string, 0),
			Distributed:      false,
			Headers:          []string{},
			TTL: configurationtypes.Duration{
				Duration: 120 * time.Second,
			},
			DefaultCacheControl: "",
		},
		URLs: make(map[string]configurationtypes.URL),
	}

	for h.Next() {
		for nesting := h.Nesting(); h.NextBlock(nesting); {
			rootOption := h.Val()
			switch rootOption {
			case "allowed_http_verbs":
				allowed := cfg.DefaultCache.AllowedHTTPVerbs
				allowed = append(allowed, h.RemainingArgs()...)
				cfg.DefaultCache.AllowedHTTPVerbs = allowed
			case "api":
				apiConfiguration := configurationtypes.API{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "basepath":
						apiConfiguration.BasePath = h.RemainingArgs()[0]
					case "prometheus":
						apiConfiguration.Prometheus = configurationtypes.APIEndpoint{}
						apiConfiguration.Prometheus.Enable = true
						for nesting := h.Nesting(); h.NextBlock(nesting); {
							directive := h.Val()
							switch directive {
							case "basepath":
								apiConfiguration.Prometheus.BasePath = h.RemainingArgs()[0]
							}
						}
					case "souin":
						apiConfiguration.Souin = configurationtypes.APIEndpoint{}
						apiConfiguration.Souin.Enable = true
						for nesting := h.Nesting(); h.NextBlock(nesting); {
							directive := h.Val()
							switch directive {
							case "basepath":
								apiConfiguration.Souin.BasePath = h.RemainingArgs()[0]
							}
						}
					}
				}
				cfg.API = apiConfiguration
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
						provider.Configuration = parseBadgerConfiguration(provider.Configuration.(map[string]interface{}))
					}
				}
				cfg.DefaultCache.Badger = provider
			case "cdn":
				cdn := configurationtypes.CDN{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "api_key":
						cdn.APIKey = h.RemainingArgs()[0]
					case "dynamic":
						cdn.Dynamic = true
					case "hostname":
						cdn.Hostname = h.RemainingArgs()[0]
					case "network":
						cdn.Network = h.RemainingArgs()[0]
					case "provider":
						cdn.Provider = h.RemainingArgs()[0]
					case "strategy":
						cdn.Strategy = h.RemainingArgs()[0]
					}
				}
				cfg.DefaultCache.CDN = cdn
			case "default_cache_control":
				args := h.RemainingArgs()
				cfg.DefaultCache.DefaultCacheControl = args[0]
			case "headers":
				cfg.DefaultCache.Headers = append(cfg.DefaultCache.Headers, h.RemainingArgs()...)
			case "log_level":
				args := h.RemainingArgs()
				cfg.LogLevel = args[0]
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
			case "regex":
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "exclude":
						cfg.DefaultCache.Regex.Exclude = h.RemainingArgs()[0]
					}
				}
			case "stale":
				args := h.RemainingArgs()
				stale, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.Stale.Duration = stale
				}
			case "ttl":
				args := h.RemainingArgs()
				ttl, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.TTL.Duration = ttl
				}
			default:
				return nil, h.Errf("unsupported root directive: %s", rootOption)
			}
		}
	}

	souinApp.DefaultCache = cfg.DefaultCache
	souinApp.API = cfg.API

	return httpcaddyfile.App{
		Name:  moduleName,
		Value: caddyconfig.JSON(souinApp, nil),
	}, nil
}

func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	dc := DefaultCache{
		AllowedHTTPVerbs: make([]string, 0),
	}
	sc := Configuration{
		DefaultCache: &dc,
	}

	for h.Next() {
		directive := h.Val()
		switch directive {
		case "allowed_http_verbs":
			allowed := sc.DefaultCache.AllowedHTTPVerbs
			allowed = append(allowed, h.RemainingArgs()...)
			sc.DefaultCache.AllowedHTTPVerbs = allowed
		case "badger":
			provider := configurationtypes.CacheProvider{}
			for nesting := h.Nesting(); h.NextBlock(nesting); {
				directive := h.Val()
				switch directive {
				case "path":
					urlArgs := h.RemainingArgs()
					provider.Path = urlArgs[0]
				case "configuration":
					provider.Configuration = parseCaddyfileRecursively(h.Dispenser)
					provider.Configuration = parseBadgerConfiguration(provider.Configuration.(map[string]interface{}))
				}
			}
			sc.DefaultCache.Badger = provider
		case "default_cache_control":
			sc.DefaultCache.DefaultCacheControl = h.RemainingArgs()[0]
		case "headers":
			sc.DefaultCache.Headers = h.RemainingArgs()
		case "stale":
			stale, err := time.ParseDuration(h.RemainingArgs()[0])
			if err == nil {
				sc.DefaultCache.Stale.Duration = stale
			}
		case "ttl":
			ttl, err := time.ParseDuration(h.RemainingArgs()[0])
			if err == nil {
				sc.DefaultCache.TTL.Duration = ttl
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
)
