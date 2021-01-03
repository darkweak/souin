package coalescing

import (
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/rfc"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strings"
	"time"
)

// Temporise will run one call to proxy then use the response for other requests that couldn't reach cached response
func (r *RequestCoalescing) Temporise(req *http.Request, rw http.ResponseWriter, retriever types.RetrieverResponsePropertiesInterface) {
	ch := r.requestGroup.DoChan(rfc.GetCacheKey(req), func() (interface{}, error) {
		rr := service.RequestReverseProxy(req, retriever)
		rr.Proxy.ServeHTTP(rw, req)

		return nil, nil
	})

	timeout := time.After(60 * time.Second)

	var result singleflight.Result
	select {
	case <-timeout:
		http.Error(rw, "Gateway Timeout", http.StatusGatewayTimeout)
		return
	case result = <-ch:
	}

	if result.Err != nil {
		http.Error(rw, result.Err.Error(), http.StatusInternalServerError)
		return
	}
}

// Initialize will return RequestCoalescing instance
func Initialize() *RequestCoalescing {
	var requestGroup singleflight.Group
	return &RequestCoalescing{
		requestGroup,
	}
}

// ServeResponse serve the response
func ServeResponse(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	callback func(rw http.ResponseWriter, rq *http.Request, r types.RetrieverResponsePropertiesInterface, rc RequestCoalescingInterface),
	rc RequestCoalescingInterface,
) {
	path := req.Host + req.URL.Path
	regexpURL := retriever.GetRegexpUrls().FindString(path)
	if "" != regexpURL {
		url := retriever.GetConfiguration().GetUrls()[regexpURL]
		retriever.SetMatchedURL(url)
	}
	headers := ""
	if retriever.GetMatchedURL().Headers != nil && len(retriever.GetMatchedURL().Headers) > 0 {
		for _, h := range retriever.GetMatchedURL().Headers {
			headers += strings.ReplaceAll(req.Header.Get(h), " ", "")
		}
	}

	callback(res, req, retriever, rc)
}
