package providers

import (
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	yaml "gopkg.in/yaml.v3"
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

type testConfiguration struct {
	defaultCache *configurationtypes.DefaultCache `yaml:"default_cache"`
}

func (*testConfiguration) GetUrls() map[string]configurationtypes.URL {
	return nil
}
func (*testConfiguration) GetPluginName() string {
	return ""
}
func (t *testConfiguration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return t.defaultCache
}
func (*testConfiguration) GetAPI() configurationtypes.API {
	return configurationtypes.API{}
}
func (*testConfiguration) GetLogLevel() string {
	return ""
}
func (*testConfiguration) GetLogger() *zap.Logger {
	return zap.NewNop()
}
func (*testConfiguration) SetLogger(*zap.Logger) {
	return
}
func (*testConfiguration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}
func (*testConfiguration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}
func (t *testConfiguration) GetCacheKeys() configurationtypes.CacheKeys {
	return nil
}

func mockConfiguration(configurationToLoad func() string) *testConfiguration {
	var config testConfiguration
	if err := yaml.Unmarshal([]byte(configurationToLoad()), &config); err != nil {
		return nil
	}
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
		t.Error("Impossible to create the Akamai surrogate provider instance")
	}
	if fastlyProvider == nil {
		t.Error("Impossible to create the Fastly surrogate provider instance")
	}
	if souinProvider == nil {
		t.Error("Impossible to create the Souin surrogate provider instance")
	}
}
