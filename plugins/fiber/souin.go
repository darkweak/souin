package fiber

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
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

// SouinFiberMiddleware declaration.
type (
	key                  string
	SouinFiberMiddleware struct {
		plugins.SouinBasePlugin
		Configuration *plugins.BaseConfiguration
		bufPool       *sync.Pool
	}
	getterContext struct {
		next func() error
		rw   http.ResponseWriter
		req  *http.Request
	}
)

func convertResponse(stdreq *http.Request, fastresp *fasthttp.Response) *http.Response {
	status := fastresp.Header.StatusCode()
	body := fastresp.Body()

	stdresp := &http.Response{
		Request:    stdreq,
		StatusCode: status,
		Status:     http.StatusText(status),
	}

	fastresp.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)
		if stdresp.Header == nil {
			stdresp.Header = make(http.Header)
		}
		stdresp.Header.Add(sk, sv)
	})

	if fastresp.Header.ContentLength() == -1 {
		stdresp.TransferEncoding = []string{"chunked"}
	}

	if body != nil {
		stdresp.Body = ioutil.NopCloser(bytes.NewReader(body))
	} else {
		stdresp.Body = ioutil.NopCloser(bytes.NewReader(nil))
	}

	return stdresp
}

func NewHTTPCache(c plugins.BaseConfiguration) *SouinFiberMiddleware {
	s := SouinFiberMiddleware{}
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

func (s *SouinFiberMiddleware) Handle(c *fiber.Ctx) error {
	var rq http.Request
	fasthttpadaptor.ConvertRequest(c.Context(), &rq, true)
	req := s.Retriever.GetContext().Method.SetContext(&rq)

	rw := &fiberWriterDecorator{
		CustomWriter: &plugins.CustomWriter{
			Response: &http.Response{},
			Buf:      s.bufPool.Get().(*bytes.Buffer),
			Rw: &fiberWriter{
				Ctx: c,
			},
		},
	}

	if b, handler := s.HandleInternally(req); b {
		handler(rw.Rw, req)

		return nil
	}

	if !plugins.CanHandle(req, s.Retriever) {
		c.Response().Header.Add("Cache-Status", "Souin; fwd=uri-miss")
		return c.Next()
	}

	req = s.Retriever.GetContext().SetContext(req)
	getterCtx := getterContext{c.Next, rw, req}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	if plugins.HasMutation(req, rw) {
		return c.Next()
	}

	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	combo := ctx.Value(getterContextCtxKey).(getterContext)

	e := plugins.DefaultSouinPluginCallback(rw.CustomWriter, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
		var e error
		c.Next()

		combo.req.Response = convertResponse(req, &c.Context().Response)
		if combo.req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req); e != nil {
			return e
		}

		rw.Response = combo.req.Response
		_, _ = rw.Send()
		return e
	})

	rw.Response.Header.Del("X-Souin-Stored-TTL")
	rCtx := rw.Rw.(*fiberWriter).Ctx.Response()
	for hk, hv := range rw.Response.Header {
		rCtx.Header.Set(hk, hv[0])
	}

	return e
}
