package coalescing

import (
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/rfc"
	"net/http"
	"strings"
)

// Drop will remove one key on the coalescing cache
func (r *RequestCoalescing) Drop(dropped string) {
	delete(r.channels, dropped)
}

// Reset will drop all associated channels then recreate the RequestCoalescing instance
// TLDR flush RequestCoalescing channels
func (r *RequestCoalescing) Reset() *RequestCoalescing {
	for k := range r.channels {
		r.Drop(k)
	}
	return Initialize()
}

// Resolve will serve http response from Proxy associated
func (r *RequestCoalescing) Resolve(rr types.ReverseResponse, req *http.Request, rw http.ResponseWriter) {
	rr.Proxy.ServeHTTP(rw, req)
	r.ResolveAll(rr, req, rw)
}

// ResolveAll will serve temporised http response from Proxy associated
func (r *RequestCoalescing) ResolveAll(rr types.ReverseResponse, req *http.Request, rw http.ResponseWriter) {
	key := rfc.GetCacheKey(req)

	close(r.channels[key])
	for v := range r.channels[key] {
		rr.Proxy.ServeHTTP(v.Rw, v.Rq)
	}
	r.Drop(key)
}

// Temporise will run one call to proxy then use the response for other requests that couldn't reach cached response
func (r *RequestCoalescing) Temporise(req *http.Request, rw http.ResponseWriter, retriever types.RetrieverResponsePropertiesInterface) {
	key := rfc.GetCacheKey(req)
	if nil == r.channels[key] {
		r.channels[key] = make(chan RequestCoalescingChannelItem)
		rr := service.RequestReverseProxy(req, retriever)
		r.Resolve(rr, req, rw)
	} else {
		if _, ok := <-r.channels[key]; ok {
			r.channels[key] <-RequestCoalescingChannelItem{req, rw}
		}
	}
}

// Initialize will return RequestCoalescing instance
func Initialize() *RequestCoalescing {
	requestCoalescing := make(map[string]chan RequestCoalescingChannelItem)
	return &RequestCoalescing{
		channels: requestCoalescing,
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
