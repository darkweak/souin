package httpcache

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// The background refresh started by stale-while-revalidate outlives the
// client request, so Caddy must run the next handlers with the request Souin
// hands it (detached from the client) rather than the client one, whose
// context is canceled as soon as the stale response is sent.
func TestStaleWhileRevalidateRefreshesThroughReverseProxy(t *testing.T) {
	var version atomic.Int64
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		v := version.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Cache-Control", "public, s-maxage=1, stale-while-revalidate=30")
		_, _ = fmt.Fprintf(w, "version %d", v)
	}))
	t.Cleanup(origin.Close)

	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
	{
		admin localhost:2999
		http_port 9080
		cache {
			ttl 5s
			stale 30s
		}
	}
	localhost:9080 {
		route /swr-reverse-proxy {
			cache
			reverse_proxy %s
		}
	}`, strings.TrimPrefix(origin.URL, "http://")), "caddyfile")

	const url = "http://localhost:9080/swr-reverse-proxy"
	get := func() string {
		t.Helper()
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET %s: %v", url, err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}

	if body := get(); body != "version 1" {
		t.Fatalf("first response: %q", body)
	}
	time.Sleep(2 * time.Second)

	if body := get(); body != "version 1" {
		t.Fatalf("the stale response should be served while revalidating, got %q", body)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if version.Load() >= 2 && get() == "version 2" {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("the background refresh never stored the new response (origin calls: %d)", version.Load())
}
