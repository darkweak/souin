package sshutil

import (
	"crypto/dsa" //nolint:staticcheck // support for DSA fingerprints
	"crypto/rsa"
	"crypto/sha256"
	"fmt"

	"github.com/pkg/errors"
	"go.step.sm/crypto/fingerprint"
	"golang.org/x/crypto/ssh"
)

// FingerprintEncoding defines the supported encodings for SSH key and
// certificate fingerprints.
type FingerprintEncoding = fingerprint.Encoding

// Supported fingerprint encodings.
const (
	// DefaultFingerprint represents base64RawStd encoding of the fingerprint.
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

// Fingerprint returns the SHA-256 fingerprint of an ssh public key or
// certificate.
func Fingerprint(pub ssh.PublicKey) string {
	return EncodedFingerprint(pub, DefaultFingerprint)
}

// EncodedFingerprint returns the SHA-256 hash of an ssh public key or
// certificate using the specified encoding. If an invalid encoding is passed,
// the return value will be an empty string.
func EncodedFingerprint(pub ssh.PublicKey, encoding FingerprintEncoding) string {
	var fp string

	sum := sha256.Sum256(pub.Marshal())
	switch encoding {
	case DefaultFingerprint:
		fp = fingerprint.Fingerprint(sum[:], Base64RawFingerprint)
	default:
		fp = fingerprint.Fingerprint(sum[:], encoding)
	}
	if fp == "" {
		return ""
	}
	return "SHA256:" + fp
}

// FormatFingerprint parses a public key from an authorized_keys file used in
// OpenSSH and returns a public key fingerprint in the following format:
//
//	<size> SHA256:<base64-raw-fingerprint> <comment> (<type)
//
// If the input is an SSH certificate, its public key will be extracted and
// taken as input for the fingerprint.
func FormatFingerprint(in []byte, encoding FingerprintEncoding) (string, error) {
	return formatFingerprint(in, encoding, false)
}

// FormatCertificateFingerprint parses an SSH certificate as used by
// OpenSSH and returns a public key fingerprint in the following format:
//
//	<size> SHA256:<base64-raw-fingerprint> <comment> (<type)
//
// If the input is not an SSH certificate, an error will be returned.
func FormatCertificateFingerprint(in []byte, encoding FingerprintEncoding) (string, error) {
	return formatFingerprint(in, encoding, true)
}

// formatFingerprint parses a public key from an authorized_keys file or an
// SSH certificate as used by OpenSSH and returns a public key fingerprint
// in the following format:
//
//	<size> SHA256:<base64-raw-fingerprint> <comment> (<type)
//
// If the input is an SSH certificate and `asCertificate` is false, the certificate
// public key will be used as input for the fingerprint. If `asCertificate` is true,
// the full contents of the certificate will be used in the fingerprint. If the input
// is not an SSH certificate, but `asCertificate` is true, an error will be returned.
func formatFingerprint(in []byte, encoding FingerprintEncoding, asCertificate bool) (string, error) {
	key, comment, _, _, err := ssh.ParseAuthorizedKey(in)
	if err != nil {
		return "", fmt.Errorf("error parsing public key: %w", err)
	}
	cert, keyIsCertificate := key.(*ssh.Certificate)
	if asCertificate && !keyIsCertificate {
		return "", fmt.Errorf("cannot fingerprint SSH key as SSH certificate")
	}
	if comment == "" {
		comment = "no comment"
	}

	typ, size, err := publicKeyTypeAndSize(key)
	if err != nil {
		return "", fmt.Errorf("error determining key type and size: %w", err)
	}

	// if the SSH key is actually an SSH certificate and when
	// the fingerprint has to be determined for the public key,
	// get the public key from the certificate and encode just
	// that, instead of encoding the entire key blob including
	// certificate bytes.
	publicKey := key
	if keyIsCertificate && !asCertificate {
		publicKey = cert.Key
	}

	fp := EncodedFingerprint(publicKey, encoding)
	if fp == "" {
		return "", fmt.Errorf("unsupported encoding format %v", encoding)
	}

	return fmt.Sprintf("%d %s %s (%s)", size, fp, comment, typ), nil
}

func publicKeyTypeAndSize(key ssh.PublicKey) (string, int, error) {
	var isCert bool
	if cert, ok := key.(*ssh.Certificate); ok {
		key = cert.Key
		isCert = true
	}

	var typ string
	var size int
	switch key.Type() {
	case ssh.KeyAlgoECDSA256:
		typ, size = "ECDSA", 256
	case ssh.KeyAlgoECDSA384:
		typ, size = "ECDSA", 384
	case ssh.KeyAlgoECDSA521:
		typ, size = "ECDSA", 521
	case ssh.KeyAlgoSKECDSA256:
		typ, size = "SK-ECDSA", 256
	case ssh.KeyAlgoED25519:
		typ, size = "ED25519", 256
	case ssh.KeyAlgoSKED25519:
		typ, size = "SK-ED25519", 256
	case ssh.KeyAlgoRSA:
		typ = "RSA"
		cpk, err := CryptoPublicKey(key)
		if err != nil {
			return "", 0, err
		}
		k, ok := cpk.(*rsa.PublicKey)
		if !ok {
			return "", 0, errors.New("unsupported key: not an RSA public key")
		}
		size = 8 * k.Size()
	case ssh.KeyAlgoDSA:
		typ = "DSA"
		cpk, err := CryptoPublicKey(key)
		if err != nil {
			return "", 0, err
		}
		k, ok := cpk.(*dsa.PublicKey)
		if !ok {
			return "", 0, errors.New("unsupported key: not a DSA public key")
		}
		size = k.Parameters.P.BitLen()
	default:
		return "", 0, errors.Errorf("public key %s is not supported", key.Type())
	}

	if isCert {
		typ += "-CERT"
	}

	return typ, size, nil
}
