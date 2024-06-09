package storage

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/tests"
)

const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"
const BASE_VALUE = "My first data"

func verifyNewValueAfterSet(client types.Storer, key string, value []byte, t *testing.T) {
	newValue := client.Get(key)

	if len(newValue) != len(value) {
		t.Errorf("Key %s should be equals to %s, %s provided", key, value, newValue)
	}
}

func setValueThenVerify(client types.Storer, key string, value []byte, matchedURL configurationtypes.URL, ttl time.Duration, t *testing.T) {
	_ = client.Set(key, value, matchedURL, ttl)
	time.Sleep(1 * time.Second)
	verifyNewValueAfterSet(client, key, value, t)
}

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	storer, err := NewStorage(c)
	if nil != err {
		t.Error("NewStorage should return a new storer")
	}
	if storer.Init() != err {
		t.Error("Init shouldn't crash")
	}
}

func TestVaryVoter(t *testing.T) {
	if !varyVoter("myBaseKey", nil, "myBaseKey") {
		t.Error("The vary voter must return true when both keys are equal")
	}

	rq := http.Request{
		Header: http.Header{
			"X-Value-Test": []string{"something-valid"},
		},
	}

	varyResponse1 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", rfc.VarySeparator, "X-Value-Test:something-valid"))

	rq.Header = http.Header{
		"X-Value-Test":        []string{"something-valid"},
		"X-Value-Test-Second": []string{"another-valid"},
	}
	varyResponse2 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", rfc.VarySeparator, "X-Value-Test:something-valid;X-Value-Test-Second:another-valid"))

	rq.Header = http.Header{
		"X-Value-Test":        []string{"something-valid"},
		"X-Value-Test-Second": []string{"another-valid"},
	}
	varyResponse3 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", rfc.VarySeparator, "X-Value-Test:something-valid;X-Value-Test-Second:another-valid;X-Value-Test-Third:"))

	rq.Header = http.Header{
		"X-Value-Test":        []string{"something-invalid"},
		"X-Value-Test-Second": []string{"another-valid"},
	}
	varyResponse4 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", rfc.VarySeparator, "X-Value-Test:something-valid;X-Value-Test-Second:another-valid;X-Value-Test-Third:"))

	rq.Header = http.Header{
		"X-Value-Test": []string{"something-valid"},
		"X-With-Comma": []string{"first; directive"},
	}
	varyResponse5 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", rfc.VarySeparator, "X-Value-Test:something-valid;X-With-Comma:first%3B%20directive"))

	rq.Header = http.Header{
		"X-Value-Test": []string{"something-valid"},
		"X-With-Comma": []string{"first:directive"},
	}
	varyResponse6 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", rfc.VarySeparator, "X-Value-Test:something-valid;X-With-Comma:first%3Adirective"))

	if !(varyResponse1 && varyResponse2 && varyResponse3 && varyResponse5 && varyResponse6) || varyResponse4 {
		t.Error("The varyVoter must match the expected")
	}
}

// GetCacheProviderClientAndMatchedURL will work as a factory to build providers from configuration and get the URL from the key passed in parameter
func GetCacheProviderClientAndMatchedURL(key string, configurationMocker func() configurationtypes.AbstractConfigurationInterface, factory func(configurationInterface configurationtypes.AbstractConfigurationInterface) (types.Storer, error)) (types.Storer, configurationtypes.URL) {
	config := configurationMocker()
	client, _ := factory(config)

	u := ""
	for k := range config.GetUrls() {
		if u != "" {
			u += "|"
		}
		u += "(" + k + ")"
	}

	regexpUrls := *regexp.MustCompile(u)

	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
		TTL:     configurationtypes.Duration{Duration: config.GetDefaultCache().GetTTL()},
		Headers: config.GetDefaultCache().GetHeaders(),
	}
	if regexpURL != "" {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return client, matchedURL
}
