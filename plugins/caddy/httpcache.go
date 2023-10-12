package httpcache

import (
	"fmt"
	"net/http"
	"time"

	"github.com/buraksezer/olric/config"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/pkg/storage"
	surrogates_providers "github.com/darkweak/souin/pkg/surrogate/providers"
	"go.uber.org/zap"
)

const moduleName = "cache"

var up = caddy.NewUsagePool()

func init() {
	caddy.RegisterModule(SouinCaddyMiddleware{})
	httpcaddyfile.RegisterGlobalOption(moduleName, parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective(moduleName, parseCaddyfileHandlerDirective)
}

// SouinCaddyMiddleware development repository of the cache handler, allows
// the user to set up an HTTP cache system, RFC-7234 compliant and
// supports the tag based cache purge, distributed and not-distributed
// storage, key generation tweaking.
type SouinCaddyMiddleware struct {
	*middleware.SouinBaseHandler
	logger        *zap.Logger
	Configuration Configuration
	// Logger level, fallback on caddy's one when not redefined.
	LogLevel string `json:"log_level,omitempty"`
	// Allowed HTTP verbs to be cached by the system.
	AllowedHTTPVerbs []string `json:"allowed_http_verbs,omitempty"`
	// Headers to add to the cache key if they are present.
	Headers []string `json:"headers,omitempty"`
	// Configure the Badger cache storage.
	Badger configurationtypes.CacheProvider `json:"badger,omitempty"`
	// Configure the global key generation.
	Key configurationtypes.Key `json:"key,omitempty"`
	// Override the cache key generation matching the pattern.
	CacheKeys configurationtypes.CacheKeys `json:"cache_keys,omitempty"`
	// Configure the Badger cache storage.
	Nuts configurationtypes.CacheProvider `json:"nuts,omitempty"`
	// Enable the Etcd distributed cache storage.
	Etcd configurationtypes.CacheProvider `json:"etcd,omitempty"`
	// Enable the Redis distributed cache storage.
	Redis configurationtypes.CacheProvider `json:"redis,omitempty"`
	// Enable the Olric distributed cache storage.
	Olric configurationtypes.CacheProvider `json:"olric,omitempty"`
	// Time to live for a key, using time.duration.
	Timeout configurationtypes.Timeout `json:"timeout,omitempty"`
	// Time to live for a key, using time.duration.
	TTL configurationtypes.Duration `json:"ttl,omitempty"`
	// Time to live for a stale key, using time.duration.
	Stale configurationtypes.Duration `json:"stale,omitempty"`
	// Storage providers chaining and order.
	Storers []string `json:"storers,omitempty"`
	// The default Cache-Control header value if none set by the upstream server.
	DefaultCacheControl string `json:"default_cache_control,omitempty"`
	// The cache name to use in the Cache-Status response header.
	CacheName string `json:"cache_name,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (SouinCaddyMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.cache",
		New: func() caddy.Module { return new(SouinCaddyMiddleware) },
	}
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (s *SouinCaddyMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	return s.SouinBaseHandler.ServeHTTP(rw, r, func(w http.ResponseWriter, _ *http.Request) error {
		return next.ServeHTTP(w, r)
	})
}

func (s *SouinCaddyMiddleware) configurationPropertyMapper() error {
	defaultCache := DefaultCache{
		Badger:              s.Badger,
		Nuts:                s.Nuts,
		Key:                 s.Key,
		DefaultCacheControl: s.DefaultCacheControl,
		CacheName:           s.CacheName,
		Distributed:         s.Olric.URL != "" || s.Olric.Path != "" || s.Olric.Configuration != nil || s.Etcd.Configuration != nil || s.Redis.URL != "" || s.Redis.Configuration != nil,
		Headers:             s.Headers,
		Olric:               s.Olric,
		Etcd:                s.Etcd,
		Redis:               s.Redis,
		Timeout:             s.Timeout,
		TTL:                 s.TTL,
		Stale:               s.Stale,
		Storers:             s.Storers,
	}
	// if s.Configuration == nil {
	// 	s.Configuration = Configuration{
	// 		CacheKeys:    s.cacheKeys,
	// 		DefaultCache: defaultCache,
	// 		LogLevel:     s.LogLevel,
	// 	}
	// }
	s.Configuration.DefaultCache = defaultCache
	return nil
}

// FromApp to initialize configuration from App structure.
func (s *SouinCaddyMiddleware) FromApp(app *SouinApp) error {
	if s.Configuration.GetDefaultCache() == nil {
		s.Configuration = Configuration{
			URLs: make(map[string]configurationtypes.URL),
		}
	}

	if app.DefaultCache.GetTTL() == 0 {
		return nil
	}

	s.Configuration.API = app.API

	if s.Configuration.GetDefaultCache() == nil {
		s.Configuration.DefaultCache = DefaultCache{
			AllowedHTTPVerbs:    app.DefaultCache.AllowedHTTPVerbs,
			Headers:             app.Headers,
			Key:                 app.Key,
			TTL:                 app.TTL,
			Stale:               app.Stale,
			DefaultCacheControl: app.DefaultCacheControl,
			CacheName:           app.CacheName,
			Timeout:             app.Timeout,
		}
		return nil
	}
	if s.Configuration.CacheKeys == nil || len(s.Configuration.CacheKeys) == 0 {
		s.Configuration.CacheKeys = configurationtypes.CacheKeys{}
	}
	if s.CacheKeys == nil {
		s.CacheKeys = app.CacheKeys
	}
	for _, cacheKey := range s.Configuration.CacheKeys {
		for k, v := range cacheKey {
			s.Configuration.CacheKeys = append(
				s.Configuration.CacheKeys,
				map[configurationtypes.RegValue]configurationtypes.Key{k: v},
			)
		}
	}

	dc := s.Configuration.DefaultCache
	appDc := app.DefaultCache
	s.Configuration.DefaultCache.AllowedHTTPVerbs = append(s.Configuration.DefaultCache.AllowedHTTPVerbs, appDc.AllowedHTTPVerbs...)
	s.Configuration.DefaultCache.CDN = app.DefaultCache.CDN
	if dc.Headers == nil {
		s.Configuration.DefaultCache.Headers = appDc.Headers
	}

	if s.Configuration.LogLevel == "" {
		s.Configuration.LogLevel = app.LogLevel
	}
	if dc.TTL.Duration == 0 {
		s.Configuration.DefaultCache.TTL = appDc.TTL
	}
	if dc.Stale.Duration == 0 {
		s.Configuration.DefaultCache.Stale = appDc.Stale
	}
	if len(dc.Storers) == 0 {
		s.Configuration.DefaultCache.Storers = appDc.Storers
	}
	if dc.Timeout.Backend.Duration == 0 {
		s.Configuration.DefaultCache.Timeout.Backend = appDc.Timeout.Backend
	}
	if dc.Timeout.Cache.Duration == 0 {
		s.Configuration.DefaultCache.Timeout.Cache = appDc.Timeout.Cache
	}
	if !dc.Key.DisableBody && !dc.Key.DisableHost && !dc.Key.DisableMethod && !dc.Key.DisableQuery && !dc.Key.Hide && len(dc.Key.Headers) == 0 {
		s.Configuration.DefaultCache.Key = appDc.Key
	}
	if dc.DefaultCacheControl == "" {
		s.Configuration.DefaultCache.DefaultCacheControl = appDc.DefaultCacheControl
	}
	if dc.CacheName == "" {
		s.Configuration.DefaultCache.CacheName = appDc.CacheName
	}
	if dc.Etcd.Configuration == nil && dc.Redis.URL == "" && dc.Redis.Path == "" && dc.Redis.Configuration == nil && dc.Olric.URL == "" && dc.Olric.Path == "" && dc.Olric.Configuration == nil {
		s.Configuration.DefaultCache.Distributed = appDc.Distributed
	}
	if dc.Olric.URL == "" && dc.Olric.Path == "" && dc.Olric.Configuration == nil {
		s.Configuration.DefaultCache.Olric = appDc.Olric
	}
	if dc.Redis.URL == "" && dc.Redis.Path == "" && dc.Redis.Configuration == nil {
		s.Configuration.DefaultCache.Redis = appDc.Redis
	}
	if dc.Etcd.Configuration == nil {
		s.Configuration.DefaultCache.Etcd = appDc.Etcd
	}
	if dc.Badger.Path == "" || dc.Badger.Configuration == nil {
		s.Configuration.DefaultCache.Badger = appDc.Badger
	}
	if dc.Nuts.Path == "" && dc.Nuts.Configuration == nil {
		s.Configuration.DefaultCache.Nuts = appDc.Nuts
	}
	if dc.Regex.Exclude == "" {
		s.Configuration.DefaultCache.Regex.Exclude = appDc.Regex.Exclude
	}

	return nil
}

// Provision to do the provisioning part.
func (s *SouinCaddyMiddleware) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)

	if err := s.configurationPropertyMapper(); err != nil {
		return err
	}

	s.Configuration.SetLogger(s.logger)
	ctxApp, _ := ctx.App(moduleName)
	app := ctxApp.(*SouinApp)

	if err := s.FromApp(app); err != nil {
		return err
	}

	bh := middleware.NewHTTPCacheHandler(&s.Configuration)
	surrogates, ok := up.LoadOrStore(surrogate_key, bh.SurrogateKeyStorer)
	if ok {
		bh.SurrogateKeyStorer = surrogates.(surrogates_providers.SurrogateInterface)
	}

	s.SouinBaseHandler = bh
	dc := s.SouinBaseHandler.Configuration.GetDefaultCache()
	if dc.GetDistributed() {
		for _, currentStorer := range s.SouinBaseHandler.Storers {
			if eo, ok := currentStorer.(*storage.EmbeddedOlric); ok {
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
						eo, err := storage.EmbeddedOlricConnectionFactory(&s.Configuration)
						if eo != nil {
							return eo.(*storage.EmbeddedOlric), err
						}
						return nil, err
					})

					if l && e == nil {
						s.SouinBaseHandler.Storers = append(s.SouinBaseHandler.Storers, v.(storage.Storer))
					}
				} else {
					s.logger.Sugar().Debug("Store the olric instance.")
					_, _ = up.LoadOrStore(key, s.SouinBaseHandler.SurrogateKeyStorer)
				}
			}
		}
	}

	if len(app.Storers) == 0 {
		app.Storers = s.SouinBaseHandler.Storers
	}

	if app.SurrogateStorage == (surrogates_providers.SurrogateInterface)(nil) {
		app.SurrogateStorage = s.SouinBaseHandler.SurrogateKeyStorer
	} else {
		s.SouinBaseHandler.SurrogateKeyStorer = app.SurrogateStorage
	}

	return nil
}

func parseCaddyfileGlobalOption(h *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
	souinApp := new(SouinApp)
	cfg := Configuration{
		DefaultCache: DefaultCache{
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

	err := parseConfiguration(&cfg, h, true)

	souinApp.DefaultCache = cfg.DefaultCache
	souinApp.API = cfg.API
	souinApp.CacheKeys = cfg.CacheKeys
	souinApp.LogLevel = cfg.LogLevel

	return httpcaddyfile.App{
		Name:  moduleName,
		Value: caddyconfig.JSON(souinApp, nil),
	}, err
}
func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var s SouinCaddyMiddleware
	return &s, s.UnmarshalCaddyfile(h.Dispenser)
}
func (s *SouinCaddyMiddleware) UnmarshalCaddyfile(h *caddyfile.Dispenser) error {
	dc := DefaultCache{
		AllowedHTTPVerbs: make([]string, 0),
	}
	s.Configuration = Configuration{
		DefaultCache: dc,
	}
	return parseConfiguration(&s.Configuration, h, false)
}

// Interface guards
var (
	_ caddy.CleanerUpper          = (*SouinCaddyMiddleware)(nil)
	_ caddy.Provisioner           = (*SouinCaddyMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyMiddleware)(nil)
	_ caddyfile.Unmarshaler       = (*SouinCaddyMiddleware)(nil)
)
