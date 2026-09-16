# Souin shared response counter — design

## Problem

Souin exposes Prometheus metrics for cache hits (`souin_cached_response_counter`),
misses (`souin_no_cached_response_counter`), upstream requests
(`souin_request_upstream_counter`) and revalidation requests
(`souin_request_revalidation_counter`). None of these capture request
coalescing: when several concurrent requests hit the same cache key during a
cache-miss, Souin's `singleflight` pool sends a single request to the
upstream (the "leader") and shares its response with every other concurrent
request for that key (the "followers"). This coalescing behavior is
currently invisible on dashboards — there is no way to distinguish "N
concurrent misses, 1 upstream call" from "N concurrent misses, N upstream
calls".

## Goal

Add a new Prometheus counter, `souin_shared_response_counter`, that counts
every follower request that reused a leader's response instead of making
its own upstream call, so this can be tracked next to the existing
hit/miss graphs.

## Scope

- Only the cache-miss path (`SouinBaseHandler.Upstream` in
  `pkg/middleware/middleware.go`), which is the case described in the
  request: a cache miss where the group's leader is the only one to reach
  the upstream.
- The revalidation path (`SouinBaseHandler.Revalidate`) uses the same
  `singleflight` mechanism but is out of scope for this change — it's a
  materially different case (a stale-but-present entry, not a miss) and
  mixing it into the same counter would make it ambiguous. It can be added
  as its own counter later if needed.

## Design

### 1. New metric constant and registration

In `pkg/api/prometheus/prometheus.go`, add a new counter constant next to
the existing ones and register it in `run()`:

```go
const (
	...
	SharedResponseCounter = "souin_shared_response_counter"
)

func run() {
	...
	push(counter, SharedResponseCounter, "Concurrent requests count that reused a leader's response instead of calling the upstream")
}
```

This mirrors the existing pattern in the file: every distinguishable case
gets its own plain `Counter` constant — the codebase does not use
`CounterVec`/labels anywhere, so this stays consistent with that style.

### 2. Correctly identifying "follower" calls

`golang.org/x/sync/singleflight`'s `Do` returns a `shared bool` that is
`true` for:
- every follower that joined an in-flight call, **and**
- the leader itself, if at least one follower joined before its call
  finished.

A naive `if shared { Increment() }` would therefore over-count by one per
coalesced group (it would count the leader, which did *not* reuse anyone
else's response — it made the real upstream call).

To count only real followers, `Upstream()` will track, via a local `bool`
scoped to that call (each HTTP request has its own stack frame and its own
closure instance, so this needs no extra synchronization), whether *this*
particular goroutine actually executed the upstream call:

```go
var executedUpstream bool
sfValue, err, shared := s.singleflightPool.Do(singleflightCacheKey, func() (interface{}, error) {
	executedUpstream = true
	// ... existing body unchanged ...
})
```

### 3. Increment point

The increment is placed where the existing "Reused response from
concurrent request" log line already lives (after the `disableCoalescing`
retry branch and the `Vary` mismatch branch), so a follower that ends up
making its own individual upstream call (private/`Set-Cookie` response, or
a `Vary` header mismatch) is correctly *not* counted — it didn't end up
reusing the shared response:

```go
if shared && !executedUpstream {
	prometheus.Increment(prometheus.SharedResponseCounter)
	s.Configuration.GetLogger().Infof("Reused response from concurrent request with the key %s", cachedKey)
}
```

Note `shared` is always `true` for a call with `executedUpstream == false`
(that's what "follower" means in `singleflight`'s semantics), so `shared &&
!executedUpstream` is equivalent to `!executedUpstream`; keeping both reads
clearly at the call site.

### 4. Documentation

Add a row to the Prometheus metrics table in `README.md`, next to the
existing `souin_*` metric rows:

| Metric | Description |
| --- | --- |
| `souin_shared_response_counter` | Count of concurrent requests that reused another (leader) request's response instead of calling the upstream |

### 5. Tests

Extend the existing test coverage (`pkg/api/prometheus/prometheus_test.go`
and/or the `Upstream()`-focused middleware tests) with a case that fires N
concurrent requests for the same cache key during a miss and asserts the
counter increases by exactly N-1.

## Out of scope

- Revalidation-path coalescing metric (`Revalidate()`) — not requested,
  can be a follow-up if needed.
- Any Grafana dashboard/JSON changes — outside this repo's concern; the
  user will wire the new metric into their existing dashboard themselves.
- The Traefik plugin's Prometheus override
  (`plugins/traefik/override/api/prometheus/prometheus.go`) is a no-op
  stub (`Increment(string) {}`) and requires no change.
