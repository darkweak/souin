package rfc

import (
	"net/http"
	"strconv"

	"github.com/pquerna/cachecontrol/cacheobject"
)

func validateMaxAgeCachedResponse(res *http.Response, maxAge int, addTime int) *http.Response {
	a, _ := strconv.Atoi(res.Header.Get("Age"))

	if maxAge >= 0 && (maxAge+addTime) < a {
		return nil
	}

	return res
}

func ValidateMaxAgeCachedResponse(co *cacheobject.RequestCacheDirectives, res *http.Response) *http.Response {
	return validateMaxAgeCachedResponse(res, int(co.MaxAge), 0)
}

func ValidateMaxAgeCachedStaleResponse(co *cacheobject.RequestCacheDirectives, res *http.Response, addTime int) *http.Response {
	if !co.MaxStaleSet && co.MaxStale < 0 {
		return nil
	}

	return validateMaxAgeCachedResponse(res, int(co.MaxStale), addTime)
}
