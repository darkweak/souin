package jose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"

	"github.com/pkg/errors"
	"go.step.sm/crypto/keyutil"
	"go.step.sm/crypto/pemutil"
	"go.step.sm/crypto/x25519"
)

const (
	jwksUsageSig = "sig"
	jwksUsageEnc = "enc"
	// defaultKeyType is the default type of the one-time token key.
	defaultKeyType = EC
	// defaultKeyCurve is the default curve of the one-time token key.
	defaultKeyCurve = P256
	// defaultKeyAlg is the default algorithm of the one-time token key.
	defaultKeyAlg = ES256
	// defaultKeySize is the default size of the one-time token key.
	defaultKeySize = 0
)

var (
	errAmbiguousCertKeyUsage = errors.New("jose/generate: certificate's key usage is ambiguous, it should be for signature or encipherment, but not both (use --subtle to ignore usage field)")
	errNoCertKeyUsage        = errors.New("jose/generate: certificate doesn't contain any key usage (use --subtle to ignore usage field)")
)

// Thumbprint computes the JWK Thumbprint of a key using SHA256 as the hash
// algorithm. It returns the hash encoded in the Base64 raw url encoding.
func Thumbprint(jwk *JSONWebKey) (string, error) {
	var sum []byte
	var err error
	switch key := jwk.Key.(type) {
	case x25519.PublicKey:
		sum, err = x25519Thumbprint(key, crypto.SHA256)
	case x25519.PrivateKey:
		var pub x25519.PublicKey
		if pub, err = key.PublicKey(); err == nil {
			sum, err = x25519Thumbprint(pub, crypto.SHA256)
		}
	case OpaqueSigner:
		sum, err = key.Public().Thumbprint(crypto.SHA256)
	default:
		sum, err = jwk.Thumbprint(crypto.SHA256)
	}
	if err != nil {
		return "", errors.Wrap(err, "error generating JWK thumbprint")
	}
	return base64.RawURLEncoding.EncodeToString(sum), nil
}

// GenerateDefaultKeyPair generates an asymmetric public/private key pair.
// Returns the public key as a JWK and the private key as an encrypted JWE.
func GenerateDefaultKeyPair(passphrase []byte) (*JSONWebKey, *JSONWebEncryption, error) {
	if len(passphrase) == 0 {
		return nil, nil, errors.New("step-jose: password cannot be empty when encryptying a JWK")
	}

	// Generate the OTT key
	jwk, err := GenerateJWK(defaultKeyType, defaultKeyCurve, defaultKeyAlg, jwksUsageSig, "", defaultKeySize)
	if err != nil {
		return nil, nil, err
	}

	jwk.KeyID, err = Thumbprint(jwk)
	if err != nil {
		return nil, nil, err
	}

	jwe, err := EncryptJWK(jwk, passphrase)
	if err != nil {
		return nil, nil, err
	}

	public := jwk.Public()
	return &public, jwe, nil
}

// GenerateJWK generates a JWK given the key type, curve, alg, use, kid and
// the size of the RSA or oct keys if necessary.
func GenerateJWK(kty, crv, alg, use, kid string, size int) (jwk *JSONWebKey, err error) {
	if kty == "OKP" && use == "enc" && (crv == "" || crv == "Ed25519") {
		return nil, errors.New("invalid algorithm: Ed25519 cannot be used for encryption")
	}

	switch {
	case kty == "EC" && crv == "":
		crv = P256
	case kty == "OKP" && crv == "":
		crv = Ed25519
	case kty == "RSA" && size == 0:
		size = DefaultRSASize
	case kty == "oct" && size == 0:
		size = DefaultOctSize
	}

	key, err := keyutil.GenerateKey(kty, crv, size)
	if err != nil {
		return nil, err
	}
	jwk = &JSONWebKey{
		Key:       key,
		KeyID:     kid,
		Use:       use,
		Algorithm: alg,
	}
	guessJWKAlgorithm(&context{alg: alg}, jwk)
	if jwk.KeyID == "" && kty != "oct" {
		jwk.KeyID, err = Thumbprint(jwk)
	}
	return jwk, err
}

// GenerateJWKFromPEM returns an incomplete JSONWebKey using the key from a
// PEM file.
func GenerateJWKFromPEM(filename string, subtle bool) (*JSONWebKey, error) {
	key, err := pemutil.Read(filename)
	if err != nil {
		return nil, err
	}

	switch key := key.(type) {
	case *rsa.PrivateKey, *rsa.PublicKey:
		return &JSONWebKey{
			Key: key,
		}, nil
	case *ecdsa.PrivateKey, *ecdsa.PublicKey, ed25519.PrivateKey, ed25519.PublicKey:
		return &JSONWebKey{
			Key:       key,
			Algorithm: algForKey(key),
		}, nil
	case *x509.Certificate:
		var use string
		if !subtle {
			use, err = keyUsageForCert(key)
			if err != nil {
				return nil, err
			}
		}
		return &JSONWebKey{
			Key:          key.PublicKey,
			Certificates: []*x509.Certificate{key},
			Algorithm:    algForKey(key.PublicKey),
			Use:          use,
		}, nil
	default:
		return nil, errors.Errorf("error parsing %s: unsupported key type '%T'", filename, key)
	}
}

func algForKey(key crypto.PublicKey) string {
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		return getECAlgorithm(key.Curve)
	case *ecdsa.PublicKey:
		return getECAlgorithm(key.Curve)
	case ed25519.PrivateKey, ed25519.PublicKey:
		return EdDSA
	default:
		return ""
	}
}

func keyUsageForCert(cert *x509.Certificate) (string, error) {
	isDigitalSignature := containsUsage(cert.KeyUsage,
		x509.KeyUsageDigitalSignature,
		x509.KeyUsageContentCommitment,
		x509.KeyUsageCertSign,
		x509.KeyUsageCRLSign,
	)
	isEncipherment := containsUsage(cert.KeyUsage,
		x509.KeyUsageKeyEncipherment,
		x509.KeyUsageDataEncipherment,
		x509.KeyUsageKeyAgreement,
		x509.KeyUsageEncipherOnly,
		x509.KeyUsageDecipherOnly,
	)
	if isDigitalSignature && isEncipherment {
		return "", errAmbiguousCertKeyUsage
	}
	if isDigitalSignature {
		return jwksUsageSig, nil
	}
	if isEncipherment {
		return jwksUsageEnc, nil
	}
	return "", errNoCertKeyUsage
}

func containsUsage(usage x509.KeyUsage, queries ...x509.KeyUsage) bool {
	for _, query := range queries {
		if usage&query == query {
			return true
		}
	}
	return false
}
