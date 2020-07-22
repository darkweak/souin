package providers

import (
	"testing"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/errors"
	"fmt"
)

const PROVIDERSCOUNT = 2

func TestInitializeProviders(t *testing.T) {
	providers := InitializeProviders(configuration.GetConfig())
	if PROVIDERSCOUNT != len(*providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(*providers), PROVIDERSCOUNT))
	}

	conf := configuration.GetConfig()
	conf.Cache.Providers = []string{"memory"}
	providers = InitializeProviders(conf)
	if nil == providers {
		errors.GenerateError(t, "Impossible to retrieve providers list")
	}

	if 1 != len(*providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(*providers), PROVIDERSCOUNT))
	}

	conf.Cache.Providers = []string{"NotValid"}
	providers = InitializeProviders(conf)
	if 0 != len(*providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(*providers), PROVIDERSCOUNT))
	}

	conf.Cache.Providers = []string{}
	providers = InitializeProviders(conf)
	if PROVIDERSCOUNT != len(*providers) {
		errors.GenerateError(t, fmt.Sprintf("%v not corresponding to %v", len(*providers), PROVIDERSCOUNT))
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

func TestPathnameNotInRegex(t *testing.T) {
	if PathnameNotInRegex(configuration.GetConfig().Regex.Exclude, configuration.GetConfig()) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if PathnameNotInRegex(configuration.GetConfig().Regex.Exclude+"/A", configuration.GetConfig()) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if !PathnameNotInRegex("/BadPath", configuration.GetConfig()) {
		errors.GenerateError(t, "Pathname shouldn't be in regex")
	}
}
