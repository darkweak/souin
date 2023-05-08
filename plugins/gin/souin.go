package gin

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/gin-gonic/gin"
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

// SouinGinMiddleware declaration.
type SouinGinMiddleware struct {
	*middleware.SouinBaseHandler
}

func New(c middleware.BaseConfiguration) *SouinGinMiddleware {
	return &SouinGinMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
}

func (s *SouinGinMiddleware) Process() gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = s.SouinBaseHandler.ServeHTTP(c.Writer, c.Request, func(cw http.ResponseWriter, _ *http.Request) error {
			if writer, ok := cw.(gin.ResponseWriter); ok {
				c.Writer = writer
			} else if writer, ok := cw.(*middleware.CustomWriter); ok {
				c.Writer = &ginWriterDecorator{
					CustomWriter: writer,
				}
			}
			c.Next()

			return nil
		})
	}
}
