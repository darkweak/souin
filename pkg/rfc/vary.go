package rfc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/pkg/storage"
)

// GetVariedCacheKey returns the varied cache key for req and resp.
func GetVariedCacheKey(rq *http.Request, headers []string) string {
	if len(headers) == 0 {
		return ""
	}
	for i, v := range headers {
		h := rq.Header.Get(v)
		if strings.Contains(h, ";") || strings.Contains(h, ":") {
			h = url.QueryEscape(h)
		}
		headers[i] = fmt.Sprintf("%s:%s", v, h)
	}

	return storage.VarySeparator + strings.Join(headers, providers.DecodedHeaderSeparator)
}

// headerAllCommaSepValues returns all comma-separated values (each
// with whitespace trimmed) for header name in headers. According to
// Section 4.2 of the HTTP/1.1 spec
// (http://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2),
// values from multiple occurrences of a header should be concatenated, if
// the header's value is a comma-separated list.
func HeaderAllCommaSepValues(headers http.Header) []string {
	var vals []string
	for _, val := range headers[http.CanonicalHeaderKey("Vary")] {
		fields := strings.Split(val, ",")
		for i, f := range fields {
			fields[i] = strings.TrimSpace(f)
		}
		vals = append(vals, fields...)
	}
	return vals
}

// varyMatches will return false unless all of the cached values for the headers listed in Vary
// match the new request
func varyMatches(cachedResp *http.Response, req *http.Request) bool {
	for _, header := range HeaderAllCommaSepValues(cachedResp.Header) {
		header = http.CanonicalHeaderKey(header)
		if header == "" || req.Header.Get(header) == "" {
			return false
		}
	}
	return true
}