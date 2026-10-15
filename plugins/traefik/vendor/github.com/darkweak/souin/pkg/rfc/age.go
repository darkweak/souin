package rfc

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/cachecontrol/cacheobject"
)

func validateMaxAgeCachedResponse(res *http.Response, maxAge int, addTime int) *http.Response {
	a, _ := strconv.Atoi(res.Header.Get("Age"))

	if maxAge >= 0 && (maxAge+addTime) < a {
		return nil
	}

	return res
}

// ValidateMaxAgeCachedResponse checks a stored response against the `max-age`
// the request asks for. RFC 9111 section 5.2.1.1 makes it a ceiling on the
// age of what the client is willing to accept, so the response's own
// freshness lifetime has no say in it.
func ValidateMaxAgeCachedResponse(co *cacheobject.RequestCacheDirectives, res *http.Response) *http.Response {
	return validateMaxAgeCachedResponse(res, int(co.MaxAge), 0)
}

func ValidateMaxAgeCachedStaleResponse(co *cacheobject.RequestCacheDirectives, resCo *cacheobject.ResponseCacheDirectives, res *http.Response, addTime int) *http.Response {
	if co.MaxStaleSet {
		return res
	}

	if resCo != nil && (resCo.StaleIfError > -1 || co.StaleIfError > 0) {
		if resCo.StaleIfError > -1 {
			if response := validateMaxAgeCachedResponse(res, int(resCo.StaleIfError), addTime); response != nil {
				return response
			}
		}

		if co.StaleIfError > 0 {
			if response := validateMaxAgeCachedResponse(res, int(co.StaleIfError), addTime); response != nil {
				return response
			}
		}
	}

	if co.MaxStale < 0 {
		return nil
	}

	return validateMaxAgeCachedResponse(res, int(co.MaxStale), addTime)
}

// ParseAge parses an `Age` header value. RFC 9111 section 5.1 defines it as a
// single delta-seconds. A sender combining several `Age` values onto one line
// produces a comma separated list, of which only the first element is
// meaningful. Trailing parameters are not part of the delta-seconds either,
// and a value that is not a delta-seconds at all cannot be used and is
// ignored.
func ParseAge(value string) time.Duration {
	if value == "" {
		return 0
	}

	first, _, _ := strings.Cut(value, ",")
	first, _, _ = strings.Cut(first, ";")

	seconds, err := strconv.Atoi(strings.TrimSpace(first))
	if err != nil || seconds < 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

// InitialAge returns the age a response already had when it reached this
// cache, following the corrected_initial_age computation of RFC 9111 section
// 4.2.3.
func InitialAge(headers http.Header, responseTime time.Time) time.Duration {
	age := ParseAge(headers.Get("Age"))

	apparentAge := time.Duration(0)
	if date, dateErr := ParseHTTPDate(headers.Get("Date")); dateErr == nil {
		if apparentAge = responseTime.Sub(date); apparentAge < 0 {
			apparentAge = 0
		}
	}

	if apparentAge > age {
		return apparentAge
	}

	return age
}
