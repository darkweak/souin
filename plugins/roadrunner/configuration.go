package roadrunner

import (
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/agnostic"
)

const (
	configurationKey = "http.cache"
)

// ParseConfiguration parse the Roadrunner configuration into a valid HTTP
// cache configuration object.
func parseConfiguration(cfg Configurer) middleware.BaseConfiguration {
	var configuration middleware.BaseConfiguration
	agnostic.ParseConfiguration(&configuration, cfg.Get(configurationKey).(map[string]interface{}))

	return configuration
}
