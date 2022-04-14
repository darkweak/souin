package stampede

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var stripOutHeaders = []string{
	"Access-Control-Allow-Credentials",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Origin",
	"Access-Control-Expose-Headers",
	"Access-Control-Max-Age",
	"Access-Control-Request-Headers",
	"Access-Control-Request-Method",
}

func Handler(cacheSize int, ttl time.Duration, paths ...string) func(next http.Handler) http.Handler {
	defaultKeyFunc := func(r *http.Request) uint64 {
		// Read the request payload, and then setup buffer for future reader
		var buf []byte
		if r.Body != nil {
			buf, _ = ioutil.ReadAll(r.Body)
			r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		}

		// Prepare cache key based on request URL path and the request data payload.
		key := BytesToHash([]byte(strings.ToLower(r.URL.Path)), buf)
		return key
	}

	return HandlerWithKey(cacheSize, ttl, defaultKeyFunc, paths...)
}

func HandlerWithKey(cacheSize int, ttl time.Duration, keyFunc func(r *http.Request) uint64, paths ...string) func(next http.Handler) http.Handler {
	// mapping of url paths that are cacheable by the stampede handler
	pathMap := map[string]struct{}{}
	for _, path := range paths {
		pathMap[strings.ToLower(path)] = struct{}{}
	}

	// Stampede handler with set ttl for how long content is fresh.
	// Requests sent to this handler will be coalesced and in scenarios
	// where there is a "stampede" or parallel requests for the same
	// method and arguments, there will be just a single handler that
	// executes, and the remaining handlers will use the response from
	// the first request. The content thereafter will be cached for up to
	// ttl time for subsequent requests for further caching.
	h := stampede(cacheSize, ttl, keyFunc)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Cache all paths, as whitelist has not been provided
			if len(pathMap) == 0 {
				h(next).ServeHTTP(w, r)
				return
			}

			// Match specific whitelist of paths
			if _, ok := pathMap[strings.ToLower(r.URL.Path)]; ok {
				// stampede-cache the matching path
				h(next).ServeHTTP(w, r)

			} else {
				// no caching
				next.ServeHTTP(w, r)
			}
		})
	}
}

func stampede(cacheSize int, ttl time.Duration, keyFunc func(r *http.Request) uint64) func(next http.Handler) http.Handler {
	cache := NewCache(cacheSize, ttl, ttl*2)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// cache key for the request
			key := keyFunc(r)

			// mark the request that actually processes the response
			first := false

			// process request (single flight)
			val, err := cache.GetFresh(r.Context(), key, func(ctx context.Context) (interface{}, error) {
				first = true
				buf := bytes.NewBuffer(nil)
				ww := &responseWriter{ResponseWriter: w, tee: buf}

				next.ServeHTTP(ww, r)

				val := responseValue{
					headers: ww.Header(),
					status:  ww.Status(),
					body:    buf.Bytes(),

					// the handler may not write header and body in some logic,
					// while writing only the body, an attempt is made to write the default header (http.StatusOK)
					skip: ww.IsHeaderWrong(),
				}
				return val, nil
			})

			// the first request to trigger the fetch should return as it's already
			// responded to the client
			if first {
				return
			}

			// handle response for other listeners
			if err != nil {
				panic(fmt.Sprintf("stampede: fail to get value, %v", err))
			}

			resp, ok := val.(responseValue)
			if !ok {
				panic("stampede: handler received unexpected response value type")
			}

			if resp.skip {
				return
			}

			header := w.Header()

		nextHeader:
			for k := range resp.headers {
				for _, match := range stripOutHeaders {
					// Prevent any header in stripOutHeaders to override the current
					// value of that header. This is important when you don't want a
					// header to affect all subsequent requests (for instance, when
					// working with several CORS domains, you don't want the first domain
					// to be recorded an to be printed in all responses)
					if match == k {
						continue nextHeader
					}
				}
				header[k] = resp.headers[k]
			}

			w.WriteHeader(resp.status)
			w.Write(resp.body)
		})
	}
}

// responseValue is response payload we will be coalescing
type responseValue struct {
	headers http.Header
	status  int
	body    []byte
	skip    bool
}

type responseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
	bytes       int
	tee         io.Writer
}

func (b *responseWriter) WriteHeader(code int) {
	if !b.wroteHeader {
		b.code = code
		b.wroteHeader = true
		b.ResponseWriter.WriteHeader(code)
	}
}

func (b *responseWriter) IsHeaderWrong() bool {
	return !b.wroteHeader && (b.code < 100 || b.code > 999)
}

func (b *responseWriter) Write(buf []byte) (int, error) {
	b.maybeWriteHeader()
	n, err := b.ResponseWriter.Write(buf)
	if b.tee != nil {
		_, err2 := b.tee.Write(buf[:n])
		if err == nil {
			err = err2
		}
	}
	b.bytes += n
	return n, err
}

func (b *responseWriter) maybeWriteHeader() {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
}

func (b *responseWriter) Status() int {
	return b.code
}

func (b *responseWriter) BytesWritten() int {
	return b.bytes
}
