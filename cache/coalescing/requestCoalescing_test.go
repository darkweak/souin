package coalescing

//func TestServeResponse(t *testing.T) {
//	rc := Initialize()
//	c := tests.MockConfiguration()
//	prs := providers.InitializeProvider(c)
//	regexpUrls := helpers.InitializeRegexp(c)
//	retriever := &types.RetrieverResponseProperties{
//		Configuration: c,
//		Providers:      prs,
//		MatchedURL:    tests.GetMatchedURL(tests.PATH),
//		RegexpUrls:    regexpUrls,
//	}
//	r := httptest.NewRequest("GET", "http://"+tests.DOMAIN+tests.PATH, nil)
//	w := httptest.NewRecorder()
//
//	ServeResponse(
//		w,
//		r,
//		retriever,
//		func(
//			rw http.ResponseWriter,
//			rq *http.Request,
//			r types.RetrieverResponsePropertiesInterface,
//			rc RequestCoalescingInterface){},
//		rc,
//	)
//}
