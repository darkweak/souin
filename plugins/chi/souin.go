package chi

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
			Stale: configurationtypes.Duration{
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
			Stale: configurationtypes.Duration{
				Duration: 10 * time.Second,
			},
		},
		LogLevel: "debug",
	}
)

// SouinChiMiddleware declaration.
type (
	key                string
	SouinChiMiddleware struct {
		plugins.SouinBasePlugin
		Configuration *plugins.BaseConfiguration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next http.HandlerFunc
		rw   http.ResponseWriter
		req  *http.Request
	}
)

func NewHTTPCache(c plugins.BaseConfiguration) *SouinChiMiddleware {
	s := SouinChiMiddleware{}
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

func (s *SouinChiMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		req := s.Retriever.GetContext().SetBaseContext(r)
		if b, handler := s.HandleInternally(req); b {
			handler(rw, req)

			return
		}

		if !plugins.CanHandle(req, s.Retriever) {
			rfc.MissCache(rw.Header().Set, req, "CANNOT-HANDLE")
			next.ServeHTTP(rw, r)

			return
		}

		customWriter := &plugins.CustomWriter{
			Response: &http.Response{},
			Buf:      s.bufPool.Get().(*bytes.Buffer),
			Rw:       rw,
			Req:      req,
		}
		req = s.Retriever.GetContext().SetContext(req)
		getterCtx := getterContext{next.ServeHTTP, customWriter, req}
		ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
		req = req.WithContext(ctx)
		if plugins.HasMutation(req, rw) {
			next.ServeHTTP(rw, r)

			return
		}
		req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		combo := ctx.Value(getterContextCtxKey).(getterContext)

		_ = plugins.DefaultSouinPluginCallback(customWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
			var e error
			combo.next.ServeHTTP(customWriter, r)

			combo.req.Response = customWriter.Response
			if combo.req.Response.StatusCode == 0 {
				combo.req.Response.StatusCode = 200
			}
			combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req)

			return e
		})
	})
}
