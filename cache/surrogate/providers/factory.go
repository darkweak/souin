package providers

import (
	"github.com/darkweak/souin/configurationtypes"
)

// SurrogateFactory generate a SurrogateInterface instance
func SurrogateFactory(config configurationtypes.AbstractConfigurationInterface) SurrogateInterface {
	cdn := config.GetDefaultCache().GetCDN()

	switch cdn.Provider {
	case "akamai":
		return generateAkamaiInstance(config)
	case "cloudflare":
		return generateCloudflareInstance(config)
	case "fastly":
		return generateFastlyInstance(config)
	default:
		return generateSouinInstance(config)
	}
}
