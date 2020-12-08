package service

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
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

func getKeyFromResponse(resp *http.Response, u configurationtypes.URL) string {
	headers := ""
	if u.Headers != nil && len(u.Headers) > 0 {
		for _, h := range u.Headers {
			headers += strings.ReplaceAll(resp.Request.Header.Get(h), " ", "")
		}
	}
	return resp.Request.Host + resp.Request.URL.Path + headers
}

func RewriteResponse(resp *http.Response) []byte {
	b := bytes.Replace(responseBodyExtractor(resp), []byte("server"), []byte("schmerver"), -1)
	body := ioutil.NopCloser(bytes.NewReader(b))
	resp.Body = body
	resp.ContentLength = int64(len(b))
	resp.Header.Set("Content-Length", strconv.Itoa(len(b)))

	return b
}

// RequestReverseProxy returns response from one of providers or the proxy response
func RequestReverseProxy(req *http.Request, r types.RetrieverResponsePropertiesInterface) types.ReverseResponse {
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
		Proxy:    proxy,
		Request:  req,
	}
}
