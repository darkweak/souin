// +build !linux,!darwin,!windows,!freebsd

package truststore

import "crypto/x509"

var (
	// NSSProfile is the path of the Firefox profiles.
	NSSProfile = ""

	// CertutilInstallHelp is the command to add NSS support.
	CertutilInstallHelp = ""
)

func installPlatform(filename string, cert *x509.Certificate) error {
	return ErrTrustNotSupported
}

func uninstallPlatform(filename string, cert *x509.Certificate) error {
	return ErrTrustNotSupported
}
