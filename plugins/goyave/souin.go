package goyave

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins"
	"goyave.dev/goyave/v4"
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

// SouinGoyaveMiddleware declaration.
type SouinGoyaveMiddleware struct {
	*middleware.SouinBaseHandler
}

func NewHTTPCache(c plugins.BaseConfiguration) *SouinGoyaveMiddleware {
	return &SouinGoyaveMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
}

func (s *SouinGoyaveMiddleware) Handle(next goyave.Handler) goyave.Handler {
	return func(res *goyave.Response, rq *goyave.Request) {
		baseWriter := res.Writer()
		defer res.SetWriter(baseWriter)
		s.ServeHTTP(newBaseResponseWriter(res), rq.Request(), func(w http.ResponseWriter, r *http.Request) error {
			if writer, ok := w.(*middleware.CustomWriter); ok {
				writer.Rw = newBaseWriter(baseWriter, w.Header())
				res.SetWriter(writer)
			}
			next(res, rq)
			res.Header().Set("this", "tht")

			return nil
		})
	}
}
