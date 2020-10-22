package helpers

import (
	"github.com/darkweak/souin/configuration"
	"regexp"
)

// InitializeRegexp will generate one strong regex from your urls defined in the configuration.yml
func InitializeRegexp(configurationInstance configuration.AbstractConfigurationInterface) regexp.Regexp {
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
func PathnameNotInExcludeRegex(pathname string, configuration configuration.AbstractConfigurationInterface) bool {
	b, _ := regexp.Match(configuration.GetDefaultCache().Regex.Exclude, []byte(pathname))
	return !b
}
