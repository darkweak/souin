//go:build go1.24 && !go1.26

package fipsutil

import (
	"crypto/fips140"
	"os"
	"strings"
	"sync"
)

var (
	only bool
	once sync.Once
)

// Only reports whether the cryptography libraries are operating in FIPS 140-3
// "only" mode. When in this mode, using non-approved cryptography functions
// will return errors or panic.
//
// Before Go 1.26 the runtime does not expose this, so it is inferred from the
// GODEBUG environment variable. That misses a default baked in at build time
// by GOFIPS140 or a //go:debug directive; Go 1.26 and later ask the runtime
// directly and do not have that gap.
func Only() bool {
	once.Do(func() {
		if !fips140.Enabled() {
			return
		}

		// Parse GODEBUG backwards as the last value is the correct one.
		settings := strings.Split(os.Getenv("GODEBUG"), ",")
		for i := len(settings) - 1; i >= 0; i-- {
			if settings[i] == "fips140=only" {
				only = true
				return
			}
		}
	})

	return only
}

// Version returns the FIPS 140-3 Go Cryptographic Module version when the
// program was built against a frozen module with GOFIPS140, and "latest"
// otherwise.
//
// crypto/fips140.Version was added in Go 1.26. Before that this reports the
// resolved GOFIPS140 build setting instead, so it carries the snapshot suffix
// ("v1.0.0-c2097c7c") where Go 1.26 and later report the formal version
// ("v1.0.0"). Use [BuildVersion] when you want the resolved value on every
// Go version.
func Version() string {
	if v := BuildVersion(); v != "" && v != "off" {
		return v
	}

	return "latest"
}
