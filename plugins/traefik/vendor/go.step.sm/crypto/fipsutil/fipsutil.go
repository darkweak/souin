//go:build go1.24

// Package fipsutil reports how the running process relates to the FIPS 140-3
// Go Cryptographic Module.
package fipsutil

import (
	"crypto/fips140"
	"runtime/debug"
	"sync"
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

var (
	buildVersion     string
	buildVersionOnce sync.Once
)

// BuildVersion returns the GOFIPS140 setting recorded in the binary, such as
// "v1.0.0-c2097c7c", or the empty string if it was built without one.
//
// This is the resolved version rather than the value passed to the go command:
// GOFIPS140=v1.0.0 is read through $GOROOT/lib/fips140/v1.0.0.txt and records
// as the exact snapshot whose checksum is fixed in the module's security
// policy.
func BuildVersion() string {
	buildVersionOnce.Do(func() {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		for _, s := range info.Settings {
			if s.Key == "GOFIPS140" {
				buildVersion = s.Value
				return
			}
		}
	})

	return buildVersion
}

// Validated reports whether the binary links a frozen module snapshot rather
// than the toolchain's in-tree crypto packages.
//
// It says nothing about whether FIPS 140-3 mode is currently on; use [Enabled]
// for that.
func Validated() bool {
	switch BuildVersion() {
	case "", "off", "latest":
		return false
	default:
		return true
	}
}
