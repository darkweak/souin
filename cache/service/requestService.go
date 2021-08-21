package service

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/darkweak/souin/cache/types"
	souintypes "github.com/darkweak/souin/plugins/souin/types"
)

func responseBodyExtractor(resp *http.Response) []byte {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte("")
	}
	err = resp.Body.Close()
	if err != nil {
		return []byte("")
	}

	return b
}

// RewriteResponse rewrite the response
func RewriteResponse(resp *http.Response) []byte {
	b := responseBodyExtractor(resp)
	lb := len(b)
	body := ioutil.NopCloser(bytes.NewReader(b))
	resp.Body = body
	resp.ContentLength = int64(lb)
	resp.Header.Set("Content-Length", strconv.Itoa(lb))

	return b
}

// RequestReverseProxy returns response from one of providers or the proxy response
func RequestReverseProxy(req *http.Request, r souintypes.SouinRetrieverResponseProperties) types.ReverseResponse {
	url := r.GetReverseProxyURL()
	req.URL.Host = req.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = func(response *http.Response) error {
		_ = RewriteResponse(response)
		return nil
	}
	proxy.Transport = r.GetTransport()

	return types.ReverseResponse{
		Proxy:   proxy,
		Request: req,
	}
}
