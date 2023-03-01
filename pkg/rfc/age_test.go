package rfc

import (
	"net/http"
	"testing"

	"github.com/pquerna/cachecontrol/cacheobject"
)

func Test_ValidateMaxAgeCachedResponse(t *testing.T) {
	coWithoutMaxAge := cacheobject.RequestCacheDirectives{
		MaxAge: -1,
	}
	coWithMaxAge := cacheobject.RequestCacheDirectives{
		MaxAge: 10,
	}

	expiredMaxAge := http.Response{
		Header: http.Header{
			"Age": []string{"11"},
		},
	}
	validMaxAge := http.Response{
		Header: http.Header{
			"Age": []string{"9"},
		},
	}

	if ValidateMaxAgeCachedResponse(&coWithoutMaxAge, &expiredMaxAge) == nil {
		t.Errorf("The max-age validation should return the response instead of nil with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithoutMaxAge, expiredMaxAge)
	}
	if ValidateMaxAgeCachedResponse(&coWithoutMaxAge, &validMaxAge) == nil {
		t.Errorf("The max-age validation should return the response instead of nil with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithoutMaxAge, validMaxAge)
	}

	if ValidateMaxAgeCachedResponse(&coWithMaxAge, &expiredMaxAge) != nil {
		t.Errorf("The max-age validation should return nil instead of the response with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithMaxAge, expiredMaxAge)
	}
	if ValidateMaxAgeCachedResponse(&coWithMaxAge, &validMaxAge) == nil {
		t.Errorf("The max-age validation should return the response instead of nil with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithMaxAge, expiredMaxAge)
	}
}

func Test_ValidateMaxStaleCachedResponse(t *testing.T) {
	coWithoutMaxStale := cacheobject.RequestCacheDirectives{
		MaxStale: -1,
	}
	coWithMaxStale := cacheobject.RequestCacheDirectives{
		MaxStale:    10,
		MaxStaleSet: true,
	}

	expiredMaxAge := http.Response{
		Header: http.Header{
			"Age": []string{"14"},
		},
	}
	validMaxAge := http.Response{
		Header: http.Header{
			"Age": []string{"12"},
		},
	}

	if ValidateMaxAgeCachedStaleResponse(&coWithoutMaxStale, &expiredMaxAge, 3) != nil {
		t.Errorf("The max-stale validation should return nil instead of the response with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithoutMaxStale, expiredMaxAge)
	}
	if ValidateMaxAgeCachedStaleResponse(&coWithoutMaxStale, &validMaxAge, 14) != nil {
		t.Errorf("The max-stale validation should return the response instead of nil with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithoutMaxStale, validMaxAge)
	}

	if ValidateMaxAgeCachedStaleResponse(&coWithMaxStale, &expiredMaxAge, 0) != nil {
		t.Errorf("The max-stale validation should return nil instead of the response with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithMaxStale, expiredMaxAge)
	}
	if ValidateMaxAgeCachedStaleResponse(&coWithMaxStale, &validMaxAge, 5) == nil {
		t.Errorf("The max-stale validation should return the response instead of nil with the given parameters:\nRequestCacheDirectives: %+v\nResponse: %+v\n", coWithMaxStale, expiredMaxAge)
	}
}
