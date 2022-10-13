package httpcache

import (
	"errors"

	"github.com/caddyserver/caddy/v2"
	"github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// SouinApp contains the whole Souin necessary items
type SouinApp struct {
	*DefaultCache
	// The provider to use.
	Provider types.AbstractProviderInterface
	// Surrogate storage to support th econfiguration reload without surrogate-key data loss.
	SurrogateStorage providers.SurrogateInterface
	// Cache-key tweaking.
	CacheKeys map[string]configurationtypes.Key `json:"cache_keys,omitempty"`
	// API endpoints enablers.
	API configurationtypes.API `json:"api,omitempty"`
	// Logger level, fallback on caddy's one when not redefined.
	LogLevel string `json:"log_level,omitempty"`
}

func init() {
	caddy.RegisterModule(SouinApp{})
}

// Provision implements caddy.Provisioner
func (s *SouinApp) Provision(_ caddy.Context) error {
	return nil
}

// Start will start the App
func (s SouinApp) Start() error {
	_, _ = up.Delete(stored_providers_key)
	_, _ = up.LoadOrStore(stored_providers_key, newStorageProvider())
	if s.DefaultCache != nil && s.DefaultCache.GetTTL() == 0 {
		return errors.New("Invalid/Incomplete default cache declaration")
	}
	return nil
}

// Stop will stop the App
func (s SouinApp) Stop() error {
	return nil
}

// CaddyModule implements caddy.ModuleInfo
func (s SouinApp) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  moduleName,
		New: func() caddy.Module { return new(SouinApp) },
	}
}

var (
	_ caddy.App         = (*SouinApp)(nil)
	_ caddy.Module      = (*SouinApp)(nil)
	_ caddy.Provisioner = (*SouinApp)(nil)
)
