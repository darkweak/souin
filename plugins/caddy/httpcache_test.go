package httpcache

import (
	"net/http"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

func TestMinimal(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		order cache before rewrite
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
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=119; key=GET-http-localhost:9080-/cache-default" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(2 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/cache-default`, 200, "Hello, default!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=117; key=GET-http-localhost:9080-/cache-default" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
}

func TestQueryString(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		order cache before rewrite
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
		order cache before rewrite
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
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=59; key=GET-http-localhost:9080-/cache-max-age" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(2 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/cache-max-age`, 200, "Hello, max-age!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=57; key=GET-http-localhost:9080-/cache-max-age" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
}

func TestMaxStale(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		admin localhost:2999
		order cache before rewrite
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
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=2; key=GET-http-localhost:9080-/cache-max-stale" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(3 * time.Second)
	reqMaxStale, _ := http.NewRequest(http.MethodGet, maxStaleURL, nil)
	reqMaxStale.Header = http.Header{"Cache-Control": []string{"max-stale=3"}}
	resp3, _ := tester.AssertResponse(reqMaxStale, 200, "Hello, max-stale!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=-1; key=GET-http-localhost:9080-/cache-max-stale; fwd=stale" {
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
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/cache-s-maxage" {
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
	if respAuthBypassAlice2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/auth-bypass-Bearer Alice-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassAlice2.Header.Get("Cache-Status"))
	}

	respAuthBypassBob1, _ := tester.AssertResponse(getRequestFor("/auth-bypass", "Bob"), 200, "Hello, auth bypass Bearer Bob!")
	if respAuthBypassBob1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/auth-bypass-Bearer Bob-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassBob1.Header.Get("Cache-Status"))
	}
	respAuthBypassBob2, _ := tester.AssertResponse(getRequestFor("/auth-bypass", "Bob"), 200, "Hello, auth bypass Bearer Bob!")
	if respAuthBypassBob2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/auth-bypass-Bearer Bob-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthBypassBob2.Header.Get("Cache-Status"))
	}

	respAuthVaryBypassAlice1, _ := tester.AssertResponse(getRequestFor("/auth-bypass-vary", "Alice"), 200, "Hello, auth vary bypass Bearer Alice!")
	if respAuthVaryBypassAlice1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-localhost:9080-/auth-bypass-vary-Bearer Alice-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthVaryBypassAlice1.Header.Get("Cache-Status"))
	}
	respAuthVaryBypassAlice2, _ := tester.AssertResponse(getRequestFor("/auth-bypass-vary", "Alice"), 200, "Hello, auth vary bypass Bearer Alice!")
	if respAuthVaryBypassAlice2.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-localhost:9080-/auth-bypass-vary-Bearer Alice-text/plain" {
		t.Errorf("unexpected Cache-Status header %v", respAuthVaryBypassAlice2.Header.Get("Cache-Status"))
	}
}
