package configurationtypes

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"
)

func cacheKeysToMap(c CacheKeys) map[string]Key {
	m := map[string]Key{}
	for _, cacheKey := range c {
		for rg, key := range cacheKey {
			m[rg.String()] = key
		}
	}

	return m
}

// Regression test for https://github.com/darkweak/souin/issues/688.
func TestCacheKeysJSONRoundTrip(t *testing.T) {
	keys := CacheKeys{
		{RegValue{Regexp: regexp.MustCompile(`.+\/404$`)}: {DisableHost: true}},
		{RegValue{Regexp: regexp.MustCompile(`.+\.css`)}: {DisableBody: true, DisableQuery: true, Headers: []string{"Authorization"}}},
		{RegValue{Regexp: regexp.MustCompile(`"quoted"`)}: {Template: "{http.request.host}"}},
		{RegValue{Regexp: regexp.MustCompile(`empty`)}: {}},
	}

	raw, err := json.Marshal(&keys)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded CacheKeys
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(cacheKeysToMap(keys), cacheKeysToMap(decoded)) {
		t.Errorf("unexpected round trip result from %s: expected %#v, got %#v", raw, cacheKeysToMap(keys), cacheKeysToMap(decoded))
	}
}

func TestCacheKeysUnmarshalJSON(t *testing.T) {
	var decoded CacheKeys
	if err := json.Unmarshal([]byte(`{".+\\/404$":{"disable_host":true},"other":{"disable_query":true}}`), &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	expected := map[string]Key{
		`.+\/404$`: {DisableHost: true},
		`other`:    {DisableQuery: true},
	}
	if !reflect.DeepEqual(expected, cacheKeysToMap(decoded)) {
		t.Errorf("expected %#v, got %#v", expected, cacheKeysToMap(decoded))
	}

	var empty CacheKeys
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil || len(empty) != 0 {
		t.Errorf("expected no cache keys, got %#v (err %v)", empty, err)
	}
}
