package fiber

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var (
	DefaultConfiguration = middleware.BaseConfiguration{
		DefaultCache: &configurationtypes.DefaultCache{
			TTL: configurationtypes.Duration{
				Duration: 10 * time.Second,
			},
		},
		LogLevel: "info",
	}
	DevDefaultConfiguration = middleware.BaseConfiguration{
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
type SouinFiberMiddleware struct {
	*middleware.SouinBaseHandler
}

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
		stdresp.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		stdresp.Body = io.NopCloser(bytes.NewReader(nil))
	}

	return stdresp
}

func NewHTTPCache(c middleware.BaseConfiguration) *SouinFiberMiddleware {
	return &SouinFiberMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
}

func (s *SouinFiberMiddleware) Handle(c *fiber.Ctx) error {
	var rq http.Request
	fasthttpadaptor.ConvertRequest(c.Context(), &rq, true)
	customWriter := newWriter(c.Response())
	err := s.SouinBaseHandler.ServeHTTP(customWriter, &rq, func(w http.ResponseWriter, r *http.Request) error {
		var err error
		if err = c.Next(); err != nil {
			return err
		}

		body := c.Response().Body()
		c.Response().Reset()
		_, err = w.Write(body)

		return err
	})

	// Synchronize the custom writer headers with the Fiber ones
	for hk, hv := range customWriter.h {
		c.Response().Header.Set(hk, hv[0])
	}

	return err
}
