package providers

import (
	"testing"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func cdnConfigurationAkamai() string {
	return `
default_cache:
  cdn:
    provider: akamai
    strategy: soft
    network: test
`
}

func cdnConfigurationFastly() string {
	return `
default_cache:
  cdn:
    provider: fastly
    strategy: soft
    api_key: test
`
}

func cdnConfigurationSouin() string {
	return `
default_cache:
  cdn:
    provider: default
    strategy: soft
    api_key: test
`
}

func mockConfiguration(configurationToLoad func() string) *configuration.Configuration {
	var config configuration.Configuration
	_ = config.Parse([]byte(configurationToLoad()))
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	logger, _ := cfg.Build()
	config.SetLogger(logger)

	return &config
}

func TestSurrogateFactory(t *testing.T) {
	akamaiConfiguration := mockConfiguration(cdnConfigurationAkamai)
	fastlyConfiguration := mockConfiguration(cdnConfigurationFastly)
	souinConfiguration := mockConfiguration(cdnConfigurationSouin)

	akamaiProvider := SurrogateFactory(akamaiConfiguration, "nuts")
	fastlyProvider := SurrogateFactory(fastlyConfiguration, "nuts")
	souinProvider := SurrogateFactory(souinConfiguration, "nuts")

	if akamaiProvider == nil {
		errors.GenerateError(t, "Impossible to create the Akamai surrogate provider instance")
	}
	if fastlyProvider == nil {
		errors.GenerateError(t, "Impossible to create the Fastly surrogate provider instance")
	}
	if souinProvider == nil {
		errors.GenerateError(t, "Impossible to create the Souin surrogate provider instance")
	}
}
