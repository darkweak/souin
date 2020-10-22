package providers

import (
	"testing"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/helpers"
)

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
	if helpers.PathnameNotInExcludeRegex(configuration.GetConfiguration().GetDefaultCache().Regex.Exclude, configuration.GetConfiguration()) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if helpers.PathnameNotInExcludeRegex(configuration.GetConfiguration().GetDefaultCache().Regex.Exclude+"/A", configuration.GetConfiguration()) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if !helpers.PathnameNotInExcludeRegex("/BadPath", configuration.GetConfiguration()) {
		errors.GenerateError(t, "Pathname shouldn't be in regex")
	}
}
