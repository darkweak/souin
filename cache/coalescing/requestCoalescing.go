package coalescing

import (
	"context"
	"fmt"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/rfc"
	"github.com/go-chi/stampede"
	"net/http"
	"strings"
	"time"
)

// Temporise will run one call to proxy then use the response for other requests that couldn't reach cached response
func (r *RequestCoalescing) Temporise(req *http.Request, rw http.ResponseWriter, nextMiddleware func(http.ResponseWriter, *http.Request) error) {
	_, e := r.Cache.Get(context.Background(), rfc.GetCacheKey(req), func(ctx context.Context) (interface{}, error) {
		return nil, nextMiddleware(rw, req)
	})

	fmt.Println(e)
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
	path := req.Host + req.URL.Path
	regexpURL := retriever.GetRegexpUrls().FindString(path)
	url := configurationtypes.URL{
		TTL:     configurationtypes.Duration{Duration: retriever.GetConfiguration().GetDefaultCache().GetTTL()},
		Headers: retriever.GetConfiguration().GetDefaultCache().GetHeaders(),
	}
	if "" != regexpURL {
		u := retriever.GetConfiguration().GetUrls()[regexpURL]
		if u.TTL.Duration != 0 {
			url.TTL = u.TTL
		}
		if len(u.Headers) != 0 {
			url.Headers = u.Headers
		}
	}
	retriever.GetTransport().SetURL(url)
	retriever.SetMatchedURL(url)

	headers := ""
	if retriever.GetMatchedURL().Headers != nil && len(retriever.GetMatchedURL().Headers) > 0 {
		for _, h := range retriever.GetMatchedURL().Headers {
			headers += strings.ReplaceAll(req.Header.Get(h), " ", "")
		}
	}

	callback(res, req, retriever, rc, nm)
}
