package caddy

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// SouinApp contains the whole Souin necessary items
type SouinApp struct {
	*DefaultCache
	Provider types.AbstractProviderInterface
	API      configurationtypes.API `json:"api,omitempty"`
	LogLevel string                 `json:"log_level,omitempty"`
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
	if s.DefaultCache != nil && s.DefaultCache.GetTTL() == 0 {
		return new(defaultCacheError)
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
