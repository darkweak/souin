package httpcache

import (
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// earlyHints103Handler is a backend that always answers with a 103 Early Hints
// interim response (carrying two Link preload headers) before the final 200.
// It counts how many times it is actually reached so the tests can assert that
// cache hits are served without touching the upstream.
type earlyHints103Handler struct {
	hits int32
}

const (
	earlyHintLinkStyle  = "</styles.css>; rel=preload; as=style"
	earlyHintLinkScript = "</script.js>; rel=preload; as=script"
)

func (h *earlyHints103Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	atomic.AddInt32(&h.hits, 1)

	// Interim 103 response with the resources the client should preload.
	w.Header().Add("Link", earlyHintLinkStyle)
	w.Header().Add("Link", earlyHintLinkScript)
	w.WriteHeader(http.StatusEarlyHints)

	// Final response, cacheable for a minute.
	w.Header().Set("Cache-Control", "max-age=60")
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello early hints!"))
}

// tracedGet performs a GET request capturing every 1xx interim response the
// server sends, so the tests can inspect the replayed 103 Early Hints.
func tracedGet(t *testing.T, url string) (*http.Response, []textproto.MIMEHeader) {
	t.Helper()

	var (
		mu    sync.Mutex
		hints []textproto.MIMEHeader
	)

	trace := &httptrace.ClientTrace{
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			if code == http.StatusEarlyHints {
				mu.Lock()
				hints = append(hints, textproto.MIMEHeader(http.Header(header).Clone()))
				mu.Unlock()
			}

			return nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("cannot build request: %v", err)
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request to %s failed: %v", url, err)
	}

	mu.Lock()
	defer mu.Unlock()

	return resp, hints
}

func assertHasEarlyHintLinks(t *testing.T, hints []textproto.MIMEHeader) {
	t.Helper()

	if len(hints) == 0 {
		t.Fatalf("expected at least one 103 Early Hints interim response, got none")
	}

	var links []string
	for _, hint := range hints {
		links = append(links, http.Header(hint).Values("Link")...)
	}

	var foundStyle, foundScript bool
	for _, link := range links {
		switch link {
		case earlyHintLinkStyle:
			foundStyle = true
		case earlyHintLinkScript:
			foundScript = true
		}
	}

	if !foundStyle || !foundScript {
		t.Errorf("expected both preload Link headers in the early hints, got %v", links)
	}
}

// TestEarlyHints103 ensures the 103 Early Hints emitted by an upstream are
// stored on the cache miss and replayed from the cache on subsequent requests
// without reaching the upstream again.
func TestEarlyHints103(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 60s
		}
	}
	localhost:9080 {
		route /early-hints {
			cache
			reverse_proxy localhost:9091
		}
	}`, "caddyfile")

	backend := &earlyHints103Handler{}
	server := &http.Server{Addr: ":9091", Handler: backend}
	go func() {
		_ = server.ListenAndServe()
	}()
	defer func() {
		_ = server.Close()
	}()
	time.Sleep(time.Second)

	// 1st request: cache miss. The upstream is reached and its 103 Early Hints
	// is forwarded to the client and stored by Souin.
	resp1, hints1 := tracedGet(t, "http://localhost:9080/early-hints")
	if resp1.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code on miss: %d", resp1.StatusCode)
	}
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/early-hints" {
		t.Errorf("unexpected Cache-Status header on miss: %v", resp1.Header.Get("Cache-Status"))
	}
	assertHasEarlyHintLinks(t, hints1)
	_ = resp1.Body.Close()

	if got := atomic.LoadInt32(&backend.hits); got != 1 {
		t.Fatalf("expected the upstream to be reached once, got %d", got)
	}

	// Early hints are stored asynchronously, leave a small window for the
	// background storers to persist them.
	time.Sleep(300 * time.Millisecond)

	// 2nd request: cache hit. Souin must replay the stored 103 Early Hints
	// before the cached 200, without reaching the upstream again.
	resp2, hints2 := tracedGet(t, "http://localhost:9080/early-hints")
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code on hit: %d", resp2.StatusCode)
	}
	compareHit(t, resp2.Header, "GET-http-localhost:9080-/early-hints", "DEFAULT", 60)
	assertHasEarlyHintLinks(t, hints2)
	_ = resp2.Body.Close()

	if got := atomic.LoadInt32(&backend.hits); got != 1 {
		t.Errorf("expected the cache hit to be served without reaching the upstream, total upstream hits %d", got)
	}
}
