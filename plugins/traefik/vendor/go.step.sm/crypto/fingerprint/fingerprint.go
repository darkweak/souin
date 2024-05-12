package fingerprint

import (
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"go.step.sm/crypto/internal/emoji"
)

// Encoding defines the supported encodings for certificates and key
// fingerprints.
//
// This type is the base for sshutil.FingerprintEncoding and
// x509util.FingerprintEncoding types.
type Encoding int

const (
	// HexFingerprint represents the hex encoding of the fingerprint.
	//
	// This is the default encoding for an X.509 certificate.
	HexFingerprint Encoding = iota + 1
	// Base64Fingerprint represents the base64 encoding of the fingerprint.
	//
	// This is the default encoding for a public key.
	Base64Fingerprint
	// Base64URLFingerprint represents the base64URL encoding of the fingerprint.
	Base64URLFingerprint
	// Base64RawFingerprint represents the base64RawStd encoding of the
	// fingerprint.
	//
	// This is the default encoding for an SSH key and certificate.
	Base64RawFingerprint
	// Base64RawURLFingerprint represents the base64RawURL encoding of the fingerprint.
	Base64RawURLFingerprint
	// EmojiFingerprint represents the emoji encoding of the fingerprint.
	EmojiFingerprint
)

// New creates a fingerprint of the given data by hashing it and returns it in
// the encoding format.
func New(data []byte, h crypto.Hash, encoding Encoding) (string, error) {
	if !h.Available() {
		return "", fmt.Errorf("hash function %q is not available", h.String())
	}
	hash := h.New()
	if _, err := hash.Write(data); err != nil {
		return "", fmt.Errorf("error creating hash: %w", err)
	}
	fp := Fingerprint(hash.Sum(nil), encoding)
	if fp == "" {
		return "", fmt.Errorf("unknown encoding value %d", encoding)
	}
	return fp, nil
}

// Fingerprint encodes the given digest using the encoding format. If an invalid
// encoding is passed, the return value will be an empty string.
func Fingerprint(digest []byte, encoding Encoding) string {
	switch encoding {
	case HexFingerprint:
		return strings.ToLower(hex.EncodeToString(digest))
	case Base64Fingerprint:
		return base64.StdEncoding.EncodeToString(digest)
	case Base64URLFingerprint:
		return base64.URLEncoding.EncodeToString(digest)
	case Base64RawFingerprint:
		return base64.RawStdEncoding.EncodeToString(digest)
	case Base64RawURLFingerprint:
		return base64.RawURLEncoding.EncodeToString(digest)
	case EmojiFingerprint:
		return emoji.Emoji(digest)
	default:
		return ""
	}
}
