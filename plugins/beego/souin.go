package beego

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"

	"github.com/beego/beego/v2/server/web"
	beegoCtx "github.com/beego/beego/v2/server/web/context"
)

const (
	getterContextCtxKey key    = "getter_context"
	name                string = "httpcache"
)

var (
	DefaultConfiguration = Configuration{
		DefaultCache: &configurationtypes.DefaultCache{
			TTL: configurationtypes.Duration{
				Duration: 10 * time.Second,
			},
		},
		LogLevel: "info",
	}
	DevDefaultConfiguration = Configuration{
		API: configurationtypes.API{
			BasePath: "/souin-api",
			Prometheus: configurationtypes.APIEndpoint{
				Enable: true,
			},
			Souin: configurationtypes.APIEndpoint{
				Enable: true,
			},
		},
		DefaultCache: &configurationtypes.DefaultCache{
			Regex: configurationtypes.Regex{
				Exclude: "/excluded",
			},
			TTL: configurationtypes.Duration{
				Duration: 5 * time.Second,
			},
		},
		LogLevel: "debug",
	}
)

// SouinBeegoMiddleware declaration.
type (
	key                  string
	SouinBeegoMiddleware struct {
		plugins.SouinBasePlugin
		Configuration *Configuration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next web.FilterFunc
		rw   http.ResponseWriter
		req  *http.Request
	}
)

func NewHTTPCache(c Configuration) *SouinBeegoMiddleware {
	s := SouinBeegoMiddleware{}
	s.Configuration = &c
	s.bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(&c)
	s.RequestCoalescing = coalescing.Initialize()
	s.MapHandler = api.GenerateHandlerMap(s.Configuration, s.Retriever.GetTransport())

	return &s
}

func configurationPropertyMapper(c map[string]interface{}) Configuration {
	configuration := Configuration{}

	for k, v := range c {
		switch k {
		case "allowed_http_verbs":
			allowed := configuration.DefaultCache.AllowedHTTPVerbs
			allowed = append(allowed, v.([]string)...)
			configuration.DefaultCache.AllowedHTTPVerbs = allowed
		case "api":
			var a configurationtypes.API
			var prometheusConfiguration, souinConfiguration, securityConfiguration map[string]interface{}
			apiConfiguration := v.(map[string]interface{})
			for apiK, apiV := range apiConfiguration {
				switch apiK {
				case "prometheus":
					prometheusConfiguration = make(map[string]interface{})
					if apiV != nil && len(apiV.(string)) != 0 {
						prometheusConfiguration = apiV.(map[string]interface{})
					}
				case "souin":
					souinConfiguration = apiV.(map[string]interface{})
				case "security":
					securityConfiguration = make(map[string]interface{})
					if apiV != nil && len(apiV.(string)) != 0 {
						securityConfiguration = apiV.(map[string]interface{})
					}
				}
			}
			if prometheusConfiguration != nil {
				a.Prometheus = configurationtypes.APIEndpoint{}
				a.Prometheus.Enable = true
				if prometheusConfiguration["basepath"] != nil {
					a.Prometheus.BasePath = prometheusConfiguration["basepath"].(string)
				}
				if prometheusConfiguration["security"] != nil {
					a.Prometheus.Security = prometheusConfiguration["security"].(bool)
				}
			}
			if souinConfiguration != nil {
				a.Souin = configurationtypes.APIEndpoint{}
				a.Souin.Enable = true
				if souinConfiguration["basepath"] != nil {
					a.Souin.BasePath = souinConfiguration["basepath"].(string)
				}
				if souinConfiguration["security"] != nil {
					a.Souin.Security = souinConfiguration["security"].(bool)
				}
			}
			if securityConfiguration != nil {
				a.Security = configurationtypes.SecurityAPI{}
				a.Security.Enable = true
				if securityConfiguration["basepath"] != nil {
					a.Security.BasePath = securityConfiguration["basepath"].(string)
				}
				if securityConfiguration["users"] != nil {
					a.Security.Users = securityConfiguration["users"].([]configurationtypes.User)
				}
			}
			configuration.API = a
		case "default_cache":
			dc := configurationtypes.DefaultCache{
				Distributed: false,
				Headers:     []string{},
				Olric: configurationtypes.CacheProvider{
					URL:           "",
					Path:          "",
					Configuration: nil,
				},
				Regex:               configurationtypes.Regex{},
				TTL:                 configurationtypes.Duration{},
				DefaultCacheControl: "",
			}
			defaultCache := v.(map[string]interface{})
			for defaultCacheK, defaultCacheV := range defaultCache {
				switch defaultCacheK {
				case "headers":
					dc.Headers = defaultCacheV.([]string)
				case "regex":
					exclude := defaultCacheV.(map[string]interface{})["exclude"].(string)
					if exclude != "" {
						dc.Regex = configurationtypes.Regex{Exclude: exclude}
					}
				case "ttl":
					ttl, err := time.ParseDuration(defaultCacheV.(string))
					if err == nil {
						dc.TTL = configurationtypes.Duration{Duration: ttl}
					}
				case "stale":
					stale, err := time.ParseDuration(defaultCacheV.(string))
					if err == nil {
						dc.Stale = configurationtypes.Duration{Duration: stale}
					}
				case "default_cache_control":
					dc.DefaultCacheControl = defaultCacheV.(string)
				}
			}
			configuration.DefaultCache = &dc
		case "log_level":
			configuration.LogLevel = v.(string)
		case "urls":
			u := make(map[string]configurationtypes.URL)
			urls := v.(map[string]interface{})

			for urlK, urlV := range urls {
				currentURL := configurationtypes.URL{
					TTL:     configurationtypes.Duration{},
					Headers: nil,
				}
				currentValue := urlV.(map[string]interface{})
				currentURL.Headers = currentValue["headers"].([]string)
				d := currentValue["ttl"].(string)
				ttl, err := time.ParseDuration(d)
				if err == nil {
					currentURL.TTL = configurationtypes.Duration{Duration: ttl}
				}
				if _, exists := currentValue["default_cache_control"]; exists {
					currentURL.DefaultCacheControl = currentValue["default_cache_control"].(string)
				}
				u[urlK] = currentURL
			}
			configuration.URLs = u
		case "ykeys":
			ykeys := make(map[string]configurationtypes.SurrogateKeys)
			d, _ := json.Marshal(v)
			_ = json.Unmarshal(d, &ykeys)
			configuration.Ykeys = ykeys
		}
	}

	return configuration
}

func NewHTTPCacheFilter() web.FilterChain {
	currentConfig := DefaultConfiguration

	if v, e := web.AppConfig.DIY("httpcache"); v != nil && e == nil {
		currentConfig = configurationPropertyMapper(v.(map[string]interface{}))
	}

	httpcache := NewHTTPCache(currentConfig)
	return httpcache.chainHandleFilter
}

func (s *SouinBeegoMiddleware) chainHandleFilter(next web.HandleFunc) web.HandleFunc {
	return func(c *beegoCtx.Context) {
		rw := c.ResponseWriter
		r := c.Request
		req := s.Retriever.GetContext().Method.SetContext(r)
		if !plugins.CanHandle(req, s.Retriever) {
			c.Output.Header("Cache-Status", "Souin; fwd=uri-miss")
			next(c)

			return
		}

		if b, handler := s.HandleInternally(req); b {
			handler(rw, req)

			return
		}

		customCtx := &beegoCtx.Context{
			Input:   c.Input,
			Output:  c.Output,
			Request: c.Request,
			ResponseWriter: &beegoCtx.Response{
				ResponseWriter: nil,
			},
		}

		customWriter := &plugins.CustomWriter{
			Response: &http.Response{},
			Buf:      s.bufPool.Get().(*bytes.Buffer),
			Rw: &beegoWriterDecorator{
				ctx:      customCtx,
				buf:      s.bufPool.Get().(*bytes.Buffer),
				Response: &http.Response{},
			},
		}
		customCtx.ResponseWriter.ResponseWriter = customWriter.Rw
		req = s.Retriever.GetContext().SetContext(req)
		getterCtx := getterContext{next, customWriter, req}
		ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
		req = req.WithContext(ctx)
		if plugins.HasMutation(req, rw) {
			next(c)

			return
		}
		req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		combo := ctx.Value(getterContextCtxKey).(getterContext)

		_ = plugins.DefaultSouinPluginCallback(customWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
			var e error
			combo.next(customCtx)

			combo.req.Response = customWriter.Rw.(*beegoWriterDecorator).Response
			combo.req.Response.StatusCode = 200
			if combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req); e != nil {
				return e
			}

			customWriter.Rw.(*beegoWriterDecorator).Send()
			return e
		})
	}
}
