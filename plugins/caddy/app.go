package caddy

import (
	"github.com/caddyserver/caddy/v2"
)

type SouinApp struct {
	*DefaultCache
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
	if s.DefaultCache != nil && s.DefaultCache.TTL == "" {
		return new(defaultCacheError)
	}
	return nil
}

// Stop will stop the App
func (s SouinApp) Stop() error {
	return nil
}

// CaddyModule implements caddy.ModuleInfo
func (a SouinApp) CaddyModule() caddy.ModuleInfo {
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
