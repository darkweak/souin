package beego

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"

	"github.com/cloudwego/hertz/pkg/app"
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

// SouinHertzMiddleware declaration.
type SouinHertzMiddleware struct {
	*middleware.SouinBaseHandler
}

func NewHTTPCache(c middleware.BaseConfiguration) app.HandlerFunc {
	httpcache := &SouinHertzMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}

	return httpcache.handle
}

func (s *SouinHertzMiddleware) handle(ctx context.Context, c *app.RequestContext) {
	u, _ := url.ParseRequestURI(string(c.Request.URI().FullURI()))
	rq := &http.Request{
		RequestURI: string(c.Request.URI().FullURI()),
		Method:     string(c.Request.Method()),
		URL:        u,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewBuffer(c.Request.Body())),
		Host:       string(c.Request.URI().Host()),
	}

	for _, cookie := range c.Request.Header.Cookies() {
		rq.AddCookie(&http.Cookie{
			Name:       string(cookie.Key()),
			Value:      string(cookie.Value()),
			Path:       "",
			Domain:     "",
			Expires:    cookie.Expire(),
			RawExpires: cookie.Expire().Local().String(),
			MaxAge:     cookie.MaxAge(),
			Secure:     cookie.Secure(),
			HttpOnly:   cookie.HTTPOnly(),
			SameSite:   http.SameSite(cookie.SameSite()),
			Raw:        string(cookie.Cookie()),
		})
	}
	c.Request.Header.VisitAll(func(key, value []byte) {
		rq.Header.Set(string(key), string(value))
	})

	rw := newWriter(&c.Response)

	_ = s.SouinBaseHandler.ServeHTTP(rw, rq, func(w http.ResponseWriter, r *http.Request) error {
		c.Response.HijackWriter(newHijackWriter(w))
		c.Next(ctx)

		return nil
	})

	if c.Response.GetHijackWriter() != nil {
		c.Response.BodyBuffer().Set(rw.buf)
		c.Response.HijackWriter(nil)
	}
}
