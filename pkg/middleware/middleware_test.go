package middleware

import (
	"bytes"
	baseCtx "context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage/types"
)

func newTestConfig() *BaseConfiguration {
	return &BaseConfiguration{
		DefaultCache: &configurationtypes.DefaultCache{
			TTL:   configurationtypes.Duration{Duration: 30 * time.Second},
			Stale: configurationtypes.Duration{Duration: 30 * time.Second},
			Timeout: configurationtypes.Timeout{
				Backend: configurationtypes.Duration{Duration: 30 * time.Second},
				Cache:   configurationtypes.Duration{Duration: 100 * time.Millisecond},
			},
		},
		LogLevel:             "error",
		SurrogateKeyDisabled: true,
	}
}

func newTestHandler(t *testing.T) (*SouinBaseHandler, types.Storer) {
	t.Helper()

	cfg := newTestConfig()
	handler := NewHTTPCacheHandler(cfg)

	if len(handler.Storers) == 0 {
		t.Fatal("expected at least one storer to be registered")
	}

	return handler, handler.Storers[0]
}

// slowNext returns a handlerFunc that waits for delay before writing body.
func slowNext(body string, delay time.Duration) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return r.Context().Err()
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
		return nil
	}
}

// TestSingleflightWinnerWritesBodyTwice reproduces the bug where the
// singleflight winner gets its body written into the CustomWriter buffer
// twice: once by the upstream handler (next()) inside the singleflight
// callback, and again by the post-Do code that unconditionally writes
// sfWriter.body back into the same buffer. ServeHTTP then calls
// customWriter.Send() which emits the doubled content.
func TestSingleflightWinnerWritesBodyTwice(t *testing.T) {
	handler, _ := newTestHandler(t)

	const expectedBody = "HELLO_WORLD"

	next := slowNext(expectedBody, 0)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test-double-write", nil)
	rec := httptest.NewRecorder()

	err := handler.ServeHTTP(rec, req, next)
	if err != nil {
		t.Fatalf("ServeHTTP failed: %v", err)
	}

	got := rec.Body.String()

	count := strings.Count(got, expectedBody)
	if count > 1 {
		t.Errorf("BUG: response body contains %d copies of the expected body (want 1).\ngot: %q", count, got)
	}
	if count == 0 {
		t.Errorf("response body does not contain expected body.\nwant: %q\ngot:  %q", expectedBody, got)
	}
}

// TestBufferPoolReuseAfterCancellation reproduces the bug where a client
// cancels its request, causing ServeHTTP to return early and put the
// bytes.Buffer back into the pool (`defer s.bufPool.Put(bufPool)`) while
// the upstream goroutine is still writing into it. A subsequent request
// picks up the same buffer from the pool and the two response bodies get
// concatenated or interleaved.
//
// Run with -race to detect the data race on errorCacheCh (line 1120/1129
// in middleware.go): the deferred close(errorCacheCh) races with the
// orphaned goroutine trying to send on the channel.
func TestBufferPoolReuseAfterCancellation(t *testing.T) {
	handler, _ := newTestHandler(t)

	const (
		body1 = "RESPONSE_FROM_CANCELLED_REQUEST"
		body2 = "RESPONSE_FROM_SECOND_REQUEST"
	)

	var (
		callCount int
		mu        sync.Mutex
	)
	upstream := func(w http.ResponseWriter, r *http.Request) error {
		mu.Lock()
		callCount++
		n := callCount
		mu.Unlock()

		body := body1
		if n >= 2 {
			body = body2
		}

		// Simulate slow upstream so the first request can be cancelled
		// while the handler is still running.
		select {
		case <-time.After(200 * time.Millisecond):
		case <-r.Context().Done():
			return r.Context().Err()
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
		return nil
	}

	// --- First request: cancel while upstream is in-flight ---
	ctx1, cancel1 := baseCtx.WithCancel(baseCtx.Background())
	req1 := httptest.NewRequest(http.MethodGet, "http://example.com/test-buffer-reuse", nil)
	req1 = req1.WithContext(ctx1)
	rec1 := httptest.NewRecorder()

	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.ServeHTTP(rec1, req1, upstream)
	}()

	// Let the upstream handler start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel1()

	<-errCh // wait for ServeHTTP to return

	// Give the orphaned goroutine time to finish writing into the
	// buffer that has been returned to the pool.
	time.Sleep(300 * time.Millisecond)

	// --- Second request: should get a clean response ---
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/test-buffer-reuse-2", nil)
	rec2 := httptest.NewRecorder()

	err := handler.ServeHTTP(rec2, req2, upstream)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	got := rec2.Body.String()

	if strings.Contains(got, body1) && strings.Contains(got, body2) {
		t.Errorf("BUG: second response contains data from BOTH requests.\ngot: %q", got)
	}
	if strings.Contains(got, body1) {
		t.Errorf("BUG: second response contains leaked data from cancelled request.\ngot: %q", got)
	}
	if got != body2 {
		t.Errorf("second response body mismatch.\nwant: %q\ngot:  %q", body2, got)
	}
}

// TestCoalescedRequestGetsCleanBody verifies that a request which joins a
// singleflight group (shared=true) receives exactly one copy of the body
// and no corruption from the pool buffer being reused.
//
// Run with -race to detect the data race on bytes.Buffer.Write (writer.go
// line 84): two coalesced goroutines both call Upstream which writes
// sfWriter.body into the same customWriter.Buf concurrently.
func TestCoalescedRequestGetsCleanBody(t *testing.T) {
	handler, _ := newTestHandler(t)

	const expectedBody = "COALESCED_BODY"

	upstream := slowNext(expectedBody, 100*time.Millisecond)

	var wg sync.WaitGroup
	results := make([]string, 2)
	errors := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "http://example.com/test-coalesced", nil)
			rec := httptest.NewRecorder()
			errors[idx] = handler.ServeHTTP(rec, req, upstream)
			results[idx] = rec.Body.String()
		}(i)
	}
	wg.Wait()

	for i, res := range results {
		if errors[i] != nil {
			t.Errorf("request %d failed: %v", i, errors[i])
			continue
		}
		count := strings.Count(res, expectedBody)
		if count > 1 {
			t.Errorf("BUG: request %d got %d copies of body (want 1).\ngot: %q", i, count, res)
		}
		if count == 0 {
			t.Errorf("request %d: body does not contain expected content.\nwant: %q\ngot:  %q", i, expectedBody, res)
		}
	}
}

// TestCancelledRequestDoesNotCorruptCoalescedResponse simulates request A
// and request B hitting the same URL concurrently (coalesced via
// singleflight). Request A is cancelled while upstream is in-flight.
// Request B should still receive a clean, single-copy response — or an
// error if the cancellation propagated through singleflight — but must
// never receive a corrupted (e.g. doubled) body.
//
// Run with -race to detect the data race on errorCacheCh: before the fix
// the cancelled request's ServeHTTP returned and deferred close(errorCacheCh),
// while the upstream goroutine (still running) later tried to send on that
// closed channel.
func TestCancelledRequestDoesNotCorruptCoalescedResponse(t *testing.T) {
	handler, _ := newTestHandler(t)

	const expectedBody = "CANCEL_COALESCE_BODY"

	upstream := slowNext(expectedBody, 200*time.Millisecond)

	// Request A: will be cancelled
	ctxA, cancelA := baseCtx.WithCancel(baseCtx.Background())
	reqA := httptest.NewRequest(http.MethodGet, "http://example.com/test-cancel-coalesce", nil)
	reqA = reqA.WithContext(ctxA)
	recA := httptest.NewRecorder()

	// Request B: should succeed
	reqB := httptest.NewRequest(http.MethodGet, "http://example.com/test-cancel-coalesce", nil)
	recB := httptest.NewRecorder()

	var wg sync.WaitGroup
	var errB error

	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = handler.ServeHTTP(recA, reqA, upstream)
	}()
	go func() {
		defer wg.Done()
		errB = handler.ServeHTTP(recB, reqB, upstream)
	}()

	// Cancel request A after both have started.
	time.Sleep(50 * time.Millisecond)
	cancelA()

	wg.Wait()

	// Request B may succeed or may get a context error if it was coalesced
	// with request A through singleflight. Either outcome is acceptable —
	// what matters is no data race and no body corruption.
	if errB != nil {
		t.Logf("request B returned error (coalesced with cancelled A): %v", errB)
		return
	}

	got := recB.Body.String()
	count := strings.Count(got, expectedBody)
	if count > 1 {
		t.Errorf("BUG: request B got %d copies of body after A was cancelled.\ngot: %q", count, got)
	}
	if count == 0 {
		t.Errorf("request B: body does not contain expected content.\nwant: %q\ngot:  %q", expectedBody, got)
	}
}

// TestBufferBytesSliceCorruption is a focused unit test demonstrating the
// underlying unsafe pattern: bytes.Buffer.Bytes() returns a slice alias
// (not a copy) so returning the buffer to a sync.Pool invalidates the
// slice when another goroutine reuses the buffer. This is exactly what
// happens in the singleflight callback when it captures
// customWriter.Buf.Bytes() and then the buffer is returned to the pool.
func TestBufferBytesSliceCorruption(t *testing.T) {
	pool := &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	// Simulate the singleflight winner writing to the buffer.
	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	original := "ORIGINAL_DATA_PAYLOAD"
	buf.WriteString(original)

	// Take a slice (like: singleflightValue.body = customWriter.Buf.Bytes())
	sliceRef := buf.Bytes()

	if string(sliceRef) != original {
		t.Fatalf("precondition failed: slice should match original")
	}

	// Return buffer to pool (like: defer s.bufPool.Put(bufPool)).
	pool.Put(buf)

	// Another request grabs the same buffer from the pool.
	buf2 := pool.Get().(*bytes.Buffer)
	buf2.Reset()
	corruption := "XXXXXXXXXXXXXXXXXXXXXXX"
	buf2.WriteString(corruption)

	afterReuse := string(sliceRef)
	if afterReuse != original {
		t.Logf("CONFIRMED: buffer slice corrupted after pool reuse.\n  original: %q\n  after:    %q", original, afterReuse)
	} else {
		t.Logf("slice was not corrupted (buffer may have been reallocated to a different address)")
	}

	// Demonstrate the safe alternative: copying.
	buf3 := pool.Get().(*bytes.Buffer)
	buf3.Reset()
	safeData := "SAFE_DATA"
	buf3.WriteString(safeData)

	safeCopy := make([]byte, buf3.Len())
	copy(safeCopy, buf3.Bytes())

	pool.Put(buf3)

	buf4 := pool.Get().(*bytes.Buffer)
	buf4.Reset()
	buf4.WriteString("YYYYYYYYYYYYYYYYYYYYYYYYY")
	_ = buf4

	if string(safeCopy) != safeData {
		t.Errorf("safe copy was corrupted — should never happen: got %q", string(safeCopy))
	}
}
