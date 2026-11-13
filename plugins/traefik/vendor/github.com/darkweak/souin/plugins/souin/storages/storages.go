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
	"github.com/darkweak/storages/simplefs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type factory func(providerConfiguration core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error)

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

// tryToRegisterStorage registers the storage declared by p and returns the
// key the middleware looks it up with, or an empty string.
func tryToRegisterStorage(name string, p configurationtypes.CacheProvider, f factory, logger core.Logger, stale time.Duration) string {
	if isProviderEmpty(p) {
		return ""
	}

	s, err := f(toCoreCacheProvider(p), logger, stale)
	if err != nil {
		logger.Errorf("Impossible to initialize the %s storage: %v", name, err)

		return ""
	}

	core.RegisterStorage(s)

	return s.Name() + "-" + s.Uuid()
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
		configuration.SetLogger(logger.Sugar())
	}
	logger := configuration.GetLogger()
	dc := configuration.GetDefaultCache()
	stale := dc.GetStale()

	badgerUuid := tryToRegisterStorage("badger", dc.GetBadger(), badger.Factory, logger, stale)
	etcdUuid := tryToRegisterStorage("etcd", dc.GetEtcd(), etcd.Factory, logger, stale)
	natsUuid := tryToRegisterStorage("nats", dc.GetNats(), nats.Factory, logger, stale)
	nutsUuid := tryToRegisterStorage("nuts", dc.GetNuts(), nuts.Factory, logger, stale)
	olricUuid := tryToRegisterStorage("olric", dc.GetOlric(), olric.Factory, logger, stale)
	otterUuid := tryToRegisterStorage("otter", dc.GetOtter(), otter.Factory, logger, stale)
	redisUuid := tryToRegisterStorage("redis", dc.GetRedis(), redis.Factory, logger, stale)
	simplefsUuid := tryToRegisterStorage("simplefs", dc.GetSimpleFS(), simplefs.Factory, logger, stale)

	// The middleware retrieves the registered storages by these identifiers.
	if d, ok := dc.(*configurationtypes.DefaultCache); ok && d != nil {
		d.Badger.Uuid = badgerUuid
		d.Etcd.Uuid = etcdUuid
		d.Nats.Uuid = natsUuid
		d.Nuts.Uuid = nutsUuid
		d.Olric.Uuid = olricUuid
		d.Otter.Uuid = otterUuid
		d.Redis.Uuid = redisUuid
		d.SimpleFS.Uuid = simplefsUuid
	}
}
