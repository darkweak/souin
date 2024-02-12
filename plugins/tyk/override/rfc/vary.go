package rfc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	VarySeparator          = "{-VARY-}"
	DecodedHeaderSeparator = ";"
)

// GetVariedCacheKey returns the varied cache key for req and resp.
func GetVariedCacheKey(rq *http.Request, headers []string) string {
	if len(headers) == 0 {
		return ""
	}
	for i, v := range headers {
		h := strings.TrimSpace(rq.Header.Get(v))
		if strings.Contains(h, ";") || strings.Contains(h, ":") {
			h = url.QueryEscape(h)
		}
		headers[i] = fmt.Sprintf("%s:%s", v, h)
	}

	return VarySeparator + strings.Join(headers, DecodedHeaderSeparator)
}

// VariedHeaderAllCommaSepValues returns all comma-separated values
// or '*' alone when the header contains it.
func VariedHeaderAllCommaSepValues(headers http.Header) ([]string, bool) {
	vals := HeaderAllCommaSepValues(headers, "Vary")
	for _, v := range vals {
		if v == "*" {
			return []string{"*"}, true
		}
	}
	return vals, false
}
