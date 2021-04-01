package rfc

import (
	"bytes"
	"fmt"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFreshness_Date(t *testing.T) {
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder()

	if 0 != getFreshness(res.Header(), r.Header) {
		errors.GenerateError(t, "Date shouldn't exist")
	}

	res.Header().Add("Date", "Mon, 08 Jan 2021 15:04:05 MST")

	if 0 != getFreshness(res.Header(), r.Header) {
		errors.GenerateError(t, fmt.Sprintf("%s", res.Header()))
		errors.GenerateError(t, "Date should exist")
	}
}

func TestGetFreshness_CacheControl(t *testing.T) {
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder()
	r.Header.Set("Cache-Control", "only-if-cached")

	if 1 != getFreshness(res.Header(), r.Header) {
		errors.GenerateError(t, "Freshness should be fresh if Response contains only-if-cached on Cache-Control header")
	}

	res.Header().Add("Cache-Control", "no-cache")

	if 0 != getFreshness(res.Header(), r.Header) {
		errors.GenerateError(t, "Freshness should be stale if Response contains no-cache on Cache-Control header")
	}

	r.Header.Set("Cache-Control", "no-cache")

	if 2 != getFreshness(res.Header(), r.Header) {
		errors.GenerateError(t, "Freshness should be transparent if Response contains no-cache on Cache-Control header")
	}
}

func TestCanStore(t *testing.T) {
	resCacheControl := make(map[string]string)
	reqCacheControl := make(map[string]string)

	if !canStore(reqCacheControl, resCacheControl) {
		errors.GenerateError(t, "Res and Req doesn't contains headers, it should return true")
	}

	reqCacheControl["no-store"] = "any"

	if canStore(reqCacheControl, resCacheControl) {
		errors.GenerateError(t, "Req contains headers, it should return false")
	}

	resCacheControl["no-store"] = "any"

	if canStore(reqCacheControl, resCacheControl) {
		errors.GenerateError(t, "Res contains headers, it should return false")
	}
}

func TestNewGatewayTimeoutResponse(t *testing.T) {
	if newGatewayTimeoutResponse(httptest.NewRequest("GET", "http://domain.com/testing", nil)).StatusCode != http.StatusGatewayTimeout {
		errors.GenerateError(t, "Status code should be 504 if valid request provided")
	}
}

func validateClonedRequest(t *testing.T, r *http.Request) {
	tmpReq := cloneRequest(r)

	if len(tmpReq.Header) != len(r.Header) {
		errors.GenerateError(t, fmt.Sprintf("Headers length should be equal, %d expected, %d provided", len(r.Header), len(tmpReq.Header)))
	}
	for k := range tmpReq.Header {
		if tmpReq.Header.Get(k) != r.Header.Get(k) {
			errors.GenerateError(t, fmt.Sprintf("Header %s should be equal to %s, %s provided", k, r.Header.Get(k), tmpReq.Header.Get(k)))
		}
	}
}

func TestCloneRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	validateClonedRequest(t, r)

	header := "Cache"
	r.Header.Set("Vary", header)
	r.Header.Set(header, "same")
	validateClonedRequest(t, r)
}

func TestGetEndToEndHeaders(t *testing.T) {
	res := httptest.NewRecorder().Result()
	e2eHeaders := getEndToEndHeaders(res.Header)

	if len(e2eHeaders) != 0 {
		errors.GenerateError(t, fmt.Sprintf("Headers should be equal to %v, %v provided", res.Header, e2eHeaders))
	}

	header := "Cache"
	res.Header.Set("Vary", header)
	res.Header.Set(header, "same")
	e2eHeaders = getEndToEndHeaders(res.Header)
	if len(e2eHeaders) != len(res.Header) {
		errors.GenerateError(t, fmt.Sprintf("Headers should be equal to %v, %v provided", res.Header, e2eHeaders))
	}

	res.Header.Set("Keep-Alive", "1")
	res.Header.Set("Connection", "keep-alive")
	e2eHeaders = getEndToEndHeaders(res.Header)
	if len(e2eHeaders) != len(res.Header)-2 {
		errors.GenerateError(t, fmt.Sprintf("Headers should be equal to %v, %v provided", res.Header, e2eHeaders))
	}
}

func verifyHeaderslength(t *testing.T, header http.Header, count int) {
	if len(parseCacheControl(header)) != count {
		errors.GenerateError(t, fmt.Sprintf("Cache control headers length should be equals to %d", count))
	}
}

func TestParseCacheControl(t *testing.T) {
	res := httptest.NewRecorder().Result()
	headers := res.Header

	headers.Add("Cache-Control", "no-cache")
	verifyHeaderslength(t, headers, 1)

	headers.Set("Cache-Control", "no-cache,only-if-cached")
	verifyHeaderslength(t, headers, 2)

	headers.Set("Cache-Control", "no-cache,,only-if-cached")
	verifyHeaderslength(t, headers, 2)

	headers.Set("Cache-Control", "no-cache, , only-if-cached")
	verifyHeaderslength(t, headers, 2)

	headers.Set("Cache-Control", "max-age=3")
	verifyHeaderslength(t, headers, 1)
}

func setCacheControlStaleOnHeader(h http.Header, value string) {
	h.Set("Cache-Control", fmt.Sprintf("stale-if-error=%s", value))
}

func TestCanStaleOnError_Req(t *testing.T) {
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()
	reqHeader := r.Header
	resHeader := res.Header
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, "It shouldn't stale")
	}

	setCacheControlStaleOnHeader(reqHeader, "notvalid")
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, "It shouldn't stale")
	}

	setCacheControlStaleOnHeader(reqHeader, "")
	if !canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, fmt.Sprintf("It should stale while testing %s", ""))
	}

	setCacheControlStaleOnHeader(reqHeader, "-3")
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, fmt.Sprintf("It should stale while testing %s", "-3"))
	}
}

func TestCanStaleOnError_Res(t *testing.T) {
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()
	reqHeader := r.Header
	resHeader := res.Header
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, "It shouldn't stale")
	}

	setCacheControlStaleOnHeader(resHeader, "notvalid")
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, "It shouldn't stale")
	}

	setCacheControlStaleOnHeader(resHeader, "")
	if !canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, fmt.Sprintf("It should stale while testing %s", ""))
	}

	setCacheControlStaleOnHeader(resHeader, "10000000")
	resHeader.Add("Date", "Mon, 08 Feb 2021 15:04:05 MST")
	if !canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, fmt.Sprintf("It should stale while testing %s", "10000000"))
	}

	setCacheControlStaleOnHeader(resHeader, "1")
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, fmt.Sprintf("It should stale while testing %s with Date header", "1"))
	}

	setCacheControlStaleOnHeader(resHeader, "-3")
	if canStaleOnError(resHeader, reqHeader) {
		errors.GenerateError(t, fmt.Sprintf("It should stale while testing %s", "-3"))
	}
}

func TestCachingReadCloser_Close(t *testing.T) {
	c := cachingReadCloser{}
	tests.ValidatePanic(t, func() {
		_ = c.Close()
	})
}

func TestCachingReadCloser_Read(t *testing.T) {
	c := cachingReadCloser{}
	tests.ValidatePanic(t, func() {
		_, _ = c.Read([]byte{})
	})

	b := []byte("Hello world")
	c = cachingReadCloser{
		R: ioutil.NopCloser(bytes.NewReader(b)),
	}

	res, err := c.Read(b)
	if err != nil {
		errors.GenerateError(t, "It shouldn't have error")
	}
	if res != len(b) {
		errors.GenerateError(t, "The result should have the same length as the byte array")
	}

	bu := new(bytes.Buffer)
	tests.ValidatePanic(t, func() {
		_, _ = c.Read(bu.Bytes())
	})
}
