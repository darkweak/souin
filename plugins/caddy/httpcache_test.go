package httpcache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

func TestMinimal(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /cache-default {
			cache
			respond "Hello, default!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, 200, "Hello, default!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-default" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, 200, "Hello, default!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=119; key=GET-http-localhost:9080-/cache-default; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(2 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, 200, "Hello, default!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=117; key=GET-http-localhost:9080-/cache-default; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
}

func TestHead(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /cache-head {
			cache
			respond "Hello, HEAD!"
		}
	}`, "caddyfile")

	headReq, _ := http.NewRequest(http.MethodHead, "http://localhost:9080/cache-head", nil)
	resp1, _ := tester.AssertResponse(headReq, 200, "")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=HEAD-http-localhost:9080-/cache-head" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}
	if resp1.Header.Get("Content-Length") != "12" {
		t.Errorf("unexpected Content-Length header %v", resp1.Header)
	}

	resp2, _ := tester.AssertResponse(headReq, 200, "")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=119; key=HEAD-http-localhost:9080-/cache-head; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header)
	}
	if resp2.Header.Get("Content-Length") != "12" {
		t.Errorf("unexpected Content-Length header %v", resp2.Header)
	}
}

func TestQueryString(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			key {
				disable_query
			}
		}
	}
	localhost:9080 {
		route /query-string {
			cache {
				key {
					disable_query
				}
			}
			respond "Hello, query string!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/query-string?query=string`, 200, "Hello, query string!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/query-string" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}
}

func TestMaxAge(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /cache-max-age {
			cache
			header Cache-Control "max-age=60"
			respond "Hello, max-age!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/cache-max-age`, 200, "Hello, max-age!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-max-age" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-max-age`, 200, "Hello, max-age!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=59; key=GET-http-localhost:9080-/cache-max-age; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(2 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/cache-max-age`, 200, "Hello, max-age!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=57; key=GET-http-localhost:9080-/cache-max-age; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
}

func TestMaxStale(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			stale 5s
		}
	}
	localhost:9080 {
		route /cache-max-stale {
			cache
			header Cache-Control "max-age=3"
			respond "Hello, max-stale!"
		}
	}`, "caddyfile")

	maxStaleURL := "http://localhost:9080/cache-max-stale"

	resp1, _ := tester.AssertGetResponse(maxStaleURL, 200, "Hello, max-stale!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-max-stale" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}

	resp2, _ := tester.AssertGetResponse(maxStaleURL, 200, "Hello, max-stale!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=2; key=GET-http-localhost:9080-/cache-max-stale; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(3 * time.Second)
	reqMaxStale, _ := http.NewRequest(http.MethodGet, maxStaleURL, nil)
	reqMaxStale.Header = http.Header{"Cache-Control": []string{"max-stale=3"}}
	resp3, _ := tester.AssertResponse(reqMaxStale, 200, "Hello, max-stale!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=-1; key=GET-http-localhost:9080-/cache-max-stale; detail=DEFAULT; fwd=stale" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}

	time.Sleep(3 * time.Second)
	resp4, _ := tester.AssertResponse(reqMaxStale, 200, "Hello, max-stale!")
	if resp4.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-max-stale" {
		t.Errorf("unexpected Cache-Status header %v", resp4.Header.Get("Cache-Status"))
	}
}

func TestSMaxAge(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		https_port 9443
		cache {
			ttl 1000s
		}
	}
	localhost:9080 {
		route /cache-s-maxage {
			cache
			header Cache-Control "s-maxage=5"
			respond "Hello, s-maxage!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/cache-s-maxage`, 200, "Hello, s-maxage!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-s-maxage" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-s-maxage`, 200, "Hello, s-maxage!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/cache-s-maxage; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header with %v", resp2.Header.Get("Cache-Status"))
	}
}

func TestAgeHeader(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 1000s
		}
	}
	localhost:9080 {
		route /age-header {
			cache
			header Cache-Control "max-age=60"
			respond "Hello, Age header!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/age-header`, 200, "Hello, Age header!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/age-header`, 200, "Hello, Age header!")
	if resp2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
	if resp2.Header.Get("Age") != "1" {
		t.Error("Age header should be present")
	}

	time.Sleep(10 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/age-header`, 200, "Hello, Age header!")
	if resp3.Header.Get("Age") != "11" {
		t.Error("Age header should be present")
	}
}

func TestKeyGeneration(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 1000s
		}
	}
	localhost:9080 {
		route /key-template-route {
			cache {
				key {
					template {method}-{host}-{path}-WITH_SUFFIX
				}
			}
			respond "Hello, template route!"
		}
		route /key-headers-route {
			cache {
				key {
					headers X-Header X-Internal
				}
			}
			respond "Hello, headers route!"
		}
		route /key-hash-route {
			cache {
				key {
					hash
				}
			}
			respond "Hello, hash route!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/key-template-route`, 200, "Hello, template route!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}
	if !strings.Contains(resp1.Header.Get("Cache-Status"), "key=GET-localhost-/key-template-route-WITH_SUFFIX") {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/key-template-route`, 200, "Hello, template route!")
	if resp2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
	if resp2.Header.Get("Age") != "1" {
		t.Error("Age header should be present")
	}
	if !strings.Contains(resp2.Header.Get("Cache-Status"), "key=GET-localhost-/key-template-route-WITH_SUFFIX") {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	rq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/key-headers-route", nil)
	rq.Header = http.Header{
		"X-Internal": []string{"my-value"},
	}
	resp1, _ = tester.AssertResponse(rq, 200, "Hello, headers route!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}
	if !strings.Contains(resp1.Header.Get("Cache-Status"), "key=GET-http-localhost:9080-/key-headers-route--my-value") {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}

	rq.Header = http.Header{
		"X-Header":   []string{"first"},
		"X-Internal": []string{"my-value"},
	}
	resp1, _ = tester.AssertResponse(rq, 200, "Hello, headers route!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}
	if !strings.Contains(resp1.Header.Get("Cache-Status"), "key=GET-http-localhost:9080-/key-headers-route-first-my-value") {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}
}

func TestNotHandledRoute(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		https_port 9443
		cache {
			ttl 1000s
			regex {
				exclude ".*handled"
			}
		}
	}
	localhost:9080 {
		route /not-handled {
			cache
			header Cache-Control "max-age=60"
			header Age "max-age=5"
			respond "Hello, Age header!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/not-handled`, 200, "Hello, Age header!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=EXCLUDED-REQUEST-URI" {
		t.Errorf("unexpected Cache-Status header value %v", resp1.Header.Get("Cache-Status"))
	}
}

func TestMaxBodyByte(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		https_port 9443
		cache {
			ttl 5s
			max_cacheable_body_bytes 30
		}
	}
	localhost:9080 {
		route /max-body-bytes-stored {
			cache
			respond "Hello, Max body bytes stored!"
		}
		route /max-body-bytes-not-stored {
			cache
			respond "Hello, Max body bytes not stored due to the response length!"
		}
	}`, "caddyfile")

	respStored1, _ := tester.AssertGetResponse(`http://localhost:9080/max-body-bytes-stored`, 200, "Hello, Max body bytes stored!")
	respStored2, _ := tester.AssertGetResponse(`http://localhost:9080/max-body-bytes-stored`, 200, "Hello, Max body bytes stored!")
	if respStored1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/max-body-bytes-stored" {
		t.Errorf("unexpected Cache-Status header value %v", respStored1.Header.Get("Cache-Status"))
	}
	if respStored1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", respStored1.Header.Get("Age"))
	}

	if respStored2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/max-body-bytes-stored; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header value %v", respStored2.Header.Get("Cache-Status"))
	}
	if respStored2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}

	respNotStored1, _ := tester.AssertGetResponse(`http://localhost:9080/max-body-bytes-not-stored`, 200, "Hello, Max body bytes not stored due to the response length!")
	respNotStored2, _ := tester.AssertGetResponse(`http://localhost:9080/max-body-bytes-not-stored`, 200, "Hello, Max body bytes not stored due to the response length!")
	if respNotStored1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; detail=UPSTREAM-RESPONSE-TOO-LARGE; key=GET-http-localhost:9080-/max-body-bytes-not-stored" {
		t.Errorf("unexpected Cache-Status header value %v", respNotStored1.Header.Get("Cache-Status"))
	}
	if respNotStored1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", respNotStored1.Header.Get("Age"))
	}

	if respNotStored2.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; detail=UPSTREAM-RESPONSE-TOO-LARGE; key=GET-http-localhost:9080-/max-body-bytes-not-stored" {
		t.Errorf("unexpected Cache-Status header value %v", respNotStored2.Header.Get("Cache-Status"))
	}
	if respNotStored2.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", respNotStored2.Header.Get("Age"))
	}
}

func TestAuthenticatedRoute(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port 9080
		https_port 9443
		cache {
			ttl 1000s
		}
	}
	localhost:9080 {
		route /no-auth-bypass {
			cache
			respond "Hello, auth {http.request.header.Authorization}!"
		}
		route /auth-bypass {
			cache {
				key {
					headers Authorization Content-Type
				}
			}
			header Cache-Control "private, s-maxage=5"
			respond "Hello, auth bypass {http.request.header.Authorization}!"
		}
		route /auth-bypass-vary {
			cache {
				key {
					headers Authorization Content-Type
				}
			}
			header Cache-Control "private, s-maxage=5"
			header Vary "Content-Type, Authorization"
			respond "Hello, auth vary bypass {http.request.header.Authorization}!"
		}
	}`, "caddyfile")

	getRequestFor := func(endpoint, user string) *http.Request {
		rq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080"+endpoint, nil)
		rq.Header = http.Header{"Authorization": []string{"Bearer " + user}, "Content-Type": []string{"text/plain"}}

		return rq
	}

	respNoAuthBypass, _ := tester.AssertResponse(getRequestFor("/no-auth-bypass", "Alice"), 200, "Hello, auth Bearer Alice!")
	if respNoAuthBypass.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; key=GET-http-localhost:9080-/no-auth-bypass; detail=PRIVATE-OR-AUTHENTICATED-RESPONSE" {
		t.Errorf("unexpected Cache-Status header %v", respNoAuthBypass.Header.Get("Cache-Status"))
	}

	respAuthBypassAlice1, _ := tester.AssertResponse(getRequestFor("/auth-bypass", "Alice"), 200, "Hello, auth bypass Bearer Alice!")
	if respAuthBypassAlice1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/auth-bypass-Bearer Alice-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassAlice1.Header.Get("Cache-Status"))
	}
	respAuthBypassAlice2, _ := tester.AssertResponse(getRequestFor("/auth-bypass", "Alice"), 200, "Hello, auth bypass Bearer Alice!")
	if respAuthBypassAlice2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/auth-bypass-Bearer Alice-text/plain; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassAlice2.Header.Get("Cache-Status"))
	}

	respAuthBypassBob1, _ := tester.AssertResponse(getRequestFor("/auth-bypass", "Bob"), 200, "Hello, auth bypass Bearer Bob!")
	if respAuthBypassBob1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/auth-bypass-Bearer Bob-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassBob1.Header.Get("Cache-Status"))
	}
	respAuthBypassBob2, _ := tester.AssertResponse(getRequestFor("/auth-bypass", "Bob"), 200, "Hello, auth bypass Bearer Bob!")
	if respAuthBypassBob2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/auth-bypass-Bearer Bob-text/plain; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassBob2.Header.Get("Cache-Status"))
	}

	respAuthVaryBypassAlice1, _ := tester.AssertResponse(getRequestFor("/auth-bypass-vary", "Alice"), 200, "Hello, auth vary bypass Bearer Alice!")
	if respAuthVaryBypassAlice1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/auth-bypass-vary-Bearer Alice-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthVaryBypassAlice1.Header.Get("Cache-Status"))
	}
	respAuthVaryBypassAlice2, _ := tester.AssertResponse(getRequestFor("/auth-bypass-vary", "Alice"), 200, "Hello, auth vary bypass Bearer Alice!")
	if respAuthVaryBypassAlice2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/auth-bypass-vary-Bearer Alice-text/plain; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header %v", respAuthVaryBypassAlice2.Header.Get("Cache-Status"))
	}
}

type testErrorHandler struct {
	iterator int
}

func (t *testErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.iterator++
	if t.iterator%2 == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello must-revalidate!"))
}

func TestMustRevalidate(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		cache {
			ttl 5s
			stale 5s
		}
	}
	localhost:9080 {
		route /cache-default {
			cache
			reverse_proxy localhost:9081
		}
	}`, "caddyfile")

	go func() {
		errorHandler := testErrorHandler{}
		_ = http.ListenAndServe(":9081", &errorHandler)
	}()
	time.Sleep(time.Second)
	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, http.StatusOK, "Hello must-revalidate!")
	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, http.StatusOK, "Hello must-revalidate!")
	time.Sleep(6 * time.Second)
	staleReq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/cache-default", nil)
	staleReq.Header = http.Header{"Cache-Control": []string{"max-stale=3, stale-if-error=84600"}}
	resp3, _ := tester.AssertResponse(staleReq, http.StatusOK, "Hello must-revalidate!")

	if resp1.Header.Get("Cache-Control") != "must-revalidate" {
		t.Errorf("unexpected resp1 Cache-Control header %v", resp1.Header.Get("Cache-Control"))
	}
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-default" {
		t.Errorf("unexpected resp1 Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected resp1 Age header %v", resp1.Header.Get("Age"))
	}

	if resp2.Header.Get("Cache-Control") != "must-revalidate" {
		t.Errorf("unexpected resp2 Cache-Control header %v", resp2.Header.Get("Cache-Control"))
	}
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/cache-default; detail=DEFAULT" {
		t.Errorf("unexpected resp2 Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}
	if resp2.Header.Get("Age") != "1" {
		t.Errorf("unexpected resp2 Age header %v", resp2.Header.Get("Age"))
	}

	if resp3.Header.Get("Cache-Control") != "must-revalidate" {
		t.Errorf("unexpected resp3 Cache-Control header %v", resp3.Header.Get("Cache-Control"))
	}
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=-2; key=GET-http-localhost:9080-/cache-default; detail=DEFAULT; fwd=stale; fwd-status=500" {
		t.Errorf("unexpected resp3 Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
	if resp3.Header.Get("Age") != "7" {
		t.Errorf("unexpected resp3 Age header %v", resp3.Header.Get("Age"))
	}

	resp4, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, http.StatusOK, "Hello must-revalidate!")
	if resp4.Header.Get("Cache-Control") != "must-revalidate" {
		t.Errorf("unexpected resp4 Cache-Control header %v", resp4.Header.Get("Cache-Control"))
	}
	if resp4.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-default" {
		t.Errorf("unexpected resp4 Cache-Status header %v", resp4.Header.Get("Cache-Status"))
	}
	if resp4.Header.Get("Age") != "" {
		t.Errorf("unexpected resp4 Age header %v", resp4.Header.Get("Age"))
	}

	time.Sleep(6 * time.Second)
	staleReq, _ = http.NewRequest(http.MethodGet, "http://localhost:9080/cache-default", nil)
	staleReq.Header = http.Header{"Cache-Control": []string{"max-stale=3"}}
	resp5, _ := tester.AssertResponse(staleReq, http.StatusGatewayTimeout, "")

	if resp5.Header.Get("Cache-Status") != "Souin; fwd=request; fwd-status=500; key=GET-http-localhost:9080-/cache-default; detail=REQUEST-REVALIDATION" {
		t.Errorf("unexpected resp5 Cache-Status header %v", resp4.Header.Get("Cache-Status"))
	}
	if resp5.Header.Get("Age") != "" {
		t.Errorf("unexpected resp5 Age header %v", resp4.Header.Get("Age"))
	}
}

type staleIfErrorHandler struct {
	iterator int
}

func (t *staleIfErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.iterator > 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t.iterator++
	w.Header().Set("Cache-Control", "stale-if-error=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello stale-if-error!"))
}

func TestStaleIfError(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		cache {
			ttl 5s
			stale 5s
		}
	}
	localhost:9080 {
		route /stale-if-error {
			cache
			reverse_proxy localhost:9085
		}
	}`, "caddyfile")

	go func() {
		staleIfErrorHandler := staleIfErrorHandler{}
		_ = http.ListenAndServe(":9085", &staleIfErrorHandler)
	}()
	time.Sleep(time.Second)
	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/stale-if-error`, http.StatusOK, "Hello stale-if-error!")
	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/stale-if-error`, http.StatusOK, "Hello stale-if-error!")

	if resp1.Header.Get("Cache-Control") != "stale-if-error=86400" {
		t.Errorf("unexpected resp1 Cache-Control header %v", resp1.Header.Get("Cache-Control"))
	}
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/stale-if-error" {
		t.Errorf("unexpected resp1 Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected resp1 Age header %v", resp1.Header.Get("Age"))
	}

	if resp2.Header.Get("Cache-Control") != "stale-if-error=86400" {
		t.Errorf("unexpected resp2 Cache-Control header %v", resp2.Header.Get("Cache-Control"))
	}
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/stale-if-error; detail=DEFAULT" {
		t.Errorf("unexpected resp2 Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}
	if resp2.Header.Get("Age") != "1" {
		t.Errorf("unexpected resp2 Age header %v", resp2.Header.Get("Age"))
	}

	time.Sleep(6 * time.Second)
	staleReq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/stale-if-error", nil)
	staleReq.Header = http.Header{"Cache-Control": []string{"stale-if-error=86400"}}
	resp3, _ := tester.AssertResponse(staleReq, http.StatusOK, "Hello stale-if-error!")

	if resp3.Header.Get("Cache-Control") != "stale-if-error=86400" {
		t.Errorf("unexpected resp3 Cache-Control header %v", resp3.Header.Get("Cache-Control"))
	}
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=-2; key=GET-http-localhost:9080-/stale-if-error; detail=DEFAULT; fwd=stale; fwd-status=500" {
		t.Errorf("unexpected resp3 Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
	if resp3.Header.Get("Age") != "7" {
		t.Errorf("unexpected resp3 Age header %v", resp3.Header.Get("Age"))
	}

	resp4, _ := tester.AssertGetResponse(`http://localhost:9080/stale-if-error`, http.StatusOK, "Hello stale-if-error!")

	if resp4.Header.Get("Cache-Status") != "Souin; hit; ttl=-2; key=GET-http-localhost:9080-/stale-if-error; detail=DEFAULT; fwd=stale; fwd-status=500" &&
		resp4.Header.Get("Cache-Status") != "Souin; hit; ttl=-3; key=GET-http-localhost:9080-/stale-if-error; detail=DEFAULT; fwd=stale; fwd-status=500" {
		t.Errorf("unexpected resp4 Cache-Status header %v", resp4.Header.Get("Cache-Status"))
	}

	if resp4.Header.Get("Age") != "7" && resp4.Header.Get("Age") != "8" {
		t.Errorf("unexpected resp4 Age header %v", resp4.Header.Get("Age"))
	}

	time.Sleep(6 * time.Second)
	resp5, _ := tester.AssertGetResponse(`http://localhost:9080/stale-if-error`, http.StatusInternalServerError, "")

	if resp5.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; key=GET-http-localhost:9080-/stale-if-error; detail=UNCACHEABLE-STATUS-CODE" {
		t.Errorf("unexpected resp5 Cache-Status header %v", resp5.Header.Get("Cache-Status"))
	}

	if resp5.Header.Get("Age") != "" {
		t.Errorf("unexpected resp5 Age header %v", resp5.Header.Get("Age"))
	}
}

type testETagsHandler struct{}

const etagValue = "AAA-BBB"

func (t *testETagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("If-None-Match"), etagValue) {
		w.WriteHeader(http.StatusNotModified)

		return
	}
	w.Header().Set("ETag", etagValue)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello etag!"))
}

func Test_ETags(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		cache {
			ttl 50s
			stale 50s
		}
	}
	localhost:9080 {
		route /etags {
			cache
			reverse_proxy localhost:9082
		}
	}`, "caddyfile")

	go func() {
		etagsHandler := testETagsHandler{}
		_ = http.ListenAndServe(":9082", &etagsHandler)
	}()
	_, _ = tester.AssertGetResponse(`http://localhost:9080/etags`, http.StatusOK, "Hello etag!")
	staleReq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/etags", nil)
	staleReq.Header = http.Header{"If-None-Match": []string{etagValue}}
	_, _ = tester.AssertResponse(staleReq, http.StatusNotModified, "")
	staleReq.Header = http.Header{}
	_, _ = tester.AssertResponse(staleReq, http.StatusOK, "Hello etag!")
	staleReq.Header = http.Header{"If-None-Match": []string{etagValue}}
	_, _ = tester.AssertResponse(staleReq, http.StatusNotModified, "")
	staleReq.Header = http.Header{"If-None-Match": []string{"other"}}
	_, _ = tester.AssertResponse(staleReq, http.StatusOK, "Hello etag!")
}

type testHugeMaxAgeHandler struct{}

func (t *testHugeMaxAgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello, huge max age!"))
}

func TestHugeMaxAgeHandler(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /huge-max-age {
			cache
			reverse_proxy localhost:9083
		}
	}`, "caddyfile")

	go func() {
		hugeMaxAgeHandler := testHugeMaxAgeHandler{}
		_ = http.ListenAndServe(":9083", &hugeMaxAgeHandler)
	}()
	time.Sleep(time.Second)

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/huge-max-age`, 200, "Hello, huge max age!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/huge-max-age" {
		t.Error("Cache-Status header should be present")
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/huge-max-age`, 200, "Hello, huge max age!")
	if resp2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
	if resp2.Header.Get("Age") != "1" {
		t.Error("Age header should be present")
	}
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=599; key=GET-http-localhost:9080-/huge-max-age; detail=DEFAULT" {
		t.Error("Cache-Status header should be present")
	}

	time.Sleep(2 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/huge-max-age`, 200, "Hello, huge max age!")
	if resp3.Header.Get("Age") != "3" {
		t.Error("Age header should be present")
	}
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=597; key=GET-http-localhost:9080-/huge-max-age; detail=DEFAULT" {
		t.Error("Cache-Status header should be present")
	}
}

type testVaryHandler struct{}

const variedHeader = "X-Varied"

func (t *testVaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(50 * time.Millisecond)
	w.Header().Set("Vary", variedHeader)
	w.Header().Set(variedHeader, r.Header.Get(variedHeader))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("Hello, vary %s!", r.Header.Get(variedHeader))))
}

func TestVaryHandler(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /vary-multiple {
			cache
			reverse_proxy localhost:9084
		}
	}`, "caddyfile")

	go func() {
		varyHandler := testVaryHandler{}
		_ = http.ListenAndServe(":9084", &varyHandler)
	}()
	time.Sleep(time.Second)

	baseRq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/vary-multiple", nil)

	rq1 := baseRq.Clone(context.Background())
	rq1.Header.Set(variedHeader, "first")
	rq2 := baseRq.Clone(context.Background())
	rq2.Header.Set(variedHeader, "second")
	rq3 := baseRq.Clone(context.Background())
	rq3.Header.Set(variedHeader, "third")
	rq4 := baseRq.Clone(context.Background())
	rq4.Header.Set(variedHeader, "fourth")

	requests := []*http.Request{
		rq1,
		rq2,
		rq3,
		rq4,
	}

	var wg sync.WaitGroup
	resultMap := &sync.Map{}

	for i, rq := range requests {
		wg.Add(1)

		go func(r *http.Request, iteration int) {
			defer wg.Done()
			res, _ := tester.AssertResponse(r, 200, fmt.Sprintf("Hello, vary %s!", r.Header.Get(variedHeader)))
			resultMap.Store(iteration, res)
		}(rq, i)
	}

	wg.Wait()

	for i := 0; i < 4; i++ {
		if res, ok := resultMap.Load(i); !ok {
			t.Errorf("unexpected nil response for iteration %d", i)
		} else {
			rs, ok := res.(*http.Response)
			if !ok {
				t.Error("The object is not type of *http.Response")
			}

			if rs.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/vary-multiple" {
				t.Errorf("The response %d doesn't match the expected header: %s", i, rs.Header.Get("Cache-Status"))
			}
		}
	}

	for i, rq := range requests {
		wg.Add(1)

		go func(r *http.Request, iteration int) {
			defer wg.Done()
			res, _ := tester.AssertResponse(r, 200, fmt.Sprintf("Hello, vary %s!", r.Header.Get(variedHeader)))
			resultMap.Store(iteration, res)
		}(rq, i)
	}

	wg.Wait()

	checker := func(res any, ttl int) {
		rs, ok := res.(*http.Response)
		if !ok {
			t.Error("The object is not type of *http.Response")
		}

		nextTTL := ttl - 1
		if (rs.Header.Get("Cache-Status") != fmt.Sprintf("Souin; hit; ttl=%d; key=GET-http-localhost:9080-/vary-multiple; detail=DEFAULT", ttl) || rs.Header.Get("Age") != fmt.Sprint(120-ttl)) &&
			(rs.Header.Get("Cache-Status") != fmt.Sprintf("Souin; hit; ttl=%d; key=GET-http-localhost:9080-/vary-multiple; detail=DEFAULT", nextTTL) || rs.Header.Get("Age") != fmt.Sprint(120-nextTTL)) {
			t.Errorf("The response doesn't match the expected header or age: %s => %s", rs.Header.Get("Cache-Status"), rs.Header.Get("Age"))
		}
	}

	if res, ok := resultMap.Load(0); !ok {
		t.Errorf("unexpected nil response for iteration %d", 0)
	} else {
		checker(res, 119)
	}

	if res, ok := resultMap.Load(1); !ok {
		t.Errorf("unexpected nil response for iteration %d", 1)
	} else {
		checker(res, 119)
	}

	if res, ok := resultMap.Load(2); !ok {
		t.Errorf("unexpected nil response for iteration %d", 2)
	} else {
		checker(res, 119)
	}

	if res, ok := resultMap.Load(3); !ok {
		t.Errorf("unexpected nil response for iteration %d", 3)
	} else {
		checker(res, 119)
	}
}

func TestDisabledVaryHandler(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /vary-multiple {
			cache {
				key {
					disable_vary
				}
			}
			reverse_proxy localhost:9084
		}
	}`, "caddyfile")

	go func() {
		varyHandler := testVaryHandler{}
		_ = http.ListenAndServe(":9084", &varyHandler)
	}()
	time.Sleep(time.Second)

	baseRq, _ := http.NewRequest(http.MethodGet, "http://localhost:9080/vary-multiple", nil)

	rq1 := baseRq.Clone(context.Background())
	rq1.Header.Set(variedHeader, "first")
	rq2 := baseRq.Clone(context.Background())
	rq2.Header.Set(variedHeader, "second")
	rq3 := baseRq.Clone(context.Background())
	rq3.Header.Set(variedHeader, "third")
	rq4 := baseRq.Clone(context.Background())
	rq4.Header.Set(variedHeader, "fourth")

	requests := []*http.Request{
		rq1,
		rq2,
		rq3,
		rq4,
	}

	resultMap := &sync.Map{}

	for i, rq := range requests {
		res, _ := tester.AssertResponse(rq, 200, "Hello, vary first!")
		resultMap.Store(i, res)
	}
}

func TestESITags(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 1000s
		}
	}
	localhost:9080 {
		route /esi-include-1 {
			cache
			respond "esi-include-1 with some long content to ensure the compute works well. Also add some dummy text with some $pecial characters without recursive esi includes"
		}
		route /esi-include-2 {
			cache
			respond "esi-include-2"
		}
		route /esi-path {
			cache
			header Cache-Control "max-age=60"
			respond "Hello <esi:include src=\"http://localhost:9080/esi-include-1\"/> and <esi:include src=\"http://localhost:9080/esi-include-2\"/>!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/esi-path`, 200, "Hello esi-include-1 with some long content to ensure the compute works well. Also add some dummy text with some $pecial characters without recursive esi includes and esi-include-2!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/esi-path" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}
	if resp1.Header.Get("Content-Length") != "180" {
		t.Errorf("unexpected Content-Length header %v", resp1.Header.Get("Content-Length"))
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/esi-path`, 200, "Hello esi-include-1 with some long content to ensure the compute works well. Also add some dummy text with some $pecial characters without recursive esi includes and esi-include-2!")
	if resp2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
	if resp2.Header.Get("Age") != "1" {
		t.Error("Age header should be present")
	}

	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/esi-include-1`, 200, "esi-include-1 with some long content to ensure the compute works well. Also add some dummy text with some $pecial characters without recursive esi includes")
	if resp3.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
	if resp3.Header.Get("Cache-Status") == "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/esi-include-1" {
		t.Error("Cache-Status should be already stored")
	}

	resp4, _ := tester.AssertGetResponse(`http://localhost:9080/esi-include-2`, 200, "esi-include-2")
	if resp4.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
	if resp4.Header.Get("Cache-Status") == "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/esi-include-2" {
		t.Error("Cache-Status should be already stored")
	}
}

func TestCacheableStatusCode(t *testing.T) {
	caddyTester := caddytest.NewTester(t)
	caddyTester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 10s
		}
	}
	localhost:9080 {
		cache

		respond /cache-200 "" 200 {
			close
		}
		respond /cache-204 "" 204 {
			close
		}
		respond /cache-301 "" 301 {
			close
		}
		respond /cache-405 "" 405 {
			close
		}
	}`, "caddyfile")

	cacheChecker := func(tester *caddytest.Tester, path string, expectedStatusCode int, expectedCached bool) {
		resp1, _ := tester.AssertGetResponse("http://localhost:9080"+path, expectedStatusCode, "")
		if resp1.Header.Get("Age") != "" {
			t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
		}

		cacheStatus := "Souin; fwd=uri-miss; "
		if expectedCached {
			cacheStatus += "stored; "
		} else {
			cacheStatus += "detail=UPSTREAM-ERROR-OR-EMPTY-RESPONSE; "
		}
		cacheStatus += "key=GET-http-localhost:9080-" + path

		if resp1.Header.Get("Cache-Status") != cacheStatus {
			t.Errorf("unexpected first Cache-Status header %v", resp1.Header.Get("Cache-Status"))
		}

		resp1, _ = tester.AssertGetResponse("http://localhost:9080"+path, expectedStatusCode, "")

		cacheStatus = "Souin; "
		detail := ""
		if expectedCached {
			if resp1.Header.Get("Age") != "1" {
				t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
			}
			cacheStatus += "hit; ttl=9; "
			detail = "; detail=DEFAULT"
		} else {
			cacheStatus += "fwd=uri-miss; detail=UPSTREAM-ERROR-OR-EMPTY-RESPONSE; "
		}
		cacheStatus += "key=GET-http-localhost:9080-" + path + detail

		if resp1.Header.Get("Cache-Status") != cacheStatus {
			t.Errorf("unexpected second Cache-Status header %v", resp1.Header.Get("Cache-Status"))
		}
	}

	cacheChecker(caddyTester, "/cache-200", 200, false)
	cacheChecker(caddyTester, "/cache-204", 204, true)
	cacheChecker(caddyTester, "/cache-301", 301, true)
	cacheChecker(caddyTester, "/cache-405", 405, true)
}

func TestExpires(t *testing.T) {
	expiresValue := time.Now().Add(time.Hour * 24)
	caddyTester := caddytest.NewTester(t)
	caddyTester.InitServer(fmt.Sprintf(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 10s
		}
	}
	localhost:9080 {
		route /expires-only {
			cache
			header Expires "%[1]s"
			respond "Hello, expires-only!"
		}
		route /expires-with-max-age {
			cache
			header Expires "%[1]s"
			header Cache-Control "max-age=60"
			respond "Hello, expires-with-max-age!"
		}
		route /expires-with-s-maxage {
			cache
			header Expires "%[1]s"
			header Cache-Control "s-maxage=5"
			respond "Hello, expires-with-s-maxage!"
		}
	}`, expiresValue.Format(time.RFC1123)), "caddyfile")

	cacheChecker := func(tester *caddytest.Tester, path string, expectedBody string, expectedDuration int) {
		resp1, _ := tester.AssertGetResponse("http://localhost:9080"+path, 200, expectedBody)
		if resp1.Header.Get("Age") != "" {
			t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
		}

		if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-"+path {
			t.Errorf("unexpected first Cache-Status header %v", resp1.Header.Get("Cache-Status"))
		}

		resp1, _ = tester.AssertGetResponse("http://localhost:9080"+path, 200, expectedBody)

		if resp1.Header.Get("Age") != "1" {
			t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
		}

		if resp1.Header.Get("Cache-Status") != fmt.Sprintf("Souin; hit; ttl=%d; key=GET-http-localhost:9080-%s; detail=DEFAULT", expectedDuration, path) {
			t.Errorf(
				"unexpected second Cache-Status header %v, expected %s",
				resp1.Header.Get("Cache-Status"),
				fmt.Sprintf("Souin; hit; ttl=%d; key=GET-http-localhost:9080-%s; detail=DEFAULT", expectedDuration, path),
			)
		}
	}

	cacheChecker(caddyTester, "/expires-only", "Hello, expires-only!", int(time.Until(expiresValue).Seconds())-1)
	cacheChecker(caddyTester, "/expires-with-max-age", "Hello, expires-with-max-age!", 59)
	cacheChecker(caddyTester, "/expires-with-s-maxage", "Hello, expires-with-s-maxage!", 4)
}

func TestComplexQuery(t *testing.T) {
	caddyTester := caddytest.NewTester(t)
	caddyTester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		https_port    9443
		cache {
			ttl 10s
		}
	}
	localhost:9080 {
		route /complex-query {
			cache
			respond "Hello, {query}!"
		}
	}`, "caddyfile")

	cacheChecker := func(tester *caddytest.Tester, query string, expectedDuration int) {
		body := fmt.Sprintf("Hello, %s!", query)
		resp1, _ := tester.AssertGetResponse("http://localhost:9080/complex-query?"+query, 200, body)
		if resp1.Header.Get("Age") != "" {
			t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
		}

		if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/complex-query?"+query {
			t.Errorf("unexpected first Cache-Status header %v", resp1.Header.Get("Cache-Status"))
		}

		resp1, _ = tester.AssertGetResponse("http://localhost:9080/complex-query?"+query, 200, body)

		if resp1.Header.Get("Age") != "1" {
			t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
		}

		if resp1.Header.Get("Cache-Status") != fmt.Sprintf("Souin; hit; ttl=%d; key=GET-http-localhost:9080-/complex-query?%s; detail=DEFAULT", expectedDuration, query) {
			t.Errorf(
				"unexpected second Cache-Status header %v, expected %s",
				resp1.Header.Get("Cache-Status"),
				fmt.Sprintf("Souin; hit; ttl=%d; key=GET-http-localhost:9080-/complex-query?%s; detail=DEFAULT", expectedDuration, query),
			)
		}
	}

	cacheChecker(caddyTester, "fields[]=id&pagination=true", 9)
	cacheChecker(caddyTester, "fields[]=id&pagination=false", 9)
}

func TestBypassWithExpiresAndRevalidate(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		debug
		admin localhost:2999
		http_port 9080
		https_port 9443
		cache {
			ttl 5s
			stale 5s
			mode bypass
		}
	}
	localhost:9080 {
		route /bypass-with-expires-and-revalidate {
			cache
			header Expires 0
			header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate"
			respond "Hello, expires and revalidate!"
		}
	}`, "caddyfile")

	respStored1, _ := tester.AssertGetResponse(`http://localhost:9080/bypass-with-expires-and-revalidate`, 200, "Hello, expires and revalidate!")
	if respStored1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/bypass-with-expires-and-revalidate" {
		t.Errorf("unexpected Cache-Status header value %v", respStored1.Header.Get("Cache-Status"))
	}
	if respStored1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", respStored1.Header.Get("Age"))
	}

	respStored2, _ := tester.AssertGetResponse(`http://localhost:9080/bypass-with-expires-and-revalidate`, 200, "Hello, expires and revalidate!")
	if respStored2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/bypass-with-expires-and-revalidate; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header value %v", respStored2.Header.Get("Cache-Status"))
	}
	if respStored2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}

	time.Sleep(5 * time.Second)
	respStored3, _ := tester.AssertGetResponse(`http://localhost:9080/bypass-with-expires-and-revalidate`, 200, "Hello, expires and revalidate!")
	if respStored3.Header.Get("Cache-Status") != "Souin; hit; ttl=-1; key=GET-http-localhost:9080-/bypass-with-expires-and-revalidate; detail=DEFAULT; fwd=stale" {
		t.Errorf("unexpected Cache-Status header value %v", respStored3.Header.Get("Cache-Status"))
	}
	if respStored3.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}

	time.Sleep(5 * time.Second)
	respStored4, _ := tester.AssertGetResponse(`http://localhost:9080/bypass-with-expires-and-revalidate`, 200, "Hello, expires and revalidate!")
	if respStored4.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/bypass-with-expires-and-revalidate" {
		t.Errorf("unexpected Cache-Status header value %v", respStored4.Header.Get("Cache-Status"))
	}
	if respStored4.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", respStored4.Header.Get("Age"))
	}
}

func TestAllowedAdditionalStatusCode(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		debug
		admin localhost:2999
		http_port 9080
		https_port 9443
		cache {
			allowed_additional_status_codes 202 400
			ttl 5s
		}
	}
	localhost:9080 {
		route /bypass-with-expires-and-revalidate {
			cache
			respond "Hello, additional status code!"
		}
	}`, "caddyfile")

	respStored1, _ := tester.AssertGetResponse(`http://localhost:9080/bypass-with-expires-and-revalidate`, 200, "Hello, additional status code!")
	if respStored1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/bypass-with-expires-and-revalidate" {
		t.Errorf("unexpected Cache-Status header value %v", respStored1.Header.Get("Cache-Status"))
	}
	if respStored1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", respStored1.Header.Get("Age"))
	}

	respStored2, _ := tester.AssertGetResponse(`http://localhost:9080/bypass-with-expires-and-revalidate`, 200, "Hello, additional status code!")
	if respStored2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/bypass-with-expires-and-revalidate; detail=DEFAULT" {
		t.Errorf("unexpected Cache-Status header value %v", respStored2.Header.Get("Cache-Status"))
	}
	if respStored2.Header.Get("Age") == "" {
		t.Error("Age header should be present")
	}
}

type testTimeoutHandler struct {
	iterator int
}

func (t *testTimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.iterator++
	if t.iterator%2 == 0 {
		time.Sleep(5 * time.Second)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello timeout!"))
}

func TestTimeout(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		cache {
			ttl 1ns
			stale 1ns
			timeout {
				backend 1s
			}
		}
	}
	localhost:9080 {
		route /cache-timeout {
			cache
			reverse_proxy localhost:9086
		}
	}`, "caddyfile")

	go func() {
		errorHandler := testTimeoutHandler{}
		_ = http.ListenAndServe(":9086", &errorHandler)
	}()
	time.Sleep(time.Second)
	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/cache-timeout`, http.StatusOK, "Hello timeout!")
	time.Sleep(time.Millisecond)
	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-timeout`, http.StatusGatewayTimeout, "Internal server error")

	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-timeout" {
		t.Errorf("unexpected resp1 Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}

	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected resp1 Age header %v", resp1.Header.Get("Age"))
	}

	if resp2.Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=DEADLINE-EXCEEDED" {
		t.Errorf("unexpected resp2 Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}
}

type testSetCookieHandler struct{}

func (t *testSetCookieHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	fmt.Println("qjzkdqzkjdbqzd")
	w.Header().Set("Set-Cookie", "foo=bar")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello set-cookie!"))
}

func TestSetCookieNotStored(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		http_port     9080
		cache {
			ttl 5s
		}
	}
	localhost:9080 {
		route /cache-set-cookie {
			cache
			reverse_proxy localhost:9087 {
				header_down +Cache-Control no-cache=Set-Cookie
			}
		}
	}`, "caddyfile")

	go func() {
		setCookieHandler := testSetCookieHandler{}
		_ = http.ListenAndServe(":9087", &setCookieHandler)
	}()
	time.Sleep(time.Second)
	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/cache-set-cookie`, http.StatusOK, "Hello set-cookie!")
	time.Sleep(time.Millisecond)
	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-set-cookie`, http.StatusOK, "Hello set-cookie!")

	if resp1.Header.Get("Set-Cookie") != "foo=bar" {
		t.Errorf("unexpected resp1 Set-Cookie header %v", resp1.Header.Get("Set-Cookie"))
	}

	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/cache-set-cookie" {
		t.Errorf("unexpected resp1 Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}

	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected resp1 Age header %v", resp1.Header.Get("Age"))
	}

	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/cache-set-cookie; detail=DEFAULT" {
		t.Errorf("unexpected resp2 Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	if resp1.Header.Get("Set-Cookie") != "" {
		t.Errorf("unexpected resp2 Set-Cookie header %v", resp1.Header.Get("Set-Cookie"))
	}
}

func TestAPIPlatformInvalidation(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		debug
		admin localhost:2999
		http_port     9080
		cache {
			api {
				souin
			}
		}
	}
	localhost:9080 {
		route /api-platform-invalidation {
			cache

			header Vary "Content-Type"
			respond "Hello invalidation!"
		}
	}`, "caddyfile")

	reqSouinAPIList, _ := http.NewRequest(http.MethodGet, "http://localhost:2999/souin-api/souin", nil)
	reqSouinAPISK, _ := http.NewRequest(http.MethodGet, "http://localhost:2999/souin-api/souin/surrogate_keys", nil)

	_, _ = tester.AssertResponse(reqSouinAPIList, http.StatusOK, "[]")
	_, _ = tester.AssertResponse(reqSouinAPISK, http.StatusOK, "{}")
	_, _ = tester.AssertGetResponse("http://localhost:9080/api-platform-invalidation", http.StatusOK, "Hello invalidation!")
	resp4 := tester.AssertResponseCode(reqSouinAPIList, http.StatusOK)
	resp5 := tester.AssertResponseCode(reqSouinAPISK, http.StatusOK)

	var list []string
	_ = json.NewDecoder(resp4.Body).Decode(&list)

	if len(list) != 1 {
		t.Errorf("unexpected list %#v", list)
	}

	var items map[string]string
	_ = json.NewDecoder(resp5.Body).Decode(&items)

	if len(items) != 2 {
		t.Errorf("unexpected list %#v", items)
	}
}
