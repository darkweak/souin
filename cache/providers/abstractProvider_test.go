package providers

import (
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/tests"
	"testing"
)

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration()
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
	config := tests.MockConfiguration()
	if helpers.PathnameNotInExcludeRegex(config.GetDefaultCache().Regex.Exclude, config) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if helpers.PathnameNotInExcludeRegex(config.GetDefaultCache().Regex.Exclude+"/A", config) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if !helpers.PathnameNotInExcludeRegex("/BadPath", config) {
		errors.GenerateError(t, "Pathname shouldn't be in regex")
	}
}
