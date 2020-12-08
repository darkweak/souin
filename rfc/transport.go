package rfc

import (
	"fmt"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/pquerna/cachecontrol"
	"net/http"
	"net/http/httputil"
	"time"
)

type VaryTransport types.Transport

func IsVaryCacheable(req *http.Request) bool {
	method := req.Method
	rangeHeader := req.Header.Get("range")
	return (method == http.MethodGet || method == http.MethodHead) && rangeHeader == ""
}

// NewTransport returns a new Transport with the
// provided Cache implementation and MarkCachedResponses set to true
func NewTransport(p types.AbstractProviderInterface) *VaryTransport {
	return &VaryTransport{Provider: p, MarkCachedResponses: true}
}

func (t *VaryTransport) GetProvider() types.AbstractProviderInterface {
	return t.Provider
}

func (t *VaryTransport) SetUrl(url configurationtypes.URL) {
	t.ConfigurationURL = url
}

func (t *VaryTransport) SetCache(key string, resp *http.Response, req *http.Request) {
	r, d, _ := cachecontrol.CachableResponse(req, resp, cachecontrol.Options{})
	respBytes, err := httputil.DumpResponse(resp, true)
	if err == nil && len(r) == 0 {
		t.Provider.Set(key, respBytes, t.ConfigurationURL, time.Duration(0))
	}
}
