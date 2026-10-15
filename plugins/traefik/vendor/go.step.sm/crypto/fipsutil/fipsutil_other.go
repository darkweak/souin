//go:build !go1.24

package fipsutil

// Enabled reports whether the cryptography libraries are operating in FIPS
// 140-3 mode.
//
// On Go < 1.24 it will always return false.
func Enabled() bool {
	return false
}

// Only reports whether the cryptography libraries are operating in FIPS 140-3
// "only" mode.
//
// On Go < 1.24 it will always return false.
func Only() bool {
	return false
}

// Version returns the FIPS 140-3 Go Cryptographic Module version.
//
// On Go < 1.24 there is no such module, so it will always return "latest".
func Version() string {
	return "latest"
}

// BuildVersion returns the GOFIPS140 setting recorded in the binary.
//
// On Go < 1.24 GOFIPS140 does not exist, so it will always return "".
func BuildVersion() string {
	return ""
}

// Validated reports whether the binary links a frozen, validated module
// snapshot.
//
// On Go < 1.24 it will always return false.
func Validated() bool {
	return false
}
