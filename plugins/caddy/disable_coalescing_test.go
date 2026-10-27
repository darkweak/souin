package httpcache

import "testing"

// The global `disable_coalescing` option is parsed into the app's DefaultCache, so FromApp must
// carry it into the handler configuration, like every other global option it inherits.
func TestFromAppInheritsDisableCoalescing(t *testing.T) {
	s := &SouinCaddyMiddleware{}
	app := &SouinApp{DefaultCache: DefaultCache{DisableCoalescing: true}}

	if err := s.FromApp(app); err != nil {
		t.Fatalf("FromApp: %v", err)
	}
	if !s.Configuration.DefaultCache.IsCoalescingDisable() {
		t.Error("the global disable_coalescing was not inherited by the handler")
	}
}

func TestFromAppKeepsCoalescingWhenNotDisabled(t *testing.T) {
	s := &SouinCaddyMiddleware{}
	app := &SouinApp{}

	if err := s.FromApp(app); err != nil {
		t.Fatalf("FromApp: %v", err)
	}
	if s.Configuration.DefaultCache.IsCoalescingDisable() {
		t.Error("coalescing was disabled although neither the app nor the handler disabled it")
	}
}
