package storage

/*
func mockEmbeddedConfiguration(c func() string, key string) (types.Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(c)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
			provider, _ := EmbeddedOlricConnectionFactory(config)

			return provider, nil
		},
	)
}

func getEmbeddedOlricClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return mockEmbeddedConfiguration(tests.EmbeddedOlricConfiguration, key)
}

func getEmbeddedOlricWithoutYAML(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return mockEmbeddedConfiguration(tests.EmbeddedOlricPlainConfigurationWithoutAdditionalYAML, key)
}

func TestIShouldBeAbleToReadAndWriteDataInEmbeddedOlric(t *testing.T) {
	client, u := getEmbeddedOlricClientAndMatchedURL("Test")
	_ = client.Set("Test", []byte(BASE_VALUE), u, time.Duration(10)*time.Second)
	res := client.Get("Test")
	if BASE_VALUE != string(res) {
		t.Errorf("%s not corresponding to %s", res, BASE_VALUE)
	}
	_ = client.Reset()
}

func TestIShouldBeAbleToReadAndWriteDataInEmbeddedOlricWithoutYAML(t *testing.T) {
	client, u := getEmbeddedOlricWithoutYAML("Test_without")
	_ = client.Set("Test", []byte(BASE_VALUE), u, time.Duration(10)*time.Second)
	time.Sleep(3 * time.Second)
	res := client.Get("Test")
	if BASE_VALUE != string(res) {
		t.Errorf("%s not corresponding to %s", res, BASE_VALUE)
	}
	_ = client.Reset()
}

func TestEmbeddedOlric_GetRequestInCache(t *testing.T) {
	client, _ := getEmbeddedOlricClientAndMatchedURL(NONEXISTENTKEY)
	res := client.Get(NONEXISTENTKEY)
	if string(res) != "" {
		t.Errorf("Key %s should not exist", NONEXISTENTKEY)
	}
	_ = client.Reset()
}

func TestEmbeddedOlric_SetRequestInCache_OneByte(t *testing.T) {
	client, u := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	_ = client.Set(BYTEKEY, []byte{65}, u, time.Duration(20)*time.Second)
	_ = client.Reset()
}

func TestEmbeddedOlric_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getEmbeddedOlricClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
	_ = client.Reset()
}

func TestEmbeddedOlric_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
	_ = client.Reset()
}

func TestEmbeddedOlric_DeleteRequestInCache(t *testing.T) {
	client, _ := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		t.Errorf("Key %s should not exist", BYTEKEY)
	}
	_ = client.Reset()
}

func TestEmbeddedOlric_Init(t *testing.T) {
	client, _ := EmbeddedOlricConnectionFactory(tests.MockConfiguration(tests.EmbeddedOlricConfiguration))
	err := client.Init()

	if nil != err {
		t.Error("Impossible to init EmbeddedOlric provider")
	}
	_ = client.Reset()
}
*/
