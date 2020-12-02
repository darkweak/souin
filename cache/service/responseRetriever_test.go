package service

import (
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/tests"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeResponse(t *testing.T) {
	c := tests.MockConfiguration()
	prs := providers.InitializeProvider(c)
	regexpUrls := helpers.InitializeRegexp(c)
	retriever := &types.RetrieverResponseProperties{
		Configuration: c,
		Providers:     prs,
		MatchedURL:    tests.GetMatchedURL(tests.PATH),
		RegexpUrls:    regexpUrls,
	}
	r := httptest.NewRequest("GET", "http://"+tests.DOMAIN+tests.PATH, nil)
	w := httptest.NewRecorder()
	ServeResponse(
		w,
		r,
		retriever,
		func(rw http.ResponseWriter, rq *http.Request, r types.RetrieverResponsePropertiesInterface, key string) {},
	)
}
