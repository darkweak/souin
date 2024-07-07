package dotweb

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/storages"
	"github.com/devfeel/dotweb"
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

// SouinDotwebMiddleware declaration.
type SouinDotwebMiddleware struct {
	dotweb.BaseMiddleware
	*middleware.SouinBaseHandler
}

func NewHTTPCache(c middleware.BaseConfiguration) *SouinDotwebMiddleware {
	storages.InitFromConfiguration(&c)
	return &SouinDotwebMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
}

func (s *SouinDotwebMiddleware) Handle(c dotweb.Context) error {
	rq := c.Request().Request
	rw := c.Response().Writer()

	return s.SouinBaseHandler.ServeHTTP(rw, rq, func(w http.ResponseWriter, r *http.Request) error {
		c.Request().Request = r
		c.Response().SetWriter(w)
		_ = s.Next(c)

		return nil
	})
}
