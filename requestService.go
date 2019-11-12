package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"bytes"
	"io/ioutil"
	"encoding/json"
)

type RequestResponse struct {
	Body    []byte `json:"body"`
	Headers map[string][]string `json:"headers"`
}

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

func hasNotAllowedHeaders(r *http.Response) bool {
	return "" != r.Header.Get("Authorization") ||
		"no-cache" == r.Header.Get("Cache-Control")
}

func getKeyFromResponse(resp *http.Response) string {
	return resp.Request.Host + resp.Request.URL.Path
}

func rewriteBody(resp *http.Response) (err error) {
	b := bytes.Replace(commonLoadingRequest(resp), []byte("server"), []byte("schmerver"), -1)
	body := ioutil.NopCloser(bytes.NewReader(b))

	if http.MethodGet == resp.Request.Method && pathnameNotInRegex(resp.Request.Host + resp.Request.URL.Path) && !hasNotAllowedHeaders(resp) {
		r, _ := json.Marshal(RequestResponse{b, resp.Header})

		setRequestInCache(getKeyFromResponse(resp), r)
	}
	resp.Body = body
	return nil
}

func requestReverseProxy(req *http.Request, url *url.URL) ReverseResponse {
	req.URL.Host = req.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = rewriteBody

	return ReverseResponse{
		"",
		proxy,
		req,
	}
}
