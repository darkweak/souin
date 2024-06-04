//go:build !linux && !darwin && !windows && !freebsd
// +build !linux,!darwin,!windows,!freebsd

package truststore

import "crypto/x509"

var (
	// NSSProfile is the path of the Firefox profiles.
	NSSProfile = ""

	// CertutilInstallHelp is the command to add NSS support.
	CertutilInstallHelp = ""
)

func installPlatform(string, *x509.Certificate) error {
	return ErrTrustNotSupported
}

func uninstallPlatform(string, *x509.Certificate) error {
	return ErrTrustNotSupported
}
