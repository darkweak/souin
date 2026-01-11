package providers

import (
	"github.com/darkweak/souin/configurationtypes"
	"time"
)

// SurrogateFactory generate a SurrogateInterface instance
func SurrogateFactory(config configurationtypes.AbstractConfigurationInterface, defaultStorerName string, defaultTTL time.Duration) SurrogateInterface {
	cdn := config.GetDefaultCache().GetCDN()

	switch cdn.Provider {
	case "akamai":
		return generateAkamaiInstance(config, defaultStorerName, defaultTTL)
	case "cloudflare":
		return generateCloudflareInstance(config, defaultStorerName, defaultTTL)
	case "fastly":
		return generateFastlyInstance(config, defaultStorerName, defaultTTL)
	default:
		return generateSouinInstance(config, defaultStorerName, defaultTTL)
	}
}
