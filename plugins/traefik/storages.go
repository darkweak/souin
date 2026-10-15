package traefik

import (
	"fmt"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/storages/core"
	"github.com/darkweak/storages/otter"
	"github.com/darkweak/storages/simplefs"
)

type factory func(providerConfiguration core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error)

func isProviderEmpty(p configurationtypes.CacheProvider) bool {
	return !p.Found && p.Configuration == nil && p.Path == "" && p.URL == ""
}

// registerStorage instantiates and registers the storage, then stores its
// registry key in the provider so the middleware can elect it.
func registerStorage(p *configurationtypes.CacheProvider, f factory, logger core.Logger, stale time.Duration) {
	s, err := f(core.CacheProvider{Configuration: p.Configuration, Path: p.Path, URL: p.URL}, logger, stale)
	if err != nil {
		logger.Errorf("Impossible to instantiate the storage: %v", err)

		return
	}

	key := fmt.Sprintf("%s-%s", s.Name(), s.Uuid())
	if existing := core.GetRegisteredStorer(key); existing == nil {
		core.RegisterStorage(s)
	}

	p.Uuid = key
}

// initStorages registers the storages declared in the configuration, Otter
// (in-memory) and/or Simplefs (filesystem). Otter is used when none is set.
func initStorages(c *Configuration) {
	if c.Logger == nil {
		c.Logger = newLogger(c.LogLevel)
	}

	dc := c.DefaultCache
	stale := dc.GetStale()

	if isProviderEmpty(dc.Otter) && isProviderEmpty(dc.SimpleFS) {
		dc.Otter.Found = true
	}

	if !isProviderEmpty(dc.Otter) {
		registerStorage(&dc.Otter, otter.Factory, c.Logger, stale)
	}

	if !isProviderEmpty(dc.SimpleFS) {
		registerStorage(&dc.SimpleFS, simplefs.Factory, c.Logger, stale)
	}
}
