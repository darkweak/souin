package httpcache

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
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
	// Logger level, fallback on caddy's one when not redefined.
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
		rfc.MissCache(rw.Header().Set, req, "CANNOT-HANDLE")
		return next.ServeHTTP(rw, r)
	}

	req = s.Retriever.GetContext().SetContext(req)
	customWriter := &plugins.CustomWriter{
		Response: &http.Response{},
		Buf:      s.bufPool.Get().(*bytes.Buffer),
		Rw:       rw,
		Req:      req,
	}
	getterCtx := getterContext{customWriter, req, next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	r.Body = req.Body
	if plugins.HasMutation(req, rw) {
		return next.ServeHTTP(rw, r)
	}
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	combo := ctx.Value(getterContextCtxKey).(getterContext)

	return plugins.DefaultSouinPluginCallback(customWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
		var e error
		if e = combo.next.ServeHTTP(customWriter, r); e != nil {
			rfc.MissCache(customWriter.Header().Set, req, "SERVE-HTTP-ERROR")
			return e
		}

		combo.req.Response = customWriter.Response
		if combo.req.Response.StatusCode == 0 {
			combo.req.Response.StatusCode = 200
		}
		combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req)

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
		Distributed:         s.Olric.URL != "" || s.Olric.Path != "" || s.Olric.Configuration != nil || s.Etcd.Configuration != nil || s.Redis.URL != "" || s.Redis.Configuration != nil,
		Headers:             s.Headers,
		Olric:               s.Olric,
		Etcd:                s.Etcd,
		Redis:               s.Redis,
		Timeout:             s.Timeout,
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
			Timeout:             app.Timeout,
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

	if s.Configuration.LogLevel == "" {
		s.Configuration.LogLevel = app.LogLevel
	}
	if dc.TTL.Duration == 0 {
		s.Configuration.DefaultCache.TTL = appDc.TTL
	}
	if dc.Stale.Duration == 0 {
		s.Configuration.DefaultCache.Stale = appDc.Stale
	}
	if dc.Timeout.Backend.Duration == 0 {
		s.Configuration.DefaultCache.Timeout.Backend = appDc.Timeout.Backend
	}
	if dc.Timeout.Cache.Duration == 0 {
		s.Configuration.DefaultCache.Timeout.Cache = appDc.Timeout.Cache
	}
	if !dc.Key.DisableBody && !dc.Key.DisableHost && !dc.Key.DisableMethod && !dc.Key.Hide {
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
func (s *SouinCaddyPlugin) Provision(ctx caddy.Context) error {
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
		_ = s.Retriever.GetTransport().GetCoalescingLayerStorage().Destruct()
		s.Retriever.GetTransport().(*rfc.VaryTransport).CoalescingLayerStorage = v.(*types.CoalescingLayerStorage)
	}

	if app.Provider == nil {
		app.Provider = s.Retriever.GetProvider()
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

	parseConfiguration(cfg, h, true)

	souinApp.DefaultCache = cfg.DefaultCache
	souinApp.API = cfg.API
	souinApp.CacheKeys = cfg.CfgCacheKeys
	souinApp.LogLevel = cfg.LogLevel

	return httpcaddyfile.App{
		Name:  moduleName,
		Value: caddyconfig.JSON(souinApp, nil),
	}, nil
}
func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var s SouinCaddyPlugin
	return &s, s.UnmarshalCaddyfile(h.Dispenser)
}
func (s *SouinCaddyPlugin) UnmarshalCaddyfile(h *caddyfile.Dispenser) error {
	dc := DefaultCache{
		AllowedHTTPVerbs: make([]string, 0),
	}
	s.Configuration = &Configuration{
		DefaultCache: &dc,
	}

	parseConfiguration(s.Configuration, h, false)

	return nil
}

// Interface guards
var (
	_ caddy.CleanerUpper          = (*SouinCaddyPlugin)(nil)
	_ caddy.Provisioner           = (*SouinCaddyPlugin)(nil)
	_ caddyhttp.MiddlewareHandler = (*SouinCaddyPlugin)(nil)
	_ caddyfile.Unmarshaler       = (*SouinCaddyPlugin)(nil)
)
