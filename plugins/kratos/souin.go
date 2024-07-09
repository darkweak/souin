package kratos

import (
	"net/http"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/storages"
	kratos_http "github.com/go-kratos/kratos/v2/transport/http"
)

type httpcacheKratosMiddleware struct {
	*middleware.SouinBaseHandler
}

// NewHTTPCacheFilter, allows the user to set up an HTTP cache system,
// RFC-7234 compliant and supports the tag based cache purge,
// distributed and not-distributed storage, key generation tweaking.
// Use it with
// httpcache.NewHTTPCacheFilter(httpcache.ParseConfiguration(config))
func NewHTTPCacheFilter(c middleware.BaseConfiguration) kratos_http.FilterFunc {
	storages.InitFromConfiguration(&c)
	s := &httpcacheKratosMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
	return s.handle
}

func (s *httpcacheKratosMiddleware) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_ = s.SouinBaseHandler.ServeHTTP(rw, req, func(w http.ResponseWriter, r *http.Request) error {
			next.ServeHTTP(w, r)

			return nil
		})
	})
}
