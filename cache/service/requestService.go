package service

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	p "github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
)

func commonLoadingRequest(resp *http.Response) []byte {
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

func hasCacheEnabled(r *http.Response) bool {
	return "no-cache" != r.Header.Get("Cache-Control")
}

func getKeyFromResponse(resp *http.Response, u configuration.URL) string {
	headers := ""
	if u.Headers != nil && len(u.Headers) > 0 {
		for _, h := range u.Headers {
			headers += strings.ReplaceAll(resp.Request.Header.Get(h), " ", "")
		}
	}
	return resp.Request.Host + resp.Request.URL.Path + headers
}

func rewriteBody(resp *http.Response, providers map[string]p.AbstractProviderInterface, configuration configuration.Configuration, matchedURL configuration.URL) (err error) {
	b := bytes.Replace(commonLoadingRequest(resp), []byte("server"), []byte("schmerver"), -1)
	body := ioutil.NopCloser(bytes.NewReader(b))
	resp.Body = body
	resp.ContentLength = int64(len(b))
	resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
	pathname := resp.Request.Host+resp.Request.URL.Path

	if p.PathnameNotInExcludeRegex(pathname, configuration) && hasCacheEnabled(resp)  && nil == resp.Request.Context().Err() {
		key := getKeyFromResponse(resp, matchedURL)
		if http.MethodGet == resp.Request.Method && len(b) > 0 {
			r, _ := json.Marshal(types.RequestResponse{Body: b, Headers: resp.Header})
			go func() {
				for _, v := range matchedURL.Providers {
					providers[v].SetRequestInCache(key, r, matchedURL)
				}
			}()
		} else {
			for _, v := range matchedURL.Providers {
				providers[v].DeleteRequestInCache(key)
				newKeySplitted := strings.Split(key, "/")
				maxSize := len(newKeySplitted) - 1
				newKey := ""
				for i := 0; i < maxSize; i++ {
					newKey += newKeySplitted[i] + "/"
				}
				for _, v := range matchedURL.Providers {
					providers[v].DeleteRequestInCache(newKey)
				}
			}
		}
	}

	return nil
}

// RequestReverseProxy returns response from one of providers or the proxy response
func RequestReverseProxy(req *http.Request, url *url.URL, providers map[string]p.AbstractProviderInterface, configuration configuration.Configuration, matchedURL configuration.URL) types.ReverseResponse {
	req.URL.Host = req.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = func(response *http.Response) error {
		return rewriteBody(response, providers, configuration, matchedURL)
	}

	return types.ReverseResponse{
		Response: "bad",
		Proxy:    proxy,
		Request:  req,
	}
}
