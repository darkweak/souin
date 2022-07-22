package httpcache

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buraksezer/olric/config"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	surrogates_providers "github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
)

type key string

const getterContextCtxKey key = "getter_context"
const moduleName = "cache"

var up = caddy.NewUsagePool()

func init() {
	caddy.RegisterModule(SouinCaddyPlugin{})
	httpcaddyfile.RegisterGlobalOption(moduleName, parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective(moduleName, parseCaddyfileHandlerDirective)
}

// SouinCaddyPlugin development repository of the cache handler, allows
// the user to set up an HTTP cache system, RFC-7234 compliant and
// supports the tag based cache purge, distributed and not-distributed
// storage, key generation tweaking.
type SouinCaddyPlugin struct {
	plugins.SouinBasePlugin
	Configuration *Configuration
	logger        *zap.Logger
	cacheKeys     map[configurationtypes.RegValue]configurationtypes.Key
	// Log level.
	LogLevel string `json:"log_level,omitempty"`
	bufPool  *sync.Pool
	// Allowed HTTP verbs to be cached by the system.
	AllowedHTTPVerbs []string `json:"allowed_http_verbs,omitempty"`
	// Headers to add to the cache key if they are present.
	Headers []string `json:"headers,omitempty"`
	// Configure the Badger cache storage.
	Badger configurationtypes.CacheProvider `json:"badger,omitempty"`
	// Configure the global key generation.
	Key configurationtypes.Key `json:"key,omitempty"`
	// Override the cache key generation matching the pattern.
	CacheKeys map[string]configurationtypes.Key `json:"cache_keys,omitempty"`
	// Configure the Badger cache storage.
	Nuts configurationtypes.CacheProvider `json:"nuts,omitempty"`
	// Enable the Etcd distributed cache storage.
	Etcd configurationtypes.CacheProvider `json:"etcd,omitempty"`
	// Enable the Olric distributed cache storage.
	Olric configurationtypes.CacheProvider `json:"olric,omitempty"`
	// Time to live for a key, using time.duration.
	TTL configurationtypes.Duration `json:"ttl,omitempty"`
	// Time to live for a stale key, using time.duration.
	Stale configurationtypes.Duration `json:"stale,omitempty"`
	// The default Cache-Control header value if none set by the upstream server.
	DefaultCacheControl string `json:"default_cache_control,omitempty"`
	// The cache name to use in the Cache-Status response header.
	CacheName string `json:"cache_name,omitempty"`
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
	req := s.Retriever.GetContext().SetBaseContext(r)
	if b, handler := s.HandleInternally(req); b {
		handler(rw, req)
		return nil
	}

	if !plugins.CanHandle(req, s.Retriever) {
		rfc.MissCache(rw.Header().Set, req)
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
			rfc.MissCache(customWriter.Header().Set, req)
			return e
		}

		customWriter.Response.Header = customWriter.Rw.Header().Clone()
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
		Nuts:                s.Nuts,
		Key:                 s.Key,
		DefaultCacheControl: s.DefaultCacheControl,
		CacheName:           s.CacheName,
		Distributed:         s.Olric.URL != "" || s.Olric.Path != "" || s.Olric.Configuration != nil || s.Etcd.Configuration != nil,
		Headers:             s.Headers,
		Olric:               s.Olric,
		Etcd:                s.Etcd,
		TTL:                 s.TTL,
		Stale:               s.Stale,
	}
	if s.Configuration == nil {
		s.Configuration = &Configuration{
			cacheKeys:    s.cacheKeys,
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
			Key:                 app.Key,
			TTL:                 app.TTL,
			Stale:               app.Stale,
			DefaultCacheControl: app.DefaultCacheControl,
			CacheName:           app.CacheName,
		}
		return nil
	}
	if s.Configuration.cacheKeys == nil {
		s.Configuration.cacheKeys = make(map[configurationtypes.RegValue]configurationtypes.Key)
	}
	if s.CacheKeys == nil {
		s.CacheKeys = app.CacheKeys
	}
	for k, v := range s.CacheKeys {
		s.Configuration.cacheKeys[configurationtypes.RegValue{Regexp: regexp.MustCompile(k)}] = v
	}

	dc := s.Configuration.DefaultCache
	appDc := app.DefaultCache
	s.Configuration.DefaultCache.AllowedHTTPVerbs = append(s.Configuration.DefaultCache.AllowedHTTPVerbs, appDc.AllowedHTTPVerbs...)
	s.Configuration.DefaultCache.CDN = app.DefaultCache.CDN
	if dc.Headers == nil {
		s.Configuration.DefaultCache.Headers = appDc.Headers
	}
	if dc.TTL.Duration == 0 {
		s.Configuration.DefaultCache.TTL = appDc.TTL
	}
	if dc.Stale.Duration == 0 {
		s.Configuration.DefaultCache.Stale = appDc.Stale
	}
	if !dc.Key.DisableBody && !dc.Key.DisableHost && !dc.Key.DisableMethod {
		s.Configuration.DefaultCache.Key = appDc.Key
	}
	if dc.DefaultCacheControl == "" {
		s.Configuration.DefaultCache.DefaultCacheControl = appDc.DefaultCacheControl
	}
	if dc.CacheName == "" {
		s.Configuration.DefaultCache.CacheName = appDc.CacheName
	}
	if dc.Olric.URL == "" || dc.Olric.Path == "" || dc.Olric.Configuration == nil {
		s.Configuration.DefaultCache.Distributed = appDc.Distributed
		s.Configuration.DefaultCache.Olric = appDc.Olric
	}
	if dc.Etcd.Configuration == nil {
		s.Configuration.DefaultCache.Distributed = appDc.Distributed
		s.Configuration.DefaultCache.Etcd = appDc.Etcd
	}
	if dc.Badger.Path == "" || dc.Badger.Configuration == nil {
		s.Configuration.DefaultCache.Badger = appDc.Badger
	}
	if dc.Nuts.Path == "" || dc.Nuts.Configuration == nil {
		s.Configuration.DefaultCache.Nuts = appDc.Nuts
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
	surrogates, ok := up.LoadOrStore(surrogate_key, s.Retriever.GetTransport().GetSurrogateKeys())
	if ok {
		s.Retriever.GetTransport().SetSurrogateKeys(surrogates.(surrogates_providers.SurrogateInterface))
	}

	dc := s.Retriever.GetConfiguration().GetDefaultCache()
	if dc.GetDistributed() {
		if eo, ok := s.Retriever.GetProvider().(*providers.EmbeddedOlric); ok {
			name := fmt.Sprintf("0.0.0.0:%d", config.DefaultPort)
			if dc.GetOlric().Configuration != nil {
				oc := dc.GetOlric().Configuration.(*config.Config)
				name = fmt.Sprintf("%s:%d", oc.BindAddr, oc.BindPort)
			} else if dc.GetOlric().Path != "" {
				name = dc.GetOlric().Path
			}

			key := "Embedded-" + name
			v, _ := up.LoadOrStore(stored_providers_key, newStorageProvider())
			v.(*storage_providers).Add(key)

			if eo.GetDM() == nil {
				v, l, e := up.LoadOrNew(key, func() (caddy.Destructor, error) {
					s.logger.Sugar().Debug("Create a new olric instance.")
					return providers.EmbeddedOlricConnectionFactory(s.Configuration)
				})

				if l && e == nil {
					s.Retriever.(*types.RetrieverResponseProperties).Provider = v.(types.AbstractProviderInterface)
					s.Retriever.GetTransport().(*rfc.VaryTransport).Provider = v.(types.AbstractProviderInterface)
				}
			} else {
				s.logger.Sugar().Debug("Store the olric instance.")
				_, _ = up.LoadOrStore(key, s.Retriever.GetProvider())
			}
		}
	}

	v, l := up.LoadOrStore(coalescing_key, s.Retriever.GetTransport().GetCoalescingLayerStorage())

	if l {
		s.logger.Sugar().Debug("Loaded coalescing layer from cache.")
		s.Retriever.GetTransport().GetCoalescingLayerStorage().Destruct()
		s.Retriever.GetTransport().(*rfc.VaryTransport).CoalescingLayerStorage = v.(*types.CoalescingLayerStorage)
	}

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
		if len(args) == 1 {
			input[val] = args[0]
		} else if len(args) > 1 {
			input[val] = args
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
			CacheName:           "",
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
			case "cache_keys":
				cacheKeys := cfg.CfgCacheKeys
				if cacheKeys == nil {
					cacheKeys = make(map[string]configurationtypes.Key)
				}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					rg := h.Val()
					ck := configurationtypes.Key{}

					for nesting := h.Nesting(); h.NextBlock(nesting); {
						directive := h.Val()
						switch directive {
						case "disable_body":
							ck.DisableBody = true
						case "disable_host":
							ck.DisableHost = true
						case "disable_method":
							ck.DisableMethod = true
						}
					}

					cacheKeys[rg] = ck
				}
				cfg.CfgCacheKeys = cacheKeys
			case "cache_name":
				args := h.RemainingArgs()
				cfg.DefaultCache.CacheName = args[0]
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
				cfg.DefaultCache.DefaultCacheControl = strings.Join(args, " ")
			case "etcd":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					}
				}
				cfg.DefaultCache.Etcd = provider
			case "headers":
				cfg.DefaultCache.Headers = append(cfg.DefaultCache.Headers, h.RemainingArgs()...)
			case "key":
				config_key := configurationtypes.Key{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "disable_body":
						config_key.DisableBody = true
					case "disable_host":
						config_key.DisableHost = true
					case "disable_method":
						config_key.DisableMethod = true
					}
				}
				cfg.DefaultCache.Key = config_key
			case "log_level":
				args := h.RemainingArgs()
				cfg.LogLevel = args[0]
			case "nuts":
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
				cfg.DefaultCache.Nuts = provider
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
	souinApp.CacheKeys = cfg.CfgCacheKeys

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
		case "cache_keys":
			cacheKeys := sc.CfgCacheKeys
			if cacheKeys == nil {
				cacheKeys = make(map[string]configurationtypes.Key)
			}
			for nesting := h.Nesting(); h.NextBlock(nesting); {
				val := h.Val()
				ck := configurationtypes.Key{}

				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "disable_body":
						ck.DisableBody = true
					case "disable_host":
						ck.DisableHost = true
					case "disable_method":
						ck.DisableMethod = true
					}
				}

				cacheKeys[val] = ck
			}
			sc.CfgCacheKeys = cacheKeys
		case "cache_name":
			args := h.RemainingArgs()
			sc.DefaultCache.CacheName = args[0]
		case "default_cache_control":
			sc.DefaultCache.DefaultCacheControl = strings.Join(h.RemainingArgs(), " ")
		case "headers":
			sc.DefaultCache.Headers = h.RemainingArgs()
		case "key":
			config_key := configurationtypes.Key{}
			for nesting := h.Nesting(); h.NextBlock(nesting); {
				directive := h.Val()
				switch directive {
				case "disable_body":
					config_key.DisableBody = true
				case "disable_host":
					config_key.DisableHost = true
				case "disable_method":
					config_key.DisableMethod = true
				}
			}
			sc.DefaultCache.Key = config_key
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
		CacheKeys:     sc.CfgCacheKeys,
	}, nil
}

// Interface guards
var (
	_ caddy.CleanerUpper          = (*SouinCaddyPlugin)(nil)
	_ caddy.Provisioner           = (*SouinCaddyPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyPlugin)(nil)
)
