package providers

import (
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/tests"
	"testing"
)

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	ps := InitializeProvider(c)
	defer ps["olric"].Reset()
	for k, p := range ps {
		if k != "olric" {
			err := p.Init()
			if nil != err {
				errors.GenerateError(t, "Init shouldn't crash")
			}
		}
	}
}

func TestPathnameNotInExcludeRegex(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	if helpers.PathnameNotInExcludeRegex(config.GetDefaultCache().GetRegex().Exclude, config) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if helpers.PathnameNotInExcludeRegex(config.GetDefaultCache().GetRegex().Exclude+"/A", config) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if !helpers.PathnameNotInExcludeRegex("/BadPath", config) {
		errors.GenerateError(t, "Pathname shouldn't be in regex")
	}
}
