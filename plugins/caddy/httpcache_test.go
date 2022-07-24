package httpcache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

func TestMaxAge(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
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
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-max-age`, 200, "Hello, max-age!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=59" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(2 * time.Second)
	resp3, _ := tester.AssertGetResponse(`http://localhost:9080/cache-max-age`, 200, "Hello, max-age!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=57" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}
}

func TestMaxStale(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		order cache before rewrite
		http_port     9080
		https_port    9443
		cache
	}
	localhost:9080 {
		route /cache-max-stale {
			cache {
				stale 5
			}
			header Cache-Control "max-age=3"
			respond "Hello, max-stale!"
		}
	}`, "caddyfile")

	maxStaleURL := "http://localhost:9080/cache-max-stale"

	resp1, _ := tester.AssertGetResponse(maxStaleURL, 200, "Hello, max-stale!")
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header)
	}

	resp2, _ := tester.AssertGetResponse(maxStaleURL, 200, "Hello, max-stale!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=2" {
		t.Errorf("unexpected Cache-Status header %v", resp2.Header.Get("Cache-Status"))
	}

	time.Sleep(3 * time.Second)
	reqMaxStale := httptest.NewRequest(http.MethodGet, maxStaleURL, nil)
	reqMaxStale.Header = http.Header{"Cache-Control": []string{"max-stale=3"}}
	resp3, _ := tester.AssertResponse(reqMaxStale, 200, "Hello, max-stale!")
	if resp3.Header.Get("Cache-Status") != "Souin; hit; ttl=-1; fwd=stale" {
		t.Errorf("unexpected Cache-Status header %v", resp3.Header.Get("Cache-Status"))
	}

	time.Sleep(3 * time.Second)
	resp4, _ := tester.AssertResponse(reqMaxStale, 200, "Hello, max-stale!")
	if resp4.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Errorf("unexpected Cache-Status header %v", resp4.Header.Get("Cache-Status"))
	}
}

func TestSMaxAge(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
	  http_port     9080
	  https_port    9443
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
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Errorf("unexpected Cache-Status header %v", resp1.Header.Get("Cache-Status"))
	}

	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/cache-s-maxage`, 200, "Hello, s-maxage!")
	if resp2.Header.Get("Cache-Status") != "Souin; hit; ttl=4" {
		t.Errorf("unexpected Cache-Status header with %v", resp2.Header.Get("Cache-Status"))
	}
}

func TestAgeHeader(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
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

func TestExistingAgeHeader(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
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
			header Age "max-age=5"
			respond "Hello, Age header!"
		}
	}`, "caddyfile")

	resp1, _ := tester.AssertGetResponse(`http://localhost:9080/age-header`, 200, "Hello, Age header!")
	if resp1.Header.Get("Age") != "" {
		t.Errorf("unexpected Age header %v", resp1.Header.Get("Age"))
	}
	resp2, _ := tester.AssertGetResponse(`http://localhost:9080/age-header`, 200, "Hello, Age header!")
	if resp2.Header.Get("Age") != "1" {
		t.Errorf("unexpected Age header value %v", resp2.Header.Get("Age"))
	}
}

func TestNotHandledRoute(t *testing.T) {
	tester := caddytest.NewTester(t)
	tester.InitServer(`
	{
		http_port     9080
		https_port    9443
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
	if resp1.Header.Get("Cache-Status") != "Souin; fwd=uri-miss" {
		t.Errorf("unexpected Cache-Status header value %v", resp1.Header.Get("Cache-Status"))
	}
}
