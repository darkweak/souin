package httpcache

import (
	"net/http"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// Regression test for https://github.com/darkweak/souin/issues/794
// (originally https://github.com/caddyserver/cache-handler/issues/137).
//
// When a stale, must-revalidate response is revalidated, Souin injects the
// stored ETag as If-None-Match and the upstream answers 304. For a client
// that issued an *unconditional* request, that 304 must be turned back into
// the full cached 200 response instead of being propagated as-is.
type issue794Handler struct{}

const issue794Etag = `"issue794-etag"`

func (h *issue794Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("If-None-Match") == issue794Etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "max-age=1, must-revalidate")
	w.Header().Set("ETag", issue794Etag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello issue794!"))
}

func TestUnconditionalRequestDoesNotGet304OnStaleRevalidation(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		cache {
			ttl 1s
			stale 3600s
		}
	}
	localhost:9080 {
		route /issue794 {
			cache
			reverse_proxy localhost:9089
		}
	}`, "caddyfile")

	go func() {
		_ = http.ListenAndServe(":9089", &issue794Handler{})
	}()
	time.Sleep(time.Second)

	// 1st request: unconditional -> 200, stored.
	_, _ = tester.AssertGetResponse(`http://localhost:9080/issue794`, http.StatusOK, "Hello issue794!")

	// Let the cached entry become stale (max-age=1 elapsed, within stale window).
	time.Sleep(2 * time.Second)

	// 2nd request: UNCONDITIONAL (no If-None-Match / If-Modified-Since), max-stale set.
	// Must return the full cached 200, not the 304 produced by the internal
	// revalidation against the upstream.
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/issue794", nil)
	req.Header = http.Header{"Cache-Control": []string{"max-stale=6"}}
	_, _ = tester.AssertResponse(req, http.StatusOK, "Hello issue794!")
}
