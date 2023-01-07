package beego

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/plugins/souin/agnostic"
	"github.com/darkweak/souin/rfc"

	"github.com/beego/beego/v2/server/web"
	beegoCtx "github.com/beego/beego/v2/server/web/context"
)

const (
	getterContextCtxKey key    = "getter_context"
	name                string = "httpcache"
)

var (
	DefaultConfiguration = plugins.BaseConfiguration{
		DefaultCache: &configurationtypes.DefaultCache{
			TTL: configurationtypes.Duration{
				Duration: 10 * time.Second,
			},
		},
		LogLevel: "info",
	}
	DevDefaultConfiguration = plugins.BaseConfiguration{
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
		Configuration *plugins.BaseConfiguration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next web.FilterFunc
		rw   http.ResponseWriter
		req  *http.Request
	}
)

func NewHTTPCache(c plugins.BaseConfiguration) *SouinBeegoMiddleware {
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

func configurationPropertyMapper(c map[string]interface{}) plugins.BaseConfiguration {
	var configuration plugins.BaseConfiguration
	agnostic.ParseConfiguration(&configuration, c)

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
		req := s.Retriever.GetContext().SetBaseContext(r)
		if !plugins.CanHandle(req, s.Retriever) {
			rfc.MissCache(c.Output.Header, req, "CANNOT-HANDLE")
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

		customWriter := &beegoWriterDecorator{
			ctx:      customCtx,
			buf:      s.bufPool.Get().(*bytes.Buffer),
			Response: &http.Response{},
			CustomWriter: &plugins.CustomWriter{
				Response: &http.Response{},
				Buf:      s.bufPool.Get().(*bytes.Buffer),
				Rw:       rw,
				Req:      req,
			},
		}

		customCtx.ResponseWriter.ResponseWriter = customWriter
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

			combo.req.Response = customWriter.Response
			if combo.req.Response.StatusCode == 0 {
				combo.req.Response.StatusCode = 200
			}
			combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req)

			return e
		})
	}
}
