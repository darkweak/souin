package beego

import (
	"bytes"
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/agnostic"
	"github.com/darkweak/souin/plugins/souin/storages"

	"github.com/beego/beego/v2/server/web"
	beegoCtx "github.com/beego/beego/v2/server/web/context"
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

// SouinBeegoMiddleware declaration.
type SouinBeegoMiddleware struct {
	*middleware.SouinBaseHandler
}

func NewHTTPCache(c middleware.BaseConfiguration) *SouinBeegoMiddleware {
	storages.InitFromConfiguration(&c)
	return &SouinBeegoMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
}

func configurationPropertyMapper(c map[string]interface{}) middleware.BaseConfiguration {
	var configuration middleware.BaseConfiguration
	agnostic.ParseConfiguration(&configuration, c)

	return configuration
}

func NewHTTPCacheFilter() web.FilterChain {
	currentConfig := DefaultConfiguration

	if v, e := web.AppConfig.DIY("httpcache"); v != nil && e == nil {
		currentConfig = configurationPropertyMapper(v.(map[string]interface{}))
	}

	httpcache := NewHTTPCache(currentConfig)
	return httpcache.chainHandleFilter
}

func (s *SouinBeegoMiddleware) chainHandleFilter(next web.HandleFunc) web.HandleFunc {
	return func(c *beegoCtx.Context) {
		rw := c.ResponseWriter.ResponseWriter
		r := c.Request

		customCtx := &beegoCtx.Context{
			Input:   c.Input,
			Output:  c.Output,
			Request: c.Request,
			ResponseWriter: &beegoCtx.Response{
				ResponseWriter: nil,
			},
		}

		_ = s.SouinBaseHandler.ServeHTTP(rw, r, func(w http.ResponseWriter, r *http.Request) error {
			customWriter := &CustomWriter{
				ctx: customCtx,
				Buf: bytes.NewBuffer([]byte{}),
				Rw:  w,
			}
			customCtx.ResponseWriter.ResponseWriter = customWriter
			next(customCtx)

			return nil
		})
	}
}
