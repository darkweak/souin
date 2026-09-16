# Souin shared response counter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `souin_shared_response_counter` Prometheus counter that counts, per concurrent cache-miss group, how many follower requests reused the singleflight leader's response instead of making their own upstream call.

**Architecture:** `pkg/api/prometheus/prometheus.go` already holds a registry of named Prometheus counters/histograms (`registered map[string]interface{}`), populated once in `run()` and mutated via the exported `Increment`/`Add` functions. `pkg/middleware/middleware.go`'s `SouinBaseHandler.Upstream` already coalesces concurrent cache-miss requests for the same key via `golang.org/x/sync/singleflight`. This plan adds one new counter constant + registration entry, and one precise increment call at the point in `Upstream` that already distinguishes "reused a concurrent response" (it currently only logs that fact).

**Tech Stack:** Go, `github.com/prometheus/client_golang/prometheus`, `golang.org/x/sync/singleflight`, standard `testing` + `httptest`.

## Global Constraints

- No `CounterVec`/labels: every distinguishable metric case gets its own plain `Counter` constant, matching every existing entry in `pkg/api/prometheus/prometheus.go`.
- Only the cache-miss path (`SouinBaseHandler.Upstream`) is instrumented. `SouinBaseHandler.Revalidate` uses the same `singleflight` pool but is explicitly out of scope (see `docs/superpowers/specs/2026-07-14-shared-response-counter-design.md`).
- The counter must count **followers only** — a coalesced group of 1 leader + N followers must increment the counter by exactly N, never N+1.
- A follower that ends up making its own individual upstream call anyway (private/`Set-Cookie` response, or `Vary` header mismatch) must **not** be counted — it did not end up reusing the shared response.
- The Traefik plugin's Prometheus override (`plugins/traefik/override/api/prometheus/prometheus.go`) is a no-op stub and needs no change.

---

### Task 1: Register the `souin_shared_response_counter` metric

**Files:**
- Modify: `pkg/api/prometheus/prometheus.go:12-21` (const block), `pkg/api/prometheus/prometheus.go:99-107` (`run()`)
- Test: `pkg/api/prometheus/prometheus_test.go:10-64` (`Test_Run`)

**Interfaces:**
- Produces: `prometheus.SharedResponseCounter` (string constant, value `"souin_shared_response_counter"`) — Task 2 and Task 3 depend on this exact identifier.

- [ ] **Step 1: Extend `Test_Run` to expect the new counter (failing test)**

In `pkg/api/prometheus/prometheus_test.go`, change the count check and add an assertion block for the new key, mirroring the existing blocks exactly:

```go
func Test_Run(t *testing.T) {
	if len(registered) != 0 {
		t.Error("The registered additional metrics array must be empty.")
	}

	run()
	if len(registered) != 6 {
		t.Error("The registered additional metrics array must have 6 items.")
	}

	i, ok := registered[RequestCounter]
	if !ok {
		t.Error("The registered array must have the souin_request_upstream_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_request_upstream_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[RequestRevalidationCounter]
	if !ok {
		t.Error("The registered array must have the souin_request_revalidation_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_request_revalidation_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[NoCachedResponseCounter]
	if !ok {
		t.Error("The registered array must have the souin_no_cached_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_no_cached_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[CachedResponseCounter]
	if !ok {
		t.Error("The registered array must have the souin_cached_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_cached_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[SharedResponseCounter]
	if !ok {
		t.Error("The registered array must have the souin_shared_response_counter key")
	}
	_, ok = i.(*prometheus.Counter)
	if ok {
		t.Errorf("The souin_shared_response_counter element must be a *prometheus.Counter object, %T given.", i)
	}

	i, ok = registered[AvgResponseTime]
	if !ok {
		t.Error("The registered array must have the souin_avg_response_time key")
	}
	_, ok = i.(prometheus.Histogram)
	if !ok {
		t.Errorf("The souin_avg_response_time element must be a prometheus.Histogram object, %T given.", i)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./pkg/api/prometheus/... -run Test_Run -v`
Expected: FAIL — `registered[SharedResponseCounter]` assertion fails (`undefined: SharedResponseCounter` compile error, or once you stub the constant, `len(registered) != 6` since it isn't registered yet).

- [ ] **Step 3: Add the constant and register it**

In `pkg/api/prometheus/prometheus.go`, add the constant to the existing block:

```go
const (
	counter = "counter"
	average = "average"

	RequestCounter             = "souin_request_upstream_counter"
	RequestRevalidationCounter = "souin_request_revalidation_counter"
	NoCachedResponseCounter    = "souin_no_cached_response_counter"
	CachedResponseCounter      = "souin_cached_response_counter"
	SharedResponseCounter      = "souin_shared_response_counter"
	AvgResponseTime            = "souin_avg_response_time"
)
```

And register it in `run()`:

```go
func run() {
	registered = make(map[string]interface{})
	push(counter, RequestCounter, "Total upstream request counter")
	push(counter, RequestRevalidationCounter, "Total revalidation request revalidation counter")
	push(counter, NoCachedResponseCounter, "No cached response counter")
	push(counter, CachedResponseCounter, "Cached response counter")
	push(counter, SharedResponseCounter, "Concurrent requests count that reused a leader's response instead of calling the upstream")
	push(average, AvgResponseTime, "Average response time")
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./pkg/api/prometheus/... -v`
Expected: PASS (`Test_Run`, `Test_Add`, `Test_Increment`, `Test_push` all green)

- [ ] **Step 5: Commit**

```bash
git add pkg/api/prometheus/prometheus.go pkg/api/prometheus/prometheus_test.go
git commit -m "feat(prometheus): register souin_shared_response_counter metric"
```

---

### Task 2: Increment the counter only for follower requests in `Upstream`

**Files:**
- Modify: `pkg/middleware/middleware.go:578-688` (`SouinBaseHandler.Upstream`)

**Interfaces:**
- Consumes: `prometheus.SharedResponseCounter` (from Task 1) and the already-imported `prometheus.Increment(name string)` (`pkg/middleware/middleware.go:26` already imports `github.com/darkweak/souin/pkg/api/prometheus` as `prometheus`).
- Produces: no new exported symbol. Task 3's test observes the effect indirectly via the `/metrics` endpoint.

**Why this needs a local `executedUpstream` flag:** `singleflight.Group.Do` returns `shared bool` that is `true` both for every follower that joined an in-flight call, **and** for the leader itself if at least one follower joined before its own call finished. A bare `if shared` would count the leader once per coalesced group, which is wrong — the leader made the real upstream call, it did not reuse anyone else's response. `executedUpstream` is a local variable, scoped to a single `Upstream()` call (each HTTP request has its own stack frame and its own closure instance passed to `Do`), so no extra synchronization is required: it is only ever written by the one goroutine whose closure actually ran, and only ever read by that same goroutine after `Do` returns.

- [ ] **Step 1: Declare and set the `executedUpstream` flag**

In `pkg/middleware/middleware.go`, inside `Upstream`, add the variable declaration right before the `singleflightPool.Do` call, and set it to `true` as the first line inside the closure:

```go
	singleflightCacheKey := cachedKey
	if s.Configuration.GetDefaultCache().IsCoalescingDisable() || disableCoalescing {
		singleflightCacheKey += uuid.NewString()
	}
	var executedUpstream bool
	sfValue, err, shared := s.singleflightPool.Do(singleflightCacheKey, func() (interface{}, error) {
		executedUpstream = true
		if e := next(customWriter, rq); e != nil {
```

(everything else inside the closure body stays exactly as-is — only the `executedUpstream = true` line is inserted right after the `func() (interface{}, error) {` opening brace, before the existing `if e := next(customWriter, rq); e != nil {` line.)

- [ ] **Step 2: Guard the increment + existing log line with `!executedUpstream`**

Locate the existing block right before the response is written back (currently reads `if shared { ... log ... }`):

```go
		if shared {
			s.Configuration.GetLogger().Infof("Reused response from concurrent request with the key %s", cachedKey)
		}
```

Replace it with:

```go
		if shared && !executedUpstream {
			prometheus.Increment(prometheus.SharedResponseCounter)
			s.Configuration.GetLogger().Infof("Reused response from concurrent request with the key %s", cachedKey)
		}
```

(Note: `shared` is always `true` when `executedUpstream` is `false` — that is what "follower" means in `singleflight`'s semantics — so `shared && !executedUpstream` is logically equivalent to `!executedUpstream`. Keeping the explicit `shared &&` makes the follower/leader distinction clear to a reader at the call site, and mirrors the condition already used one paragraph above for the `disableCoalescing` branch.)

- [ ] **Step 3: Build to confirm it compiles**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Run the existing middleware test suite to confirm no regression**

Run: `go test ./pkg/middleware/... -v -race`
Expected: PASS — in particular `TestSingleflightWinnerWritesBodyTwice`, `TestCoalescedRequestGetsCleanBody`, `TestCancelledRequestDoesNotCorruptCoalescedResponse` must still pass (they exercise the exact code path just modified).

- [ ] **Step 5: Commit**

```bash
git add pkg/middleware/middleware.go
git commit -m "feat(middleware): increment souin_shared_response_counter for coalesced followers"
```

---

### Task 3: Add a test proving the counter counts followers only

**Files:**
- Modify: `pkg/middleware/middleware_test.go` (add new test + import)

**Interfaces:**
- Consumes: `newTestHandler(t)` and `slowNext(body, delay)` (already defined in `pkg/middleware/middleware_test.go:32` and `:46`), `prometheus.InitializePrometheus(configurationtypes.AbstractConfigurationInterface) *prometheus.PrometheusAPI` and `(*PrometheusAPI).HandleRequest(w http.ResponseWriter, r *http.Request)` from `github.com/darkweak/souin/pkg/api/prometheus`, `prometheus.SharedResponseCounter` (from Task 1).

**Why scrape `/metrics` instead of reading `registered` directly:** `pkg/api/prometheus`'s `registered` map and its `getMetricValue` helper are unexported and internal to that package's own tests. From `pkg/middleware`'s test package, the only public way to observe a counter's value is the same way a real Prometheus server would: scrape the `/metrics` HTTP endpoint and parse the exposition-format text. This also means the test must record the counter's value **before** firing requests and assert on the **delta** — other tests in this file (`TestCoalescedRequestGetsCleanBody`, `TestCancelledRequestDoesNotCorruptCoalescedResponse`) already create coalesced groups and will have bumped the same package-global counter earlier in the same test binary run.

- [ ] **Step 1: Add the new test (will fail to compile until Task 1+2 land, but both are already merged from prior tasks — this step is the failing-first verification for this specific test)**

Add this import to the top of `pkg/middleware/middleware_test.go`:

```go
import (
	"bytes"
	baseCtx "context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/api/prometheus"
	"github.com/darkweak/souin/pkg/storage/types"
)
```

(only `"strconv"` and `"github.com/darkweak/souin/pkg/api/prometheus"` are new; the rest already exist.)

Append this test and its helper to the end of `pkg/middleware/middleware_test.go`:

```go
// scrapeCounterValue renders the Prometheus text exposition format via the
// real /metrics handler and extracts the current value of a single counter.
// Returns 0 if the metric line is not present yet (e.g. never incremented).
func scrapeCounterValue(t *testing.T, promAPI *prometheus.PrometheusAPI, name string) float64 {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	promAPI.HandleRequest(rec, req)

	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if !strings.HasPrefix(line, name+" ") {
			continue
		}
		fields := strings.Fields(line)
		v, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			t.Fatalf("failed to parse metric line %q: %v", line, err)
		}
		return v
	}
	return 0
}

// TestSharedResponseCounterCountsFollowersOnly fires 3 concurrent requests
// for the same cache-miss key. Souin's singleflight pool must send exactly
// one of them to the upstream (the leader) and share its response with the
// other 2 (the followers). souin_shared_response_counter must increase by
// exactly 2 (followers only) — not 3 (which would wrongly include the
// leader) and not 0 (which would mean coalescing isn't observed at all).
func TestSharedResponseCounterCountsFollowersOnly(t *testing.T) {
	handler, _ := newTestHandler(t)
	promAPI := prometheus.InitializePrometheus(handler.Configuration)

	before := scrapeCounterValue(t, promAPI, "souin_shared_response_counter")

	const (
		expectedBody = "SHARED_RESPONSE_COUNTER_BODY"
		concurrency  = 3
	)

	upstream := slowNext(expectedBody, 150*time.Millisecond)

	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "http://example.com/test-shared-response-counter", nil)
			rec := httptest.NewRecorder()
			errs[idx] = handler.ServeHTTP(rec, req, upstream)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
	}

	after := scrapeCounterValue(t, promAPI, "souin_shared_response_counter")
	got := after - before
	want := float64(concurrency - 1)
	if got != want {
		t.Errorf("souin_shared_response_counter increased by %v, want %v (concurrency=%d)", got, want, concurrency)
	}
}
```

- [ ] **Step 2: Run the new test to verify it passes**

Run: `go test ./pkg/middleware/... -run TestSharedResponseCounterCountsFollowersOnly -v -race`
Expected: PASS, with `souin_shared_response_counter increased by 2` (i.e. no error logged).

If it fails with `got 3, want 2`: Task 2's `executedUpstream` guard is missing or wrong — re-check Task 2 Step 2.
If it fails with `got 0, want 2`: the 3 goroutines did not coalesce (e.g. `150*time.Millisecond` wasn't enough headroom on a slow CI runner, or `IsCoalescingDisable()` is somehow true in `newTestConfig()`) — increase the delay or inspect `newTestConfig()` in `pkg/middleware/middleware_test.go:17`.

- [ ] **Step 3: Run the full middleware test suite once more**

Run: `go test ./pkg/middleware/... -v -race`
Expected: PASS, all tests including the new one.

- [ ] **Step 4: Commit**

```bash
git add pkg/middleware/middleware_test.go
git commit -m "test(middleware): verify souin_shared_response_counter counts followers only"
```

---

### Task 4: Document the metric and run full verification

**Files:**
- Modify: `README.md:289-294` (Prometheus metrics table)

**Interfaces:**
- Consumes: nothing new.
- Produces: nothing consumed by later tasks (this is the last task).

- [ ] **Step 1: Add the metric row to the README table**

In `README.md`, change:

```markdown
| Key                                | Definition                                          |
|:-----------------------------------|:----------------------------------------------------|
| `souin_request_upstream_counter`   | Count the incoming requests that go to the upstream |
| `souin_no_cached_response_counter` | Count the uncacheable responses                     |
| `souin_cached_response_counter`    | Count the cacheable responses                       |
| `souin_avg_response_time`          | Average response time                               |
```

to:

```markdown
| Key                                | Definition                                          |
|:-----------------------------------|:----------------------------------------------------|
| `souin_request_upstream_counter`   | Count the incoming requests that go to the upstream |
| `souin_no_cached_response_counter` | Count the uncacheable responses                     |
| `souin_cached_response_counter`    | Count the cacheable responses                       |
| `souin_shared_response_counter`    | Count concurrent requests that reused another (leader) request's response instead of calling the upstream |
| `souin_avg_response_time`          | Average response time                               |
```

- [ ] **Step 2: Run the full test suite for the affected modules**

Run: `go test ./pkg/api/prometheus/... ./pkg/middleware/... -v -race`
Expected: PASS, all tests green.

- [ ] **Step 3: Run `go vet` on the whole module**

Run: `go vet ./...`
Expected: no output (no issues).

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: document souin_shared_response_counter metric"
```

---

## Post-plan note

This plan intentionally does not touch `SouinBaseHandler.Revalidate` (revalidation-path coalescing) or add any Grafana dashboard changes — both are explicitly out of scope per `docs/superpowers/specs/2026-07-14-shared-response-counter-design.md`.
