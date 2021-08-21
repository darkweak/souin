package helpers

import (
	"regexp"

	configurationtypes "github.com/darkweak/souin/configurationtypes"
)

// InitializeRegexp will generate one strong regex from your urls defined in the configuration.yml
func InitializeRegexp(configurationInstance configurationtypes.AbstractConfigurationInterface) regexp.Regexp {
	u := ""
	for k := range configurationInstance.GetUrls() {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

// PathnameNotInExcludeRegex check if pathname is in parameter regex var
func PathnameNotInExcludeRegex(pathname string, configuration configurationtypes.AbstractConfigurationInterface) bool {
	b, _ := regexp.Match(configuration.GetDefaultCache().GetRegex().Exclude, []byte(pathname))
	return !b
}
