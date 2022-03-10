package rfc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/context"
)

// GetVariedCacheKey returns the varied cache key for req and resp.
func GetVariedCacheKey(req *http.Request, headers []string) string {
	for i, v := range headers {
		headers[i] = fmt.Sprintf("%s:%s", v, req.Header.Get(v))
	}
	return req.Context().Value(context.Key).(string) + providers.VarySeparator + strings.Join(headers, ";")
}

func ValidateMaxAgeCachedResponse(req *http.Request, res *http.Response) *http.Response {
	if res == nil {
		return nil
	}

	cc := parseCacheControl(req.Header)
	if maxAge, ok := cc["max-age"]; ok {
		ma, _ := strconv.Atoi(maxAge)
		a, _ := strconv.Atoi(res.Header.Get("Age"))

		if ma < a {
			return nil
		}
	}

	return res
}

func ValidateStaleCachedResponse(req *http.Request, res *http.Response) *http.Response {
	if res == nil {
		return nil
	}

	cc := parseCacheControl(req.Header)
	if maxStale, ok := cc["max-stale"]; ok {
		ms, _ := strconv.Atoi(maxStale)
		a, _ := strconv.Atoi(res.Header.Get("Age"))

		if ms < a {
			return nil
		}
	}

	return res
}

// getFreshness will return one of fresh/stale/transparent based on the cache-control
// values of the request and the response
//
// fresh indicates the response can be returned
// stale indicates that the response needs validating before it is returned
// transparent indicates the response should not be used to fulfill the request
//
// Because this is only a private cache, 'public' and 'private' in cache-control aren't
// significant. Similarly, smax-age isn't used.
func getFreshness(respHeaders, reqHeaders http.Header) (freshness int) {
	respCacheControl := parseCacheControl(respHeaders)
	reqCacheControl := parseCacheControl(reqHeaders)
	if _, ok := reqCacheControl["no-cache"]; ok {
		return transparent
	}
	if _, ok := respCacheControl["no-cache"]; ok {
		return stale
	}
	if _, ok := reqCacheControl["only-if-cached"]; ok {
		return fresh
	}

	date, err := date(respHeaders)
	if err != nil {
		return stale
	}
	currentAge := clock.since(date)

	var lifetime time.Duration
	var zeroDuration time.Duration

	// If a response includes both an Expires header and a max-age directive,
	// the max-age directive overrides the Expires header, even if the Expires header is more restrictive.
	if maxAge, ok := respCacheControl["max-age"]; ok {
		lifetime, err = time.ParseDuration(maxAge + "s")
		if err != nil {
			lifetime = zeroDuration
		}
	} else {
		expiresHeader := respHeaders.Get("Expires")
		if expiresHeader != "" {
			expires, e := time.Parse(time.RFC1123, expiresHeader)
			if e != nil {
				lifetime = zeroDuration
			} else {
				lifetime = expires.Sub(date)
			}
		}
	}

	if maxAge, ok := reqCacheControl["max-age"]; ok {
		// the client is willing to accept a response whose age is no greater than the specified time in seconds
		lifetime, err = time.ParseDuration(maxAge + "s")
		if err != nil {
			lifetime = zeroDuration
		}
	}
	if minfresh, ok := reqCacheControl["min-fresh"]; ok {
		//  the client wants a response that will still be fresh for at least the specified number of seconds.
		minfreshDuration, e := time.ParseDuration(minfresh + "s")
		if e == nil {
			currentAge = currentAge + minfreshDuration
		}
	}

	if maxstale, ok := reqCacheControl["max-stale"]; ok {
		// Indicates that the client is willing to accept a response that has exceeded its expiration time.
		// If max-stale is assigned a value, then the client is willing to accept a response that has exceeded
		// its expiration time by no more than the specified number of seconds.
		// If no value is assigned to max-stale, then the client is willing to accept a stale response of any age.
		//
		// Responses served only because of a max-stale value are supposed to have a Warning header added to them,
		// but that seems like a  hassle, and is it actually useful? If so, then there needs to be a different
		// return-value available here.
		if maxstale == "" {
			return fresh
		}
		maxstaleDuration, e := time.ParseDuration(maxstale + "s")
		if e == nil {
			currentAge = currentAge - maxstaleDuration
		}
	}

	if lifetime > currentAge {
		return fresh
	}

	return stale
}

// Returns true if either the request or the response includes the stale-if-error
// cache control extension: https://tools.ietf.org/html/rfc5861
func canStaleOnError(respHeaders, reqHeaders http.Header) bool {
	respCacheControl := parseCacheControl(respHeaders)
	reqCacheControl := parseCacheControl(reqHeaders)

	var err error
	lifetime := time.Duration(-1)

	for _, cc := range []cacheControl{respCacheControl, reqCacheControl} {
		if staleMaxAge, ok := cc["stale-if-error"]; ok {
			if staleMaxAge != "" {
				lifetime, err = time.ParseDuration(staleMaxAge + "s")
				if err != nil {
					return false
				}
			} else {
				return true
			}
		}
	}

	if lifetime >= 0 {
		date, err := date(respHeaders)
		if err != nil {
			return false
		}
		currentAge := clock.since(date)
		if lifetime > currentAge {
			return true
		}
	}

	return false
}

func getEndToEndHeaders(respHeaders http.Header) []string {
	// These headers are always hop-by-hop
	hopByHopHeaders := map[string]struct{}{
		"Connection":          {},
		"Keep-Alive":          {},
		"Proxy-Authenticate":  {},
		"Proxy-Authorization": {},
		"Te":                  {},
		"Trailers":            {},
		"Transfer-Encoding":   {},
		"Upgrade":             {},
	}

	for _, extra := range strings.Split(respHeaders.Get("connection"), ",") {
		// any header listed in connection, if present, is also considered hop-by-hop
		if strings.Trim(extra, " ") != "" {
			hopByHopHeaders[http.CanonicalHeaderKey(extra)] = struct{}{}
		}
	}
	endToEndHeaders := []string{}
	for respHeader := range respHeaders {
		if _, ok := hopByHopHeaders[respHeader]; !ok {
			endToEndHeaders = append(endToEndHeaders, respHeader)
		}
	}
	return endToEndHeaders
}

func canStore(reqCacheControl cacheControl, respCacheControl cacheControl, status int) (canStore bool) {
	if !cachableStatusCode(status) {
		return false
	}

	for _, t := range []string{"no-cache", "no-store"} {
		if _, ok := respCacheControl[t]; ok {
			return false
		}
		if _, ok := reqCacheControl[t]; ok {
			return false
		}
	}
	return true
}

func cachableStatusCode(statusCode int) bool {
	switch statusCode {
	case 200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501:
		return true
	default:
		return false
	}
}

func newGatewayTimeoutResponse(req *http.Request) *http.Response {
	var b bytes.Buffer
	b.WriteString("HTTP/1.1 504 Gateway Timeout\r\n\r\n")
	resp, _ := http.ReadResponse(bufio.NewReader(&b), req)
	return resp
}

type cacheControl map[string]string

func parseCacheControl(headers http.Header) cacheControl {
	cc := cacheControl{}
	ccHeader := headers.Get("Cache-Control")
	for _, part := range strings.Split(ccHeader, ",") {
		part = strings.Trim(part, " ")
		if part == "" {
			continue
		}
		if strings.ContainsRune(part, '=') {
			keyval := strings.Split(part, "=")
			cc[strings.Trim(keyval[0], " ")] = strings.Trim(keyval[1], ",")
		} else {
			cc[part] = ""
		}
	}
	return cc
}

// ErrNoDateHeader indicates that the HTTP headers contained no Date header.
var ErrNoDateHeader = errors.New("no Date header")

// Date parses and returns the value of the Date header.
func date(respHeaders http.Header) (date time.Time, err error) {
	dateHeader := respHeaders.Get("date")
	if dateHeader == "" {
		err = ErrNoDateHeader
		return
	}

	return time.Parse(time.RFC1123, dateHeader)
}

type realClock struct{}

func (c *realClock) since(d time.Time) time.Duration {
	return time.Since(d)
}

type timer interface {
	since(d time.Time) time.Duration
}

var clock timer = &realClock{}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
// (This function copyright goauth2 authors: https://code.google.com/p/goauth2)
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}

// headerAllCommaSepValues returns all comma-separated values (each
// with whitespace trimmed) for header name in headers. According to
// Section 4.2 of the HTTP/1.1 spec
// (http://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2),
// values from multiple occurrences of a header should be concatenated, if
// the header's value is a comma-separated list.
func headerAllCommaSepValues(headers http.Header) []string {
	var vals []string
	for _, val := range headers[http.CanonicalHeaderKey("vary")] {
		fields := strings.Split(val, ",")
		for i, f := range fields {
			fields[i] = strings.TrimSpace(f)
		}
		vals = append(vals, fields...)
	}
	return vals
}

// cachingReadCloser is a wrapper around ReadCloser R that calls OnEOF
// handler with a full copy of the content read from R when EOF is
// reached.
type cachingReadCloser struct {
	// Underlying ReadCloser.
	R io.ReadCloser
	// OnEOF is called with a copy of the content of R when EOF is reached.
	OnEOF func(io.Reader)

	buf bytes.Buffer // buf stores a copy of the content of R.
}

// Read reads the next len(p) bytes from R or until R is drained. The
// return value n is the number of bytes read. If R has no data to
// return, err is io.EOF and OnEOF is called with a full copy of what
// has been read so far.
func (r *cachingReadCloser) Read(p []byte) (n int, err error) {
	if r.R == nil {
		r.OnEOF(bytes.NewReader(p))
		return 0, io.EOF
	}
	n, err = r.R.Read(p)
	r.buf.Write(p[:n])
	if err == io.EOF {
		r.Close()
		r.OnEOF(bytes.NewReader(r.buf.Bytes()))
	}
	return
}

func (r *cachingReadCloser) Close() error {
	return r.R.Close()
}
