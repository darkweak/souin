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

func Handler(cacheSize int, ttl time.Duration, paths ...string) func(next http.Handler) http.Handler {
	keyFunc := func(r *http.Request) uint64 {
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
	h := HandlerWithKey(cacheSize, ttl, keyFunc)

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

func HandlerWithKey(cacheSize int, ttl time.Duration, keyFunc ...func(r *http.Request) uint64) func(next http.Handler) http.Handler {
	cache := NewCache(cacheSize, ttl, ttl*2)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// cache key for the request
			var key uint64
			if len(keyFunc) > 0 {
				key = keyFunc[0](r)
			} else {
				key = StringToHash(r.Method, strings.ToLower(r.URL.Path))
			}

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

			for k, v := range resp.headers {
				w.Header().Set(k, strings.Join(v, ", "))
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
