package coalescing

import (
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"github.com/darkweak/souin/tests"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeResponse(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	regexpUrls := helpers.InitializeRegexp(c)
	prs := providers.InitializeProvider(c)
	defer prs["olric"].Reset()
	rc := Initialize()
	retriever := &types.RetrieverResponseProperties{
		Configuration: c,
		Providers:     prs,
		MatchedURL:    tests.GetMatchedURL(tests.PATH),
		RegexpUrls:    regexpUrls,
		Transport:     rfc.NewTransport(prs),
	}
	r := httptest.NewRequest("GET", "http://"+tests.DOMAIN+tests.PATH, nil)
	w := httptest.NewRecorder()

	ServeResponse(
		w,
		r,
		retriever,
		func(
			http.ResponseWriter,
			*http.Request,
			types.RetrieverResponsePropertiesInterface,
			RequestCoalescingInterface,
			func(http.ResponseWriter, *http.Request) error,
		) {
		},
		rc,
		func(http.ResponseWriter, *http.Request) error {
			return nil
		},
	)
}
