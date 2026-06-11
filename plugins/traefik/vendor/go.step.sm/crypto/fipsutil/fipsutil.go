//go:build go1.24

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

// Enabled reports whether the cryptography libraries are operating in FIPS
// 140-3 mode.
//
// It can be controlled at runtime using the GODEBUG setting "fips140". If set
// to "on", FIPS 140-3 mode is enabled. If set to "only", non-approved
// cryptography functions will additionally return errors or panic.
//
// This can't be changed after the program has started.
func Enabled() bool {
	return fips140.Enabled()
}

// Only reports whether the cryptography libraries are operating in FIPS 140-3
// "only" mode. When in this mode, using non-approved cryptography functions
// will return errors or panic.
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
