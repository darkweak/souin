package coalescing

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"github.com/darkweak/souin/tests"
)

func commonInitializer() (*httptest.ResponseRecorder, *http.Request, *types.RetrieverResponseProperties) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	regexpUrls := helpers.InitializeRegexp(c)
	var excludedRegexp *regexp.Regexp = nil
	if c.GetDefaultCache().GetRegex().Exclude != "" {
		excludedRegexp = regexp.MustCompile(c.GetDefaultCache().GetRegex().Exclude)
	}
	retriever := &types.RetrieverResponseProperties{
		Configuration: c,
		Provider:      prs,
		MatchedURL:    tests.GetMatchedURL(tests.PATH),
		RegexpUrls:    regexpUrls,
		Transport:     rfc.NewTransport(prs, ykeys.InitializeYKeys(c.Ykeys), surrogate.InitializeSurrogate(c)),
		ExcludeRegex:  excludedRegexp,
	}
	r := httptest.NewRequest("GET", "http://"+tests.DOMAIN+tests.PATH, nil)
	w := httptest.NewRecorder()

	return w, r, retriever
}

func TestServeResponse(t *testing.T) {
	rc := Initialize()
	w, r, retriever := commonInitializer()
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
