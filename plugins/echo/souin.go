package souin

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
	"github.com/darkweak/souin/rfc"
	"github.com/labstack/echo/v4"
)

const (
	getterContextCtxKey key = "getter_context"
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
			AllowedHTTPVerbs: []string{http.MethodGet},
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

// SouinEchoPlugin declaration.
type (
	key             string
	SouinEchoPlugin struct {
		plugins.SouinBasePlugin
		Configuration *plugins.BaseConfiguration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next echo.HandlerFunc
		rw   http.ResponseWriter
		req  *http.Request
	}
)

func New(c plugins.BaseConfiguration) *SouinEchoPlugin {
	s := SouinEchoPlugin{}
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

func (s *SouinEchoPlugin) Process(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		rw := c.Response().Writer
		req = s.Retriever.GetContext().SetBaseContext(req)
		if !plugins.CanHandle(req, s.Retriever) {
			rfc.MissCache(rw.Header().Set, req, "CANNOT-HANDLE")
			return next(c)
		}

		if b, handler := s.HandleInternally(req); b {
			handler(rw, req)
			return nil
		}

		customWriter := &plugins.CustomWriter{
			Response: &http.Response{},
			Buf:      s.bufPool.Get().(*bytes.Buffer),
			Rw:       rw,
			Req:      req,
		}

		req = s.Retriever.GetContext().SetContext(req)
		getterCtx := getterContext{next, customWriter, req}
		c.Response().Writer = customWriter
		ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
		req = req.WithContext(ctx)
		if plugins.HasMutation(req, rw) {
			return next(c)
		}
		req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		combo := ctx.Value(getterContextCtxKey).(getterContext)

		return plugins.DefaultSouinPluginCallback(customWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
			var e error
			if e = combo.next(c); e != nil {
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
}
