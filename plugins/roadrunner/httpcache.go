package roadrunner

import (
	"net/http"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/roadrunner-server/errors"
	"go.uber.org/zap"
)

const pluginName string = "cache"

type (
	// Configurer interface used to parse yaml configuration.
	// Implementation will be provided by the RoadRunner automatically via Init method.
	Configurer interface {
		// Get used to get config section
		Get(name string) any
		// Has checks if config section exists.
		Has(name string) bool
	}

	// Logger is the main RR's logger interface
	// It's providing a named instance of the *zap.Logger
	Logger interface {
		NamedLogger(name string) *zap.Logger
	}

	Plugin struct {
		*middleware.SouinBaseHandler
	}
)

// Name is the plugin name
func (*Plugin) Name() string {
	return pluginName
}

// Init allows the user to set up an HTTP cache system,
// RFC-7234 compliant and supports the tag based cache purge,
// distributed and not-distributed storage, key generation tweaking.
func (m *Plugin) Init(cfg Configurer, log Logger) error {
	const op = errors.Op("httpcache_middleware_init")
	if !cfg.Has(configurationKey) {
		return errors.E(op, errors.Disabled)
	}

	c := parseConfiguration(cfg)
	c.SetLogger(log.NamedLogger(pluginName))

	m.SouinBaseHandler = middleware.NewHTTPCacheHandler(&c)

	return nil
}

// Middleware is the request entrypoint to determine if either a cached
// response can be reused or if the roundtrip response can be stored in
// the cache system.
func (m *Plugin) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, rq *http.Request) {
		_ = m.SouinBaseHandler.ServeHTTP(rw, rq, func(w http.ResponseWriter, r *http.Request) error {
			next.ServeHTTP(w, r)

			return nil
		})
	})
}
