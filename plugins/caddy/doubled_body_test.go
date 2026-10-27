package httpcache

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// phpLikeOrigin mimics an API Platform response the way FrankenPHP emits it:
// every header is ADDED (not set) to the writer's header map, every execution
// carries its own X-Request-Id, and the body is one JSON-LD document. A slow
// origin makes concurrent identical requests overlap.
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

// assertSingleDocument is the per-response contract the feed relies on.
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
	dec := json.NewDecoder(bytesReader(body))
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
			# No disable_coalescing: these tests need coalesced flights. (In v1.7.9 the
			# global option never reaches the handler anyway; see Mooply README.)
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
			reverse_proxy localhost:9091
		}
		route /public {
			cache
			reverse_proxy localhost:9092
		}
	}`

// Mooply roadmap #1878: the gateway emitted ONE response carrying two origin
// executions (every origin header twice, two X-Request-Ids, and sometimes two
// bodies). Concurrent identical requests for a `private` response must each
// run the origin once and get exactly one document.
func TestConcurrentPrivateResponsesAreNotDoubled(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(doubledBodyCaddyfile, "caddyfile")

	priv := &phpLikeOrigin{cacheControl: "max-age=0, no-store, private"}
	pub := &phpLikeOrigin{cacheControl: "max-age=0, public, s-maxage=604800"}
	go func() { _ = http.ListenAndServe(":9091", priv) }()
	go func() { _ = http.ListenAndServe(":9092", pub) }()
	time.Sleep(time.Second)

	runConcurrentIdentical(t, tester, "http://localhost:9080/private", 8, nil)
	if got := priv.calls.Load(); got != 8 {
		t.Errorf("private: want the origin called once per request (8), got %d", got)
	}
}

// Same shape, public+stored response, and the concurrent requests differ in a
// header the response varies on (Origin), which drives Upstream's re-run branch.
func TestConcurrentPublicResponsesWithVaryAreNotDoubled(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(doubledBodyCaddyfile, "caddyfile")

	priv := &phpLikeOrigin{cacheControl: "max-age=0, no-store, private"}
	pub := &phpLikeOrigin{cacheControl: "max-age=0, public, s-maxage=604800"}
	go func() { _ = http.ListenAndServe(":9091", priv) }()
	go func() { _ = http.ListenAndServe(":9092", pub) }()
	time.Sleep(time.Second)

	runConcurrentIdentical(t, tester, "http://localhost:9080/public", 8, func(i int, r *http.Request) {
		r.Header.Set("Origin", fmt.Sprintf("https://o%d.example", i%2))
	})
}

func bytesReader(b []byte) io.Reader { return &sliceReader{b: b} }

type sliceReader struct{ b []byte }

func (s *sliceReader) Read(p []byte) (int, error) {
	if len(s.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.b)
	s.b = s.b[n:]
	return n, nil
}
