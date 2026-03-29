package httpcache

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

type softPurgeOrigin struct {
	mu              sync.Mutex
	version         string
	hits            int
	etag            string
	conditionalHits int
}

func (s *softPurgeOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hits++
	if s.etag != "" && r.Header.Get("If-None-Match") == s.etag {
		s.conditionalHits++
		w.Header().Set("Cache-Control", "max-age=300, stale-while-revalidate=30")
		w.Header().Set("Surrogate-Key", "post-1")
		w.Header().Set("Etag", s.etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "max-age=300, stale-while-revalidate=30")
	w.Header().Set("Surrogate-Key", "post-1")
	if s.etag != "" {
		w.Header().Set("Etag", s.etag)
	}
	_, _ = w.Write([]byte(s.version))
}

func (s *softPurgeOrigin) setVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version = version
	s.etag = `"` + version + `"`
}

func (s *softPurgeOrigin) conditionalHitCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.conditionalHits
}

func TestSoftPurgeServesStaleThenRefreshesInBackground(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		cache {
			api {
				souin
			}
			stale 1m
		}
	}
	localhost:9080 {
		route /soft-purge {
			cache
			reverse_proxy localhost:9088
		}
	}`, "caddyfile")

	origin := &softPurgeOrigin{version: "version-1"}
	go func() {
		_ = http.ListenAndServe(":9088", origin)
	}()

	time.Sleep(time.Second)

	resp1, _ := tester.AssertGetResponse("http://localhost:9080/soft-purge", http.StatusOK, "version-1")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/soft-purge" {
		t.Fatalf("unexpected initial Cache-Status header %q", resp1.Header.Get("Cache-Status"))
	}

	origin.setVersion("version-2")

	purgeReq, _ := http.NewRequest("PURGE", "http://localhost:2999/souin-api/souin", nil)
	purgeReq.Header.Set("Surrogate-Key", "post-1")
	purgeReq.Header.Set("Souin-Purge-Mode", "soft")
	_, _ = tester.AssertResponse(purgeReq, http.StatusNoContent, "")

	resp2, _ := tester.AssertGetResponse("http://localhost:9080/soft-purge", http.StatusOK, "version-1")
	cacheStatus := resp2.Header.Get("Cache-Status")
	if cacheStatus == "" || !containsAll(cacheStatus, "Souin; hit;", "; fwd=stale", "; detail=SOFT-PURGE-SWR") {
		t.Fatalf("unexpected soft purge Cache-Status header %q", cacheStatus)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get("http://localhost:9080/soft-purge")
		if err != nil {
			t.Fatalf("unable to fetch refreshed response: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if string(body) == "version-2" && containsAll(resp.Header.Get("Cache-Status"), "Souin; hit;", "key=GET-http-localhost:9080-/soft-purge") && !strings.Contains(resp.Header.Get("Cache-Status"), "SOFT-PURGED") {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("background refresh did not replace the soft-purged object in time, last body %q last Cache-Status %q", string(body), resp.Header.Get("Cache-Status"))
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}

	return true
}

func TestSoftPurgeConditionalRevalidationWithNotModified(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		cache {
			api {
				souin
			}
			stale 1m
		}
	}
	localhost:9080 {
		route /soft-purge-conditional {
			cache
			reverse_proxy localhost:9089
		}
	}`, "caddyfile")

	origin := &softPurgeOrigin{}
	origin.setVersion("version-1")
	go func() {
		_ = http.ListenAndServe(":9089", origin)
	}()

	time.Sleep(time.Second)

	_, _ = tester.AssertGetResponse("http://localhost:9080/soft-purge-conditional", http.StatusOK, "version-1")

	purgeReq, _ := http.NewRequest("PURGE", "http://localhost:2999/souin-api/souin", nil)
	purgeReq.Header.Set("Surrogate-Key", "post-1")
	purgeReq.Header.Set("Souin-Purge-Mode", "soft")
	_, _ = tester.AssertResponse(purgeReq, http.StatusNoContent, "")

	resp2, _ := tester.AssertGetResponse("http://localhost:9080/soft-purge-conditional", http.StatusOK, "version-1")
	cacheStatus := resp2.Header.Get("Cache-Status")
	if !containsAll(cacheStatus, "Souin; hit;", "; fwd=stale", "; detail=SOFT-PURGE-REVALIDATE") {
		t.Fatalf("unexpected conditional soft purge Cache-Status header %q", cacheStatus)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get("http://localhost:9080/soft-purge-conditional")
		if err != nil {
			t.Fatalf("unable to fetch revalidated response: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if string(body) == "version-1" && containsAll(resp.Header.Get("Cache-Status"), "Souin; hit;", "key=GET-http-localhost:9080-/soft-purge-conditional") && !strings.Contains(resp.Header.Get("Cache-Status"), "SOFT-PURGE") {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("conditional background refresh did not clear the soft purge marker in time, last body %q last Cache-Status %q", string(body), resp.Header.Get("Cache-Status"))
		}

		time.Sleep(100 * time.Millisecond)
	}

	if origin.conditionalHitCount() == 0 {
		t.Fatal("expected background refresh to use conditional revalidation")
	}
}

func TestSoftPurgeWithBypassRequestModeStillServesStaleAndRefreshes(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		cache {
			api {
				souin
			}
			mode bypass_request
			stale 1m
		}
	}
	localhost:9080 {
		route /soft-purge-bypass-request {
			cache
			reverse_proxy localhost:9090
		}
	}`, "caddyfile")

	origin := &softPurgeOrigin{version: "version-1"}
	go func() {
		_ = http.ListenAndServe(":9090", origin)
	}()

	time.Sleep(time.Second)

	primeReq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/soft-purge-bypass-request", nil)
	primeReq.Header.Set("Cache-Control", "no-cache")
	resp1, _ := tester.AssertResponse(primeReq, http.StatusOK, "version-1")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/soft-purge-bypass-request" {
		t.Fatalf("unexpected initial Cache-Status header %q", resp1.Header.Get("Cache-Status"))
	}

	origin.setVersion("version-2")

	purgeReq, _ := http.NewRequest("PURGE", "http://localhost:2999/souin-api/souin", nil)
	purgeReq.Header.Set("Surrogate-Key", "post-1")
	purgeReq.Header.Set("Souin-Purge-Mode", "soft")
	_, _ = tester.AssertResponse(purgeReq, http.StatusNoContent, "")

	staleReq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/soft-purge-bypass-request", nil)
	staleReq.Header.Set("Cache-Control", "no-cache")
	resp2, _ := tester.AssertResponse(staleReq, http.StatusOK, "version-1")
	cacheStatus := resp2.Header.Get("Cache-Status")
	if !containsAll(cacheStatus, "Souin; hit;", "; fwd=stale", "; detail=SOFT-PURGE-SWR") {
		t.Fatalf("unexpected soft purge Cache-Status header with bypass_request %q", cacheStatus)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/soft-purge-bypass-request", nil)
		req.Header.Set("Cache-Control", "no-cache")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("unable to fetch refreshed response: %v", err)
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if string(body) == "version-2" && containsAll(resp.Header.Get("Cache-Status"), "Souin; hit;", "key=GET-http-localhost:9080-/soft-purge-bypass-request") && !strings.Contains(resp.Header.Get("Cache-Status"), "SOFT-PURGED") {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("background refresh did not replace the soft-purged object in bypass_request mode, last body %q last Cache-Status %q", string(body), resp.Header.Get("Cache-Status"))
		}

		time.Sleep(100 * time.Millisecond)
	}
}
