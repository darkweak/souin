package goyave

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
	"goyave.dev/goyave/v4"
)

const (
	getterContextCtxKey key = "getter_context"
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

// SouinGoyaveMiddleware declaration.
type (
	key                   string
	SouinGoyaveMiddleware struct {
		plugins.SouinBasePlugin
		Configuration *Configuration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next goyave.Handler
		rw   *goyaveWriterDecorator
		req  *http.Request
	}
)

func NewHTTPCache(c Configuration) *SouinGoyaveMiddleware {
	s := SouinGoyaveMiddleware{}
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

func (s *SouinGoyaveMiddleware) Handle(next goyave.Handler) goyave.Handler {
	return func(response *goyave.Response, request *goyave.Request) {
		req := s.Retriever.GetContext().Method.SetContext(request.Request())
		if b, handler := s.HandleInternally(req); b || response.Hijacked() {
			handler(response, req)

			return
		}

		if !plugins.CanHandle(req, s.Retriever) {
			response.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
			next(response, request)

			return
		}

		req = s.Retriever.GetContext().SetContext(req)
		customWriter := &goyaveWriterDecorator{
			Response:       &http.Response{},
			buf:            s.bufPool.Get().(*bytes.Buffer),
			writer:         response.Writer(),
			request:        req,
			goyaveResponse: response,
		}
		getterCtx := getterContext{next, customWriter, req}
		ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
		req = req.WithContext(ctx)
		if plugins.HasMutation(req, response) {
			next(response, request)

			return
		}
		req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		combo := ctx.Value(getterContextCtxKey).(getterContext)
		response.SetWriter(combo.rw)

		_ = plugins.DefaultSouinPluginCallback(combo.rw, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
			combo.rw.updateCache = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually
			next(response, request)

			return nil
		})
	}
}
