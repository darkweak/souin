package tests

import (
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"testing"
)

func GetMatchedURL(key string) configurationtypes.URL {
	config := MockConfiguration()
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
		TTL:     config.GetDefaultCache().TTL,
		Headers: config.GetDefaultCache().Headers,
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return matchedURL
}

func ValidatePanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			errors.GenerateError(t, "The code did not panic")
		}
	}()
	f()
}
