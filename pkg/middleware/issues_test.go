package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
)

func newTestHandlerWithConfig(t *testing.T, configure func(*BaseConfiguration)) *SouinBaseHandler {
	t.Helper()

	cfg := newTestConfig()
	configure(cfg)

	return NewHTTPCacheHandler(cfg)
}

func serve(t *testing.T, handler *SouinBaseHandler, req *http.Request, next handlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	_ = handler.ServeHTTP(rec, req, next)

	return rec
}

// A stored `no-cache` response must be revalidated before each reuse.
func TestNoCacheResponseIsRevalidated(t *testing.T) {
	handler, _ := newTestHandler(t)

	var calls atomic.Int32
	var conditional atomic.Int32
	next := func(w http.ResponseWriter, r *http.Request) error {
		calls.Add(1)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", `"v1"`)
		if r.Header.Get("If-None-Match") == `"v1"` {
			conditional.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		_, _ = w.Write([]byte("hello"))
		return nil
	}

	for i := 0; i < 3; i++ {
		rec := serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/no-cache", nil), next)
		if rec.Code != http.StatusOK || rec.Body.String() != "hello" {
			t.Fatalf("request %d: unexpected response %d %q (%s)", i, rec.Code, rec.Body.String(), rec.Header().Get("Cache-Status"))
		}
	}

	if calls.Load() != 3 || conditional.Load() != 2 {
		t.Errorf("expected 3 upstream calls, 2 of them conditional, got %d and %d", calls.Load(), conditional.Load())
	}
}

// The qualified `no-cache="field"` only forbids reusing the listed fields.
func TestQualifiedNoCacheResponseIsReusedWithoutListedFields(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cacheControl  string
		upstreamCalls int32
	}{
		{"qualified", `no-cache="X-Secret", max-age=60`, 1},
		{"qualified and unqualified", `no-cache, no-cache="X-Secret", max-age=60`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, _ := newTestHandler(t)

			var calls atomic.Int32
			next := func(w http.ResponseWriter, r *http.Request) error {
				calls.Add(1)
				w.Header().Set("Cache-Control", tc.cacheControl)
				w.Header().Set("X-Secret", "secret")
				w.Header().Set("X-Public", "public")
				_, _ = w.Write([]byte("BODY"))
				return nil
			}

			serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/qualified-"+strings.ReplaceAll(tc.name, " ", "-"), nil), next)
			rec := serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/qualified-"+strings.ReplaceAll(tc.name, " ", "-"), nil), next)

			if calls.Load() != tc.upstreamCalls {
				t.Fatalf("expected %d upstream calls, got %d", tc.upstreamCalls, calls.Load())
			}
			if tc.upstreamCalls == 1 {
				if got := rec.Header().Get("X-Secret"); got != "" {
					t.Errorf("X-Secret is listed in no-cache and must not be served from cache, got %q", got)
				}
				if got := rec.Header().Get("X-Public"); got != "public" {
					t.Errorf("expected X-Public to be served from cache, got %q", got)
				}
			}
		})
	}
}

// Requests coalesced on an upstream call that answered 50x must get that
// answer, not an empty 200.
func TestCoalescedRequestsGetUpstream50x(t *testing.T) {
	handler, _ := newTestHandler(t)

	var calls atomic.Int32
	next := func(w http.ResponseWriter, r *http.Request) error {
		calls.Add(1)
		time.Sleep(300 * time.Millisecond)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("origin says 500"))
		return nil
	}

	const n = 12
	var wg sync.WaitGroup
	recs := make([]*httptest.ResponseRecorder, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recs[i] = serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/500", nil), next)
		}(i)
	}
	wg.Wait()

	for i, rec := range recs {
		if rec.Code != http.StatusInternalServerError || rec.Body.String() != "origin says 500" {
			t.Errorf("request %d: expected 500 %q, got %d %q", i, "origin says 500", rec.Code, rec.Body.String())
		}
	}
	if calls.Load() >= n {
		t.Errorf("expected the requests to be coalesced, got %d upstream calls", calls.Load())
	}
}

// `private` forbids a shared cache to store the response, whatever the
// keying configuration says about Authorization.
func TestPrivateResponseIsNotStored(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cacheControl  string
		authorization string
		keyHeaders    []string
		upstreamCalls int32
	}{
		{"private with Authorization as keying header", "private, max-age=60", "Bearer a", []string{"Authorization"}, 2},
		{"private with s-maxage", "private, s-maxage=60", "", nil, 2},
		{"public authorized response with s-maxage", "public, s-maxage=60", "Bearer a", nil, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newTestHandlerWithConfig(t, func(c *BaseConfiguration) {
				c.DefaultCache.Key = configurationtypes.Key{Headers: tc.keyHeaders}
			})

			var calls atomic.Int32
			next := func(w http.ResponseWriter, r *http.Request) error {
				calls.Add(1)
				w.Header().Set("Cache-Control", tc.cacheControl)
				_, _ = w.Write([]byte("BODY"))
				return nil
			}

			for i := 0; i < 2; i++ {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/private-"+strings.ReplaceAll(tc.name, " ", "-"), nil)
				if tc.authorization != "" {
					req.Header.Set("Authorization", tc.authorization)
				}
				serve(t, handler, req, next)
			}

			if calls.Load() != tc.upstreamCalls {
				t.Errorf("expected %d upstream calls, got %d", tc.upstreamCalls, calls.Load())
			}
		})
	}
}

// A response that sets a cookie isn't stored, unless the origin asked to
// drop the cookie from the stored copy with `no-cache="Set-Cookie"`.
func TestSetCookieResponseIsNotStored(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cacheControl  string
		upstreamCalls int32
	}{
		{"cookie", "public, max-age=60", 2},
		{"cookie excluded by no-cache", `public, max-age=60, no-cache="Set-Cookie"`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, _ := newTestHandler(t)

			var calls atomic.Int32
			next := func(w http.ResponseWriter, r *http.Request) error {
				calls.Add(1)
				w.Header().Set("Cache-Control", tc.cacheControl)
				w.Header().Set("Set-Cookie", "session=abc")
				_, _ = w.Write([]byte("BODY"))
				return nil
			}

			serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/cookie-"+strings.ReplaceAll(tc.name, " ", "-"), nil), next)
			rec := serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/cookie-"+strings.ReplaceAll(tc.name, " ", "-"), nil), next)

			if calls.Load() != tc.upstreamCalls {
				t.Errorf("expected %d upstream calls, got %d (%s)", tc.upstreamCalls, calls.Load(), rec.Header().Get("Cache-Status"))
			}
			if tc.upstreamCalls == 1 && rec.Header().Get("Set-Cookie") != "" {
				t.Errorf("the cookie must not be served from cache, got %q", rec.Header().Get("Set-Cookie"))
			}
		})
	}
}

// Serving or refreshing a stale `stale-if-error` response must not duplicate
// its body.
func TestStaleIfErrorResponseBodyIsNotDuplicated(t *testing.T) {
	handler := newTestHandlerWithConfig(t, func(c *BaseConfiguration) {
		c.DefaultCache.Stale = configurationtypes.Duration{Duration: time.Hour}
	})

	next := func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Cache-Control", "s-maxage=1, stale-if-error=3600")
		_, _ = w.Write([]byte("Hello World"))
		return nil
	}

	for i, wait := range []time.Duration{0, 0, 2100 * time.Millisecond, 0} {
		time.Sleep(wait)
		rec := serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/sie", nil), next)
		if rec.Body.String() != "Hello World" {
			t.Errorf("request %d: expected %q, got %q (%s)", i, "Hello World", rec.Body.String(), rec.Header().Get("Cache-Status"))
		}
	}
}

// Within `stale-while-revalidate` the stale response is served and refreshed
// in the background, even though the client request is over by then.
func TestStaleWhileRevalidateRefreshesInBackground(t *testing.T) {
	handler := newTestHandlerWithConfig(t, func(c *BaseConfiguration) {
		c.DefaultCache.Stale = configurationtypes.Duration{Duration: 0}
	})

	var calls atomic.Int32
	var canceled atomic.Int32
	next := func(w http.ResponseWriter, r *http.Request) error {
		n := calls.Add(1)
		if n > 1 {
			// Like a reverse proxy would, give up on a request that's gone.
			select {
			case <-time.After(50 * time.Millisecond):
			case <-r.Context().Done():
				canceled.Add(1)
				return r.Context().Err()
			}
		}
		w.Header().Set("Cache-Control", "s-maxage=1, stale-while-revalidate=60")
		if n == 1 {
			_, _ = w.Write([]byte("v1"))
		} else {
			_, _ = w.Write([]byte("v2"))
		}
		return nil
	}

	serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/swr", nil), next)
	time.Sleep(2100 * time.Millisecond)

	rec := serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/swr", nil), next)
	if rec.Body.String() != "v1" {
		t.Fatalf("expected the stale response to be served, got %q (%s)", rec.Body.String(), rec.Header().Get("Cache-Status"))
	}

	time.Sleep(300 * time.Millisecond)
	rec = serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/swr", nil), next)
	if rec.Body.String() != "v2" || !strings.Contains(rec.Header().Get("Cache-Status"), "hit") {
		t.Errorf("expected the refreshed response to be served from cache, got %q (%s)", rec.Body.String(), rec.Header().Get("Cache-Status"))
	}
	if canceled.Load() != 0 {
		t.Errorf("the background revalidation ran with a canceled context")
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 upstream calls, got %d", calls.Load())
	}
}

// A request `no-cache` is reported as forwarded because of the request.
func TestRequestNoCacheCacheStatus(t *testing.T) {
	handler, _ := newTestHandler(t)

	next := func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte("BODY"))
		return nil
	}

	serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/rq-no-cache", nil), next)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/rq-no-cache", nil)
	req.Header.Set("Cache-Control", "no-cache")
	rec := serve(t, handler, req, next)

	if status := rec.Header().Get("Cache-Status"); !strings.HasPrefix(status, "Souin; fwd=request;") {
		t.Errorf("expected fwd=request, got %q", status)
	}
}

// After a `304` the stored response is refreshed, so the next request is a
// plain hit.
func TestNotModifiedFreshensStoredResponse(t *testing.T) {
	handler := newTestHandlerWithConfig(t, func(c *BaseConfiguration) {
		c.DefaultCache.Stale = configurationtypes.Duration{Duration: 0}
	})

	var calls atomic.Int32
	next := func(w http.ResponseWriter, r *http.Request) error {
		calls.Add(1)
		w.Header().Set("ETag", `"v1"`)
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.Header().Set("Cache-Control", "max-age=60")
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		w.Header().Set("Cache-Control", "max-age=1")
		_, _ = w.Write([]byte("BODY"))
		return nil
	}

	serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/304", nil), next)
	time.Sleep(2100 * time.Millisecond)
	for i := 0; i < 2; i++ {
		rec := serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/304", nil), next)
		if rec.Code != http.StatusOK || rec.Body.String() != "BODY" {
			t.Fatalf("request %d: unexpected response %d %q", i, rec.Code, rec.Body.String())
		}
	}

	if calls.Load() != 2 {
		t.Errorf("expected 2 upstream calls, got %d", calls.Load())
	}
}

// The request that executed a coalesced flight answered `private` must not
// run the handler a second time into its own response (#886).
func TestCoalescedPrivateResponseIsNotDuplicated(t *testing.T) {
	handler, _ := newTestHandler(t)

	var calls atomic.Int32
	next := func(w http.ResponseWriter, r *http.Request) error {
		n := calls.Add(1)
		time.Sleep(80 * time.Millisecond)
		w.Header().Add("Cache-Control", "private")
		w.Header().Add("X-Request-Id", string(rune('a'+n)))
		_, _ = w.Write([]byte("BODY"))
		return nil
	}

	const concurrent = 8
	recorders := make([]*httptest.ResponseRecorder, concurrent)
	var wg sync.WaitGroup
	for i := range recorders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recorders[i] = serve(t, handler, httptest.NewRequest(http.MethodGet, "http://example.com/coalesced-private", nil), next)
		}(i)
		time.Sleep(time.Millisecond)
	}
	wg.Wait()

	for i, rec := range recorders {
		if ids := rec.Header().Values("X-Request-Id"); len(ids) != 1 {
			t.Errorf("response %d carries %d upstream executions: %v", i, len(ids), ids)
		}
		if body := rec.Body.String(); body != "BODY" {
			t.Errorf("response %d has body %q", i, body)
		}
	}
	if calls.Load() > concurrent {
		t.Errorf("expected at most %d upstream calls, got %d", concurrent, calls.Load())
	}
}
