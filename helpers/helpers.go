package helpers

import (
	configuration_types "github.com/darkweak/souin/configuration_types"
	"regexp"
)

// InitializeRegexp will generate one strong regex from your urls defined in the configuration.yml
func InitializeRegexp(configurationInstance configuration_types.AbstractConfigurationInterface) regexp.Regexp {
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
func PathnameNotInExcludeRegex(pathname string, configuration configuration_types.AbstractConfigurationInterface) bool {
	b, _ := regexp.Match(configuration.GetDefaultCache().Regex.Exclude, []byte(pathname))
	return !b
}
