package gin

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
	"github.com/gin-gonic/gin"
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

// SouinGinPlugin declaration.
type (
	key            string
	SouinGinPlugin struct {
		plugins.SouinBasePlugin
		Configuration *Configuration
		bufPool       *sync.Pool
	}
	getterContext struct {
		c   *gin.Context
		rw  http.ResponseWriter
		req *http.Request
	}
)

func New(c Configuration) *SouinGinPlugin {
	s := SouinGinPlugin{}
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

func (s *SouinGinPlugin) Process() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := s.Retriever.GetContext().Method.SetContext(c.Request)
		if !plugins.CanHandle(req, s.Retriever) {
			c.Writer.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
			c.Next()
			return
		}

		if b, handler := s.HandleInternally(req); b {
			handler(c.Writer, req)
			return
		}

		if c.Writer.Status() == http.StatusNotFound {
			c.Writer.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
			c.Next()
			return
		}

		customWriter := &ginWriterDecorator{
			CustomWriter: &plugins.CustomWriter{
				Response: &http.Response{},
				Buf:      s.bufPool.Get().(*bytes.Buffer),
				Rw:       c.Writer,
			},
		}
		c.Writer = customWriter
		req = s.Retriever.GetContext().SetContext(req)
		getterCtx := getterContext{c, customWriter, req}
		ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
		req = req.WithContext(ctx)
		req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		combo := ctx.Value(getterContextCtxKey).(getterContext)

		plugins.DefaultSouinPluginCallback(customWriter.CustomWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
			var e error
			combo.c.Next()

			combo.req.Response = customWriter.Response
			if combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req); e != nil {
				return e
			}

			_, _ = customWriter.Send()
			return e
		})
	}
}
