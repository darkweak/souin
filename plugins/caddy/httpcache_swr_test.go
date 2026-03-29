package httpcache

import (
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

type staleWhileRevalidateOrigin struct {
	mu      sync.Mutex
	version string
	hits    int
}

func (s *staleWhileRevalidateOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hits++
	w.Header().Set("Cache-Control", "max-age=1, stale-while-revalidate=30")
	_, _ = w.Write([]byte(s.version))
}

func (s *staleWhileRevalidateOrigin) setVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version = version
}

func testStaleWhileRevalidateByMode(t *testing.T, mode string, path string, upstreamPort string) {
	t.Helper()

	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		cache {
			mode `+mode+`
			stale 1m
		}
	}
	localhost:9080 {
		route `+path+` {
			cache
			reverse_proxy localhost:`+upstreamPort+`
		}
	}`, "caddyfile")

	origin := &staleWhileRevalidateOrigin{version: "version-1"}
	go func() {
		_ = http.ListenAndServe(":"+upstreamPort, origin)
	}()

	time.Sleep(time.Second)

	resp1, _ := tester.AssertGetResponse("http://localhost:9080"+path, http.StatusOK, "version-1")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-"+path {
		t.Fatalf("unexpected initial Cache-Status header %q", resp1.Header.Get("Cache-Status"))
	}

	time.Sleep(2 * time.Second)
	origin.setVersion("version-2")

	resp2, _ := tester.AssertGetResponse("http://localhost:9080"+path, http.StatusOK, "version-1")
	if !containsAll(resp2.Header.Get("Cache-Status"), "Souin; hit;", "; fwd=stale") {
		t.Fatalf("expected stale response in %s mode, got Cache-Status %q", mode, resp2.Header.Get("Cache-Status"))
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get("http://localhost:9080" + path)
		if err != nil {
			t.Fatalf("unable to fetch refreshed response in %s mode: %v", mode, err)
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if string(body) == "version-2" && containsAll(resp.Header.Get("Cache-Status"), "Souin; hit;", "key=GET-http-localhost:9080-"+path) && !containsAll(resp.Header.Get("Cache-Status"), "; fwd=stale") {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("background revalidation did not refresh the stale response in %s mode, last body %q last Cache-Status %q", mode, string(body), resp.Header.Get("Cache-Status"))
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func TestStaleWhileRevalidateInBypassRequestMode(t *testing.T) {
	testStaleWhileRevalidateByMode(t, "bypass_request", "/stale-while-revalidate-bypass-request", "9091")
}

func TestStaleWhileRevalidateInStrictMode(t *testing.T) {
	testStaleWhileRevalidateByMode(t, "strict", "/stale-while-revalidate-strict", "9092")
}
