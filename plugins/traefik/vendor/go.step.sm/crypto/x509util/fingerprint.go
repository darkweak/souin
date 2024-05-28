package x509util

import (
	"crypto/sha256"
	"crypto/x509"

	"go.step.sm/crypto/fingerprint"
)

// FingerprintEncoding defines the supported encodings in certificate
// fingerprints.
type FingerprintEncoding = fingerprint.Encoding

// Supported fingerprint encodings.
const (
	// DefaultFingerprint represents the hex encoding of the fingerprint.
	DefaultFingerprint = FingerprintEncoding(0)
	// HexFingerprint represents the hex encoding of the fingerprint.
	HexFingerprint = fingerprint.HexFingerprint
	// Base64Fingerprint represents the base64 encoding of the fingerprint.
	Base64Fingerprint = fingerprint.Base64Fingerprint
	// Base64URLFingerprint represents the base64URL encoding of the fingerprint.
	Base64URLFingerprint = fingerprint.Base64URLFingerprint
	// Base64RawFingerprint represents the base64RawStd encoding of the fingerprint.
	Base64RawFingerprint = fingerprint.Base64RawFingerprint
	// Base64RawURLFingerprint represents the base64RawURL encoding of the fingerprint.
	Base64RawURLFingerprint = fingerprint.Base64RawURLFingerprint
	// EmojiFingerprint represents the emoji encoding of the fingerprint.
	EmojiFingerprint = fingerprint.EmojiFingerprint
)

// Fingerprint returns the SHA-256 fingerprint of the certificate.
func Fingerprint(cert *x509.Certificate) string {
	return EncodedFingerprint(cert, DefaultFingerprint)
}

// EncodedFingerprint returns the SHA-256 hash of the certificate using the
// specified encoding. If an invalid encoding is passed, the return value will
// be an empty string.
func EncodedFingerprint(cert *x509.Certificate, encoding FingerprintEncoding) string {
	sum := sha256.Sum256(cert.Raw)
	switch encoding {
	case DefaultFingerprint:
		return fingerprint.Fingerprint(sum[:], HexFingerprint)
	default:
		return fingerprint.Fingerprint(sum[:], encoding)
	}
}
