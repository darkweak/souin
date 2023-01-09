package providers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"

func verifyNewValueAfterSet(client types.AbstractProviderInterface, key string, value []byte, t *testing.T) {
	newValue := client.Get(key)

	if len(newValue) != len(value) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should be equals to %s, %s provided", key, value, newValue))
	}
}

func setValueThenVerify(client types.AbstractProviderInterface, key string, value []byte, matchedURL configurationtypes.URL, ttl time.Duration, t *testing.T) {
	_ = client.Set(key, value, matchedURL, ttl)
	time.Sleep(1 * time.Second)
	verifyNewValueAfterSet(client, key, value, t)
}

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	p := InitializeProvider(c)
	err := p.Init()
	if nil != err {
		errors.GenerateError(t, "Init shouldn't crash")
	}
}

func TestVaryVoter(t *testing.T) {
	if !varyVoter("myBaseKey", nil, "myBaseKey") {
		errors.GenerateError(t, "The vary voter must return true when both keys are equal")
	}

	rq := http.Request{
		Header: http.Header{
			"X-Value-Test": []string{"something-valid"},
		},
	}

	varyResponse1 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", VarySeparator, "X-Value-Test:something-valid"))

	rq.Header = http.Header{
		"X-Value-Test":        []string{"something-valid"},
		"X-Value-Test-Second": []string{"another-valid"},
	}
	varyResponse2 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", VarySeparator, "X-Value-Test:something-valid;X-Value-Test-Second:another-valid"))

	rq.Header = http.Header{
		"X-Value-Test":        []string{"something-valid"},
		"X-Value-Test-Second": []string{"another-valid"},
	}
	varyResponse3 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", VarySeparator, "X-Value-Test:something-valid;X-Value-Test-Second:another-valid;X-Value-Test-Third:"))

	rq.Header = http.Header{
		"X-Value-Test":        []string{"something-invalid"},
		"X-Value-Test-Second": []string{"another-valid"},
	}
	varyResponse4 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", VarySeparator, "X-Value-Test:something-valid;X-Value-Test-Second:another-valid;X-Value-Test-Third:"))

	rq.Header = http.Header{
		"X-Value-Test": []string{"something-valid"},
		"X-With-Comma": []string{"first; directive"},
	}
	varyResponse5 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", VarySeparator, "X-Value-Test:something-valid;X-With-Comma:first%3B%20directive"))

	rq.Header = http.Header{
		"X-Value-Test": []string{"something-valid"},
		"X-With-Comma": []string{"first:directive"},
	}
	varyResponse6 := varyVoter("baseKey", &rq, fmt.Sprintf("baseKey%s%s", VarySeparator, "X-Value-Test:something-valid;X-With-Comma:first%3Adirective"))

	if !(varyResponse1 && varyResponse2 && varyResponse3 && varyResponse5 && varyResponse6) || varyResponse4 {
		errors.GenerateError(t, "The varyVoter must match the expected")
	}
}
