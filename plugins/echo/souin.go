package souin

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/labstack/echo/v4"
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
			AllowedHTTPVerbs: []string{http.MethodGet},
			Regex: configurationtypes.Regex{
				Exclude: "/excluded",
			},
			Nuts: configurationtypes.CacheProvider{
				Path: "./tmp",
			},
			TTL: configurationtypes.Duration{
				Duration: 5 * time.Second,
			},
			Stale: configurationtypes.Duration{
				Duration: 5 * time.Second,
			},
		},
		LogLevel: "debug",
	}
)

// SouinEchoPlugin declaration.
type SouinEchoMiddleware struct {
	*middleware.SouinBaseHandler
}

func NewMiddleware(c middleware.BaseConfiguration) *SouinEchoMiddleware {
	return &SouinEchoMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
}

func (s *SouinEchoMiddleware) Process(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		rw := c.Response().Writer

		return s.SouinBaseHandler.ServeHTTP(rw, req, func(customWriter http.ResponseWriter, _ *http.Request) error {
			c.Response().Writer = customWriter
			return next(c)
		})
	}
}
