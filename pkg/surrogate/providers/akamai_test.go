package providers

/*
func mockAkamaiProvider() *AkamaiSurrogateStorage {
	instanciator, _ := storage.NewStorageFromName("badger")
	storer, _ := instanciator(tests.MockConfiguration(tests.BadgerConfiguration))
	ass := &AkamaiSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    storer,
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
			logger:     zap.NewNop(),
		},
		url: "http://akamai/invalidate_tag",
	}

	ass.baseStorage.parent = ass

	return ass
}

func TestAkamaiSurrogateStorage_Purge(t *testing.T) {
	ap := mockAkamaiProvider()
	headerMock := http.Header{}
	headerMock.Set(surrogateKey, baseHeaderValue)

	cacheKeys1, surrogateKeys1 := ap.baseStorage.Purge(headerMock)
	cacheKeys2, surrogateKeys2 := ap.Purge(headerMock)

	if len(cacheKeys1) != len(cacheKeys2) {
		t.Error("The cache keys length should match.")
	}
	if len(surrogateKeys1) != len(surrogateKeys2) {
		t.Error("The surrogate keys length should match.")
	}

	for i, key := range cacheKeys1 {
		if key != cacheKeys2[i] {
			t.Errorf("The cache key %d should match %s, %s given.", i, key, cacheKeys2[i])
		}
	}
	for i, key := range surrogateKeys1 {
		if key != surrogateKeys2[i] {
			t.Errorf("The surrogate key %d should match %s, %s given.", i, key, surrogateKeys2[i])
		}
	}
}

func TestAkamaiSurrogateStorage_Store(t *testing.T) {
	ap := mockAkamaiProvider()
	res := http.Response{
		Header: http.Header{},
	}
	res.Header.Set(surrogateKey, baseHeaderValue)

	var e error
	if e = ap.Store(&res, "stored"); e != nil {
		t.Error("It should not throw an error while store.")
	}

	edgeTagValue := res.Header.Get(edgeCacheTag)
	if edgeTagValue != baseHeaderValue {
		t.Errorf("EdgeTag should match %s, %s given.", baseHeaderValue, edgeTagValue)
	}

	if res.Header.Get(surrogateKey) != "" {
		t.Error("The response should not contains the Surrogate keys header.")
	}

	if res.Header.Get(surrogateControl) != "" {
		t.Error("The response should not contains the Surrogate control header.")
	}
}
*/
