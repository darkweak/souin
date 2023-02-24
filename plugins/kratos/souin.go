package kratos

import (
	"net/http"

	"github.com/darkweak/souin/pkg/middleware"
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
	s := &httpcacheKratosMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}
	return s.handle
}

func (s *httpcacheKratosMiddleware) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		s.SouinBaseHandler.ServeHTTP(rw, req, func(w http.ResponseWriter, r *http.Request) error {
			next.ServeHTTP(w, r)

			return nil
		})
	})
}
