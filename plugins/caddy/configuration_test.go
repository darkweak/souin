package httpcache

import (
	"encoding/json"
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// roundTripCaddyfileDirective parses a `cache` handler directive the way the
// Caddyfile adapter does, then marshals/unmarshals it to JSON exactly like Caddy
// does before provisioning the module.
func roundTripCaddyfileDirective(t *testing.T, input string) (*SouinCaddyMiddleware, error) {
	t.Helper()

	var parsed SouinCaddyMiddleware
	if err := parsed.UnmarshalCaddyfile(caddyfile.NewTestDispenser(input)); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(&parsed)
	if err != nil {
		t.Fatalf("unable to marshal the parsed configuration: %v", err)
	}

	var loaded SouinCaddyMiddleware
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unable to unmarshal the configuration %s: %v", raw, err)
	}

	return &loaded, nil
}

// Regression test for https://github.com/darkweak/souin/issues/847.
func TestSimpleFSConfigurationSizes(t *testing.T) {
	cases := map[string]float64{
		"10485760": 10485760,
		"10mb":     10_000_000,
		"10MiB":    10485760,
		"1GB":      1_000_000_000,
	}

	for value, expected := range cases {
		loaded, err := roundTripCaddyfileDirective(t, `cache {
			simplefs {
				configuration {
					size 1000
					directory_size `+value+`
				}
			}
		}`)
		if err != nil {
			t.Fatalf("unexpected error for directory_size %s: %v", value, err)
		}

		cfg, ok := loaded.Configuration.DefaultCache.SimpleFS.Configuration.(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected simplefs configuration type %T", loaded.Configuration.DefaultCache.SimpleFS.Configuration)
		}

		if cfg["directory_size"] != expected {
			t.Errorf("directory_size %s: expected %v, got %#v", value, expected, cfg["directory_size"])
		}
		if cfg["size"] != float64(1000) {
			t.Errorf("size: expected 1000, got %#v", cfg["size"])
		}
	}
}

func TestSimpleFSConfigurationInvalidSizes(t *testing.T) {
	for _, block := range []string{
		"directory_size tenmegs",
		"directory_size 10 mb extra",
		"size 10mb",
		"size -1",
	} {
		_, err := roundTripCaddyfileDirective(t, `cache {
			simplefs {
				configuration {
					`+block+`
				}
			}
		}`)
		if err == nil {
			t.Errorf("expected an error for %q, got nil", block)
		}
	}
}
