package httpcache

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// The global `disable_coalescing` option is parsed into the app's DefaultCache, so FromApp must
// carry it into the handler configuration, like every other global option it inherits.
func TestFromAppInheritsDisableCoalescing(t *testing.T) {
	s := &SouinCaddyMiddleware{}
	app := &SouinApp{DefaultCache: DefaultCache{DisableCoalescing: true}}

	if err := s.FromApp(app); err != nil {
		t.Fatalf("FromApp: %v", err)
	}
	if !s.Configuration.DefaultCache.IsCoalescingDisable() {
		t.Error("the global disable_coalescing was not inherited by the handler")
	}
}

func TestFromAppKeepsCoalescingWhenNotDisabled(t *testing.T) {
	s := &SouinCaddyMiddleware{}
	app := &SouinApp{}

	if err := s.FromApp(app); err != nil {
		t.Fatalf("FromApp: %v", err)
	}
	if s.Configuration.DefaultCache.IsCoalescingDisable() {
		t.Error("coalescing was disabled although neither the app nor the handler disabled it")
	}
}

type slowCountingOrigin struct {
	calls atomic.Int64
}

func (o *slowCountingOrigin) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	o.calls.Add(1)
	time.Sleep(200 * time.Millisecond)
	w.Header().Set("Cache-Control", "public, s-maxage=60")
	_, _ = w.Write([]byte("ok"))
}

func concurrentOriginCalls(t *testing.T, caddyfile string, port string) int64 {
	t.Helper()
	tester := caddytest.NewTester(t)
	tester.InitServer(caddyfile, "caddyfile")

	origin := &slowCountingOrigin{}
	go func() { _ = http.ListenAndServe(":"+port, origin) }()
	time.Sleep(time.Second)

	const n = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := tester.Client.Get("http://localhost:9080/coalescing")
			if err != nil {
				t.Errorf("request: %v", err)
				return
			}
			_ = res.Body.Close()
		}()
	}
	close(start)
	wg.Wait()

	return origin.calls.Load()
}

// End to end through the Caddyfile: with the global option set, identical concurrent cache
// misses each reach the upstream instead of sharing one call.
func TestGlobalDisableCoalescingTakesEffect(t *testing.T) {
	calls := concurrentOriginCalls(t, `
	{
		admin localhost:2999
		http_port 9080
		cache {
			disable_coalescing
		}
	}
	localhost:9080 {
		route /coalescing {
			cache
			reverse_proxy localhost:9093
		}
	}`, "9093")

	if calls != 8 {
		t.Errorf("with disable_coalescing, want one upstream call per request (8), got %d", calls)
	}
}

// Counter-case: without the option the same burst is coalesced into a single upstream call,
// so the test above is measuring the option and not the burst shape.
func TestCoalescingIsOnByDefault(t *testing.T) {
	calls := concurrentOriginCalls(t, `
	{
		admin localhost:2999
		http_port 9080
		cache
	}
	localhost:9080 {
		route /coalescing {
			cache
			reverse_proxy localhost:9094
		}
	}`, "9094")

	if calls != 1 {
		t.Errorf("without disable_coalescing, want the burst coalesced into 1 upstream call, got %d", calls)
	}
}
