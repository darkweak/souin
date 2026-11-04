package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pquerna/cachecontrol/cacheobject"
)

func TestServeHTTPPropagatesUpstreamAbortToRequestGoroutine(t *testing.T) {
	for _, tc := range []struct {
		name string
		next handlerFunc
	}{
		{"panic", func(http.ResponseWriter, *http.Request) error { panic(http.ErrAbortHandler) }},
		{"wrapped panic", func(http.ResponseWriter, *http.Request) error {
			panic(fmt.Errorf("upstream: %w", http.ErrAbortHandler))
		}},
		{"returned error", func(http.ResponseWriter, *http.Request) error { return http.ErrAbortHandler }},
		{"wrapped error", func(http.ResponseWriter, *http.Request) error {
			return fmt.Errorf("upstream: %w", http.ErrAbortHandler)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, _ := newTestHandler(t)
			request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://synthetic.test/%s/%d.js", t.Name(), time.Now().UnixNano()), nil)
			recorder := httptest.NewRecorder()
			func() {
				defer func() {
					if recovered := recover(); recovered != http.ErrAbortHandler {
						t.Errorf("expected request goroutine to panic http.ErrAbortHandler, got %v", recovered)
					}
				}()
				_ = handler.ServeHTTP(recorder, request, func(w http.ResponseWriter, r *http.Request) error {
					w.Header().Set("Cache-Control", "public, max-age=60")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("prefix"))
					return tc.next(w, r)
				})
			}()
			if recorder.Body.Len() != 0 {
				t.Fatalf("aborted response was sent: %q", recorder.Body.String())
			}
			calls := 0
			for range 2 {
				recorder := httptest.NewRecorder()
				err := handler.ServeHTTP(recorder, request.Clone(request.Context()), func(w http.ResponseWriter, _ *http.Request) error {
					calls++
					w.Header().Set("Cache-Control", "public, max-age=60")
					_, err := w.Write([]byte("healthy"))
					return err
				})
				if err != nil || recorder.Body.String() != "healthy" {
					t.Fatalf("healthy response after abort: body=%q error=%v", recorder.Body.String(), err)
				}
			}
			if calls != 1 {
				t.Fatalf("healthy response should be cached, got %d upstream calls", calls)
			}
		})
	}
}

func TestUpstreamReturnsAbort(t *testing.T) {
	handler, _ := newTestHandler(t)
	request := httptest.NewRequest(http.MethodGet, "http://synthetic.test/upstream.js", nil)
	request = handler.context.SetContext(handler.context.SetBaseContext(request), request)
	writer := NewCustomWriter(request, httptest.NewRecorder(), new(bytes.Buffer))
	cc, _ := cacheobject.ParseRequestCacheControl("")
	started, release := make(chan struct{}), make(chan struct{})
	upstreamErr := make(chan error, 1)
	go func() {
		upstreamErr <- handler.Upstream(writer, request, func(http.ResponseWriter, *http.Request) error {
			close(started)
			<-release
			panic(http.ErrAbortHandler)
		}, cc, "abort", request.URL.Path, false)
	}()
	<-started
	joined := handler.singleflightPool.DoChan("abort", func() (interface{}, error) {
		t.Error("coalesced request unexpectedly called the upstream")
		return nil, nil
	})
	close(release)
	if err := <-upstreamErr; !errors.Is(err, http.ErrAbortHandler) {
		t.Fatalf("expected abort error, got %v", err)
	}
	if result := <-joined; !errors.Is(result.Err, http.ErrAbortHandler) {
		t.Fatalf("coalesced request should receive abort error, got %v", result.Err)
	}
}
