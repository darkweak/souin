package beego

import (
	"bytes"
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/plugins/souin/agnostic"

	"github.com/beego/beego/v2/server/web"
	beegoCtx "github.com/beego/beego/v2/server/web/context"
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
	SouinBeegoMiddleware struct {
		*middleware.SouinBaseHandler
	}
)

func NewHTTPCache(c plugins.BaseConfiguration) *SouinBeegoMiddleware {
	return &SouinBeegoMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
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
		rw := c.ResponseWriter.ResponseWriter
		r := c.Request

		customCtx := &beegoCtx.Context{
			Input:   c.Input,
			Output:  c.Output,
			Request: c.Request,
			ResponseWriter: &beegoCtx.Response{
				ResponseWriter: nil,
			},
		}

		s.ServeHTTP(rw, r, func(w http.ResponseWriter, r *http.Request) error {
			customWriter := &CustomWriter{
				ctx: customCtx,
				Buf: bytes.NewBuffer([]byte{}),
				Rw:  w,
			}
			customCtx.ResponseWriter.ResponseWriter = customWriter
			next(customCtx)

			return nil
		})
	}
}

/*
func (s *SouinBeegoMiddleware) chainHandleFilter2(next web.HandleFunc) web.HandleFunc {
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
*/
