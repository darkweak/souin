package providers

import (
	"testing"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/errors"
	"fmt"
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

func TestInitializeProviders(t *testing.T) {
	providers := InitializeProviders(configuration.GetConfig())
	if PROVIDERSCOUNT != len(providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(providers), PROVIDERSCOUNT))
	}

	conf := configuration.GetConfig()
	conf.DefaultCache.Providers = []string{"memory"}
	providers = InitializeProviders(conf)
	if nil == providers {
		errors.GenerateError(t, "Impossible to retrieve providers list")
	}

	if 1 != len(providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(providers), 1))
	}

	conf.DefaultCache.Providers = []string{"NotValid"}
	providers = InitializeProviders(conf)
	if 0 != len(providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(providers), 0))
	}

	conf.DefaultCache.Providers = []string{}
	providers = InitializeProviders(conf)
	if PROVIDERSCOUNT != len(providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(providers), PROVIDERSCOUNT))
	}
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
