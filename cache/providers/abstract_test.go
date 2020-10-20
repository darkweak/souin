package providers

import (
	"testing"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/errors"
	"regexp"
)

const PROVIDERSCOUNT = 3

func MockInitializeRegexp(configurationInstance configuration.Configuration) regexp.Regexp {
	u := ""
	for k := range configurationInstance.URLs {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

func TestContainsString(t *testing.T) {
	a := []string{"a", "b", "c"}
	if !contains(a, "a") {
		errors.GenerateError(t, "a character should exist")
	}

	if contains(a, "x") {
		errors.GenerateError(t, "x character shouldn't exist")
	}
}

func TestPathnameNotInExcludeRegex(t *testing.T) {
	if PathnameNotInExcludeRegex(configuration.GetConfig().DefaultCache.Regex.Exclude, configuration.GetConfig()) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if PathnameNotInExcludeRegex(configuration.GetConfig().DefaultCache.Regex.Exclude+"/A", configuration.GetConfig()) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if !PathnameNotInExcludeRegex("/BadPath", configuration.GetConfig()) {
		errors.GenerateError(t, "Pathname shouldn't be in regex")
	}
}
