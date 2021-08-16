package types

import (
	"net/http"
	"net/http/httputil"
)

// ReverseResponse object contains the response from reverse-proxy
type ReverseResponse struct {
	Response *http.Response
	Proxy    *httputil.ReverseProxy
	Request  *http.Request
}
