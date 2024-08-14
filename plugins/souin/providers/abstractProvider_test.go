package providers

import (
	"crypto/tls"
	"log"
	"testing"

	"github.com/darkweak/souin/plugins/souin/configuration"
	"github.com/darkweak/souin/tests"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

func mockConfiguration(configurationToLoad func() string) *configuration.Configuration {
	var config configuration.Configuration
	if e := yaml.Unmarshal([]byte(configurationToLoad()), &config); e != nil {
		log.Fatal(e)
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
	config.SetLogger(logger.Sugar())

	return &config
}

func TestInitProviders(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates: make([]tls.Certificate, 0),
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)
	InitProviders(config, &configChannel, mockConfiguration(tests.BaseConfiguration))
}

func TestCommonProvider_LoadFromConfigFile(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 1 != len(config.Certificates) {
		t.Errorf("Certificates length %d not corresponding to received %d", 1, len(config.Certificates))
	}
}

func TestCommonProvider_LoadFromConfigFile2(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates: make([]tls.Certificate, 0),
	}

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 0 != len(config.Certificates) {
		t.Errorf("Certificates length %d not corresponding to received %d", 0, len(config.Certificates))
	}
}

func TestCommonProvider_LoadFromConfigFile3(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates: make([]tls.Certificate, 0),
	}

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 0 != len(config.Certificates) {
		t.Errorf("Certificates length %d not corresponding to received %d", 0, len(config.Certificates))
	}
}

func TestCommonProvider_LoadFromConfigFile4(t *testing.T) {
	configChannel := make(chan int)
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config := &tls.Config{
		Certificates: []tls.Certificate{v},
	}

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 1 != len(config.Certificates) {
		t.Errorf("Certificates length %d not corresponding to received %d", 1, len(config.Certificates))
	}
}
