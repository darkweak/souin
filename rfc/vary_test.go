package rfc

import (
	"github.com/darkweak/souin/errors"
	"net/http/httptest"
	"testing"
)

func TestVaryMatches(t *testing.T) {
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if no header sent")
	}

	header := "Cache"
	r.Header.Set("Vary", header)
	r.Header.Set(header, "same")
	res.Header.Set("vary", header)
	res.Header.Set("X-Varied-"+header, "different")

	if varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return false if Response contains X-Varied-* header same than * in Request header")
	}
}
