package keyutil

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"

	"go.step.sm/crypto/fingerprint"
)

// FingerprintEncoding defines the supported encodings in certificate
// fingerprints.
type FingerprintEncoding = fingerprint.Encoding

// Supported fingerprint encodings.
const (
	// DefaultFingerprint represents the base64 encoding of the fingerprint.
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

// subjectPublicKeyInfo is a PKIX public key structure defined in RFC 5280.
type subjectPublicKeyInfo struct {
	Algorithm        pkix.AlgorithmIdentifier
	SubjectPublicKey asn1.BitString
}

// Fingerprint returns the SHA-256 fingerprint of an public key.
//
// The fingerprint is calculated from the encoding of the key according to RFC
// 5280 section 4.2.1.2, but using SHA-256 instead of SHA-1.
func Fingerprint(pub crypto.PublicKey) (string, error) {
	return EncodedFingerprint(pub, DefaultFingerprint)
}

// EncodedFingerprint returns the SHA-256 hash of the certificate using the
// specified encoding.
//
// The fingerprint is calculated from the encoding of the key according to RFC
// 5280 section 4.2.1.2, but using SHA-256 instead of SHA-1.
func EncodedFingerprint(pub crypto.PublicKey, encoding FingerprintEncoding) (string, error) {
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("error marshaling public key: %w", err)
	}
	var info subjectPublicKeyInfo
	if _, err = asn1.Unmarshal(b, &info); err != nil {
		return "", fmt.Errorf("error unmarshaling public key: %w", err)
	}
	if encoding == DefaultFingerprint {
		encoding = Base64Fingerprint
	}

	sum := sha256.Sum256(info.SubjectPublicKey.Bytes)
	fp := fingerprint.Fingerprint(sum[:], encoding)
	if fp == "" {
		return "", fmt.Errorf("error formatting fingerprint: unsupported encoding")
	}
	return "SHA256:" + fp, nil
}
