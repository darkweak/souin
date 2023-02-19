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
func (m *Plugin) Init(cfg Configurer, log *zap.Logger) error {
	const op = errors.Op("httpcache_middleware_init")
	if !cfg.Has(configurationKey) {
		return errors.E(op, errors.Disabled)
	}

	c := parseConfiguration(cfg)
	c.SetLogger(log)

	m.SouinBaseHandler = middleware.NewHTTPCacheHandler(&c)

	return nil
}

// Middleware is the request entrypoint to determine if either a cached
// response can be reused or if the roundtrip response can be stored in
// the cache system.
func (m *Plugin) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, rq *http.Request) {
		m.SouinBaseHandler.ServeHTTP(rw, rq, func(w http.ResponseWriter, r *http.Request) error {
			next.ServeHTTP(w, r)

			return nil
		})
	})
}

/*
// TODO implement the part given from rustatian once they release the next major verison
// Name is the plugin name
func (p *Plugin) Name() string {
	return pluginName
}

// Init allows the user to set up an HTTP cache system,
// RFC-7234 compliant and supports the tag based cache purge,
// distributed and not-distributed storage, key generation tweaking.
func (p *Plugin) Init(cfg Configurer, log Logger) error {
	const op = errors.Op("httpcache_middleware_init")
	if !cfg.Has(configurationKey) {
		return errors.E(op, errors.Disabled)
	}

	c := parseConfiguration(cfg)
	c.SetLogger(log.NamedLogger(pluginName))
	p.Configuration = &c
	p.bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	p.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(&c)
	p.RequestCoalescing = coalescing.Initialize()
	p.MapHandler = api.GenerateHandlerMap(p.Configuration, p.Retriever.GetTransport())

	return nil
}

// Middleware is the request entrypoint to determine if either a cached
// response can be reused or if the roundtrip response can be stored in
// the cache system.
func (p *Plugin) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		req := p.Retriever.GetContext().SetBaseContext(r)
		if b, handler := p.HandleInternally(req); b {
			handler(rw, req)

			return
		}

		if !plugins.CanHandle(req, p.Retriever) {
			rfc.MissCache(rw.Header().Set, req, "CANNOT-HANDLE")
			next.ServeHTTP(rw, r)

			return
		}

		customWriter := &plugins.CustomWriter{
			Response: &http.Response{},
			Buf:      p.bufPool.Get().(*bytes.Buffer),
			Rw:       rw,
			Req:      req,
		}
		req = p.Retriever.GetContext().SetContext(req)
		getterCtx := getterContext{next.ServeHTTP, customWriter, req}
		ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
		req = req.WithContext(ctx)
		if plugins.HasMutation(req, rw) {
			next.ServeHTTP(rw, r)

			return
		}
		req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		combo := ctx.Value(getterContextCtxKey).(getterContext)

		_ = plugins.DefaultSouinPluginCallback(customWriter, req, p.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
			var e error
			combo.next.ServeHTTP(customWriter, r)

			combo.req.Response = customWriter.Response
			if combo.req.Response.StatusCode == 0 {
				combo.req.Response.StatusCode = 200
			}
			combo.req.Response, e = p.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(combo.req)

			return e
		})
	})
}
*/
