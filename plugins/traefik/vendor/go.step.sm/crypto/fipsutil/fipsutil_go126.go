//go:build go1.26

package fipsutil

import "crypto/fips140"

// Only reports whether the cryptography libraries are operating in FIPS 140-3
// "only" mode. When in this mode, using non-approved cryptography functions
// will return errors or panic.
//
// Note that the Go project documents "only" mode as a best-effort mode for
// testing, assessment and debugging that is not intended for production use.
func Only() bool {
	return fips140.Enforced()
}

// Version returns the FIPS 140-3 Go Cryptographic Module version, such as
// "v1.0.0", when the program was built against a frozen module with GOFIPS140,
// and "latest" otherwise.
//
// Only a concrete version ties a running process to a CMVP certificate;
// "latest" ties it to nothing.
func Version() string {
	return fips140.Version()
}
