package tests

import (
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"testing"
)

// GetMatchedURL is an helper to mock the matchedURL
func GetMatchedURL(key string) configurationtypes.URL {
	config := MockConfiguration(BaseConfiguration)
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
		TTL:     config.GetDefaultCache().GetTTL(),
		Headers: config.GetDefaultCache().GetHeaders(),
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return matchedURL
}

// ValidatePanic is an helper to check if function should panic
func ValidatePanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			errors.GenerateError(t, "The code did not panic")
		}
	}()
	f()
}
