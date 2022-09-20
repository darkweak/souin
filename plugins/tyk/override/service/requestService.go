package service

import (
	"net/http"
	"net/http/httputil"

	"github.com/darkweak/souin/cache/types"
	souintypes "github.com/darkweak/souin/plugins/souin/types"
)

// RequestReverseProxy returns response from one of providers or the proxy response
func RequestReverseProxy(req *http.Request, r souintypes.SouinRetrieverResponseProperties) types.ReverseResponse {
	url := r.GetReverseProxyURL()
	req.URL.Host = req.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = r.GetTransport()

	return types.ReverseResponse{
		Proxy:   proxy,
		Request: req,
	}
}
