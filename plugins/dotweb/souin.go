package dotweb

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
	"github.com/devfeel/dotweb"
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

// SouinDotwebMiddleware declaration.
type (
	key                   string
	SouinDotwebMiddleware struct {
		dotweb.BaseMiddleware
		plugins.SouinBasePlugin
		Configuration *plugins.BaseConfiguration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next func(ctx dotweb.Context) error
		rw   http.ResponseWriter
		req  *http.Request
	}
)

func NewHTTPCache(c plugins.BaseConfiguration) *SouinDotwebMiddleware {
	s := SouinDotwebMiddleware{}
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

func (s *SouinDotwebMiddleware) Handle(c dotweb.Context) error {
	req := s.Retriever.GetContext().Method.SetContext(c.Request().Request)
	rw := c.Response().Writer()
	if b, handler := s.HandleInternally(req); b {
		handler(rw, req)

		return nil
	}

	if !plugins.CanHandle(req, s.Retriever) {
		rw.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
		return s.Next(c)
	}

	customWriter := &plugins.CustomWriter{
		Response: &http.Response{},
		Buf:      s.bufPool.Get().(*bytes.Buffer),
		Rw:       rw,
	}
	req = s.Retriever.GetContext().SetContext(req)
	getterCtx := getterContext{s.Next, customWriter, req}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	if plugins.HasMutation(req, rw) {
		return s.Next(c)
	}
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	combo := ctx.Value(getterContextCtxKey).(getterContext)
	c.Request().Request = req
	c.Response().SetWriter(customWriter)

	return plugins.DefaultSouinPluginCallback(customWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
		var e error
		combo.next(c)

		combo.req.Response = customWriter.Response
		if combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req); e != nil {
			return e
		}

		_, _ = customWriter.Send()
		return e
	})
}
