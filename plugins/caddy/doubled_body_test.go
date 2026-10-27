package httpcache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// phpLikeOrigin is an upstream that ADDS (rather than sets) every header, as
// FrankenPHP does, stamps each execution with its own X-Request-Id and returns
// one JSON document. It is slow so that concurrent identical requests overlap.
type phpLikeOrigin struct {
	calls        atomic.Int64
	cacheControl string
}

func (h *phpLikeOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := h.calls.Add(1)
	time.Sleep(80 * time.Millisecond)
	w.Header().Add("Content-Type", "application/ld+json")
	w.Header().Add("Cache-Control", h.cacheControl)
	w.Header().Add("Vary", "Origin, Accept, Authorization")
	w.Header().Add("X-Request-Id", fmt.Sprintf("exec-%d", n))
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"@id":"/users/1","@type":"User","exec":%d}`, n)
}

// assertSingleDocument asserts that one response carries exactly one upstream execution.
func assertSingleDocument(t *testing.T, label string, res *http.Response) {
	t.Helper()
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()

	if got := res.Header.Values("Content-Type"); len(got) != 1 {
		t.Errorf("%s: want exactly one Content-Type, got %q", label, got)
	}
	if got := res.Header.Values("X-Request-Id"); len(got) != 1 {
		t.Errorf("%s: the origin ran %d times into ONE response (X-Request-Id %q)", label, len(got), got)
	}
	var v map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&v); err != nil {
		t.Errorf("%s: body is not JSON: %v (%q)", label, err, body)
	} else if dec.More() {
		t.Errorf("%s: body holds more than one JSON document: %q", label, body)
	}
	if cl := res.Header.Get("Content-Length"); cl != "" && cl != fmt.Sprint(len(body)) {
		t.Errorf("%s: Content-Length %s != body length %d", label, cl, len(body))
	}
}

func runConcurrentIdentical(t *testing.T, tester *caddytest.Tester, url string, n int, mutate func(i int, r *http.Request)) {
	t.Helper()
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rq, _ := http.NewRequest(http.MethodGet, url, nil)
			rq.Header.Set("Authorization", "Bearer same-token")
			rq.Header.Set("Accept", "application/ld+json")
			if mutate != nil {
				mutate(i, rq)
			}
			<-start
			res, err := tester.Client.Do(rq)
			if err != nil {
				t.Errorf("request %d: %v", i, err)
				return
			}
			assertSingleDocument(t, fmt.Sprintf("request %d", i), res)
		}(i)
	}
	close(start)
	wg.Wait()
}

const doubledBodyCaddyfile = `
	{
		admin localhost:2999
		http_port     9080
		cache {
			allowed_http_verbs GET HEAD
			default_cache_control no-store
			mode strict
			# No disable_coalescing: these tests need coalesced flights.
			key {
				disable_host
				disable_scheme
				disable_body
				hide
				headers Authorization
				hash
			}
		}
	}
	localhost:9080 {
		route /private {
			cache
			reverse_proxy %s
		}
		route /public {
			cache
			reverse_proxy %s
		}
	}`

// startDoubledBodyServer runs each origin on its own ephemeral port, closed when the test ends,
// so repeated runs (-count) never reach a previous run's origin.
func startDoubledBodyServer(t *testing.T, priv, pub http.Handler) *caddytest.Tester {
	t.Helper()
	privSrv := httptest.NewServer(priv)
	t.Cleanup(privSrv.Close)
	pubSrv := httptest.NewServer(pub)
	t.Cleanup(pubSrv.Close)

	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(doubledBodyCaddyfile, privSrv.Listener.Addr(), pubSrv.Listener.Addr()), "caddyfile")

	return tester
}

// Concurrent identical requests for a `private` response must each run the
// upstream once and get exactly one document. Before the fix the executing
// request re-ran the upstream into its own response (every header twice, two
// X-Request-Ids, sometimes two bodies).
func TestConcurrentPrivateResponsesAreNotDoubled(t *testing.T) {
	priv := &phpLikeOrigin{cacheControl: "max-age=0, no-store, private"}
	pub := &phpLikeOrigin{cacheControl: "max-age=0, public, s-maxage=604800"}
	tester := startDoubledBodyServer(t, priv, pub)

	runConcurrentIdentical(t, tester, "http://localhost:9080/private", 8, nil)
	if got := priv.calls.Load(); got != 8 {
		t.Errorf("private: want the origin called once per request (8), got %d", got)
	}
}

// Same shape, public+stored response, and the concurrent requests differ in a
// header the response varies on (Origin), which drives Upstream's re-run branch.
func TestConcurrentPublicResponsesWithVaryAreNotDoubled(t *testing.T) {
	priv := &phpLikeOrigin{cacheControl: "max-age=0, no-store, private"}
	pub := &phpLikeOrigin{cacheControl: "max-age=0, public, s-maxage=604800"}
	tester := startDoubledBodyServer(t, priv, pub)

	runConcurrentIdentical(t, tester, "http://localhost:9080/public", 8, func(i int, r *http.Request) {
		r.Header.Set("Origin", fmt.Sprintf("https://o%d.example", i%2))
	})
}
