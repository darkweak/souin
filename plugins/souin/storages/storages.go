package storages

import (
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/storages/badger"
	"github.com/darkweak/storages/core"
	"github.com/darkweak/storages/etcd"
	"github.com/darkweak/storages/nats"
	"github.com/darkweak/storages/nuts"
	"github.com/darkweak/storages/olric"
	"github.com/darkweak/storages/otter"
	"github.com/darkweak/storages/redis"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type factory func(providerConfiguration core.CacheProvider, logger *zap.Logger, stale time.Duration) (core.Storer, error)

func isProviderEmpty(p configurationtypes.CacheProvider) bool {
	return p.Configuration == nil && p.Path == "" && p.URL == ""
}
func toCoreCacheProvider(p configurationtypes.CacheProvider) core.CacheProvider {
	return core.CacheProvider{
		Configuration: p.Configuration,
		Path:          p.Path,
		URL:           p.URL,
	}
}
func tryToRegisterStorage(p configurationtypes.CacheProvider, f factory, logger *zap.Logger, stale time.Duration) {
	if !isProviderEmpty(p) {
		if s, err := f(toCoreCacheProvider(p), logger, stale); err == nil {
			core.RegisterStorage(s)
		}
	}
}

func InitFromConfiguration(configuration configurationtypes.AbstractConfigurationInterface) {
	if configuration.GetLogger() == nil {
		var logLevel zapcore.Level
		if configuration.GetLogLevel() == "" {
			logLevel = zapcore.FatalLevel
		} else if err := logLevel.UnmarshalText([]byte(configuration.GetLogLevel())); err != nil {
			logLevel = zapcore.FatalLevel
		}
		cfg := zap.Config{
			Encoding:         "json",
			Level:            zap.NewAtomicLevelAt(logLevel),
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
		configuration.SetLogger(logger)
	}
	logger := configuration.GetLogger()
	stale := configuration.GetDefaultCache().GetStale()
	tryToRegisterStorage(configuration.GetDefaultCache().GetBadger(), badger.Factory, logger, stale)
	tryToRegisterStorage(configuration.GetDefaultCache().GetEtcd(), etcd.Factory, logger, stale)
	tryToRegisterStorage(configuration.GetDefaultCache().GetNats(), nats.Factory, logger, stale)
	tryToRegisterStorage(configuration.GetDefaultCache().GetNuts(), nuts.Factory, logger, stale)
	tryToRegisterStorage(configuration.GetDefaultCache().GetOlric(), olric.Factory, logger, stale)
	tryToRegisterStorage(configuration.GetDefaultCache().GetOtter(), otter.Factory, logger, stale)
	tryToRegisterStorage(configuration.GetDefaultCache().GetRedis(), redis.Factory, logger, stale)
}
