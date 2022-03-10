package coalescing

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/souin/cache/types"
	souin_ctx "github.com/darkweak/souin/context"
	"github.com/go-chi/stampede"
)

// Temporize will run one call to proxy then use the response for other requests that couldn't reach cached response
func (r *RequestCoalescing) Temporize(req *http.Request, rw http.ResponseWriter, nextMiddleware func(http.ResponseWriter, *http.Request) error) {
	_, e := r.Cache.Get(context.Background(), req.Context().Value(souin_ctx.Key).(string), func(ctx context.Context) (interface{}, error) {
		return nil, nextMiddleware(rw, req)
	})

	if e != nil {
		http.Error(rw, "Gateway Timeout", http.StatusGatewayTimeout)
	}
}

// Initialize will return RequestCoalescing instance
func Initialize() *RequestCoalescing {
	return &RequestCoalescing{
		stampede.NewCache(512, 1*time.Second, 2*time.Second),
	}
}

// ServeResponse serve the response
func ServeResponse(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	callback func(http.ResponseWriter, *http.Request, types.RetrieverResponsePropertiesInterface, RequestCoalescingInterface, func(http.ResponseWriter, *http.Request) error),
	rc RequestCoalescingInterface,
	nm func(w http.ResponseWriter, r *http.Request) error,
) {
	headers := ""
	if retriever.GetMatchedURL().Headers != nil && len(retriever.GetMatchedURL().Headers) > 0 {
		for _, h := range retriever.GetMatchedURL().Headers {
			headers += strings.ReplaceAll(req.Header.Get(h), " ", "")
		}
	}

	callback(res, req, retriever, rc, nm)
}
