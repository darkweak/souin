// Package keyutil implements utilities to generate cryptographic keys.
package keyutil

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"math/big"
	"sync/atomic"

	"github.com/pkg/errors"
	"go.step.sm/crypto/x25519"
	"golang.org/x/crypto/ssh"
)

var (
	// DefaultKeyType is the default type of a private key.
	DefaultKeyType = "EC"
	// DefaultKeySize is the default size (in # of bits) of a private key.
	DefaultKeySize = 2048
	// DefaultKeyCurve is the default curve of a private key.
	DefaultKeyCurve = "P-256"
	// DefaultSignatureAlgorithm is the default signature algorithm used on a
	// certificate with the default key type.
	DefaultSignatureAlgorithm = x509.ECDSAWithSHA256
	// MinRSAKeyBytes is the minimum acceptable size (in bytes) for RSA keys
	// signed by the authority.
	MinRSAKeyBytes = 256
)

type atomicBool int32

func (b *atomicBool) isSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue()    { atomic.StoreInt32((*int32)(b), 1) }
func (b *atomicBool) setFalse()   { atomic.StoreInt32((*int32)(b), 0) }

var insecureMode atomicBool

// Insecure enables the insecure mode in this package and returns a function to
// revert the configuration. The insecure mode removes the minimum limits when
// generating RSA keys.
func Insecure() (revert func()) {
	insecureMode.setTrue()
	return func() {
		insecureMode.setFalse()
	}
}

// PublicKey extracts a public key from a private key.
func PublicKey(priv interface{}) (crypto.PublicKey, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	case ed25519.PrivateKey:
		return k.Public(), nil
	case x25519.PrivateKey:
		return k.Public(), nil
	case *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey, x25519.PublicKey:
		return k, nil
	case crypto.Signer:
		return k.Public(), nil
	default:
		return nil, errors.Errorf("unrecognized key type: %T", priv)
	}
}

// GenerateDefaultKey generates a public/private key pair using sane defaults
// for key type, curve, and size.
func GenerateDefaultKey() (crypto.PrivateKey, error) {
	return GenerateKey(DefaultKeyType, DefaultKeyCurve, DefaultKeySize)
}

// GenerateDefaultKeyPair generates a public/private key pair using configured
// default values for key type, curve, and size.
func GenerateDefaultKeyPair() (crypto.PublicKey, crypto.PrivateKey, error) {
	return GenerateKeyPair(DefaultKeyType, DefaultKeyCurve, DefaultKeySize)
}

// GenerateKey generates a key of the given type (kty).
func GenerateKey(kty, crv string, size int) (crypto.PrivateKey, error) {
	switch kty {
	case "EC", "RSA", "OKP":
		return GenerateSigner(kty, crv, size)
	case "oct":
		return generateOctKey(size)
	default:
		return nil, errors.Errorf("unrecognized key type: %s", kty)
	}
}

// GenerateKeyPair creates an asymmetric crypto keypair using input
// configuration.
func GenerateKeyPair(kty, crv string, size int) (crypto.PublicKey, crypto.PrivateKey, error) {
	signer, err := GenerateSigner(kty, crv, size)
	if err != nil {
		return nil, nil, err
	}
	return signer.Public(), signer, nil
}

// GenerateDefaultSigner returns an asymmetric crypto key that implements
// crypto.Signer using sane defaults.
func GenerateDefaultSigner() (crypto.Signer, error) {
	return GenerateSigner(DefaultKeyType, DefaultKeyCurve, DefaultKeySize)
}

// GenerateSigner creates an asymmetric crypto key that implements
// crypto.Signer.
func GenerateSigner(kty, crv string, size int) (crypto.Signer, error) {
	switch kty {
	case "EC":
		return generateECKey(crv)
	case "RSA":
		return generateRSAKey(size)
	case "OKP":
		return generateOKPKey(crv)
	default:
		return nil, errors.Errorf("unrecognized key type: %s", kty)
	}
}

// ExtractKey returns the given public or private key or extracts the public key
// if a x509.Certificate or x509.CertificateRequest is given.
func ExtractKey(in interface{}) (interface{}, error) {
	switch k := in.(type) {
	case *rsa.PublicKey, *rsa.PrivateKey,
		*ecdsa.PublicKey, *ecdsa.PrivateKey,
		ed25519.PublicKey, ed25519.PrivateKey,
		x25519.PublicKey, x25519.PrivateKey:
		return in, nil
	case []byte:
		return in, nil
	case *x509.Certificate:
		return k.PublicKey, nil
	case *x509.CertificateRequest:
		return k.PublicKey, nil
	case ssh.CryptoPublicKey:
		return k.CryptoPublicKey(), nil
	case *ssh.Certificate:
		return ExtractKey(k.Key)
	default:
		return nil, errors.Errorf("cannot extract the key from type '%T'", k)
	}
}

// VerifyPair that the public key matches the given private key.
func VerifyPair(pub crypto.PublicKey, priv crypto.PrivateKey) error {
	signer, ok := priv.(crypto.Signer)
	if !ok {
		return errors.New("private key type does implement crypto.Signer")
	}
	if !Equal(pub, signer.Public()) {
		return errors.New("private key does not match public key")
	}
	return nil
}

// Equal reports if x and y are the same key.
func Equal(x, y any) bool {
	switch xx := x.(type) {
	case *ecdsa.PublicKey:
		yy, ok := y.(*ecdsa.PublicKey)
		return ok && xx.Equal(yy)
	case *ecdsa.PrivateKey:
		yy, ok := y.(*ecdsa.PrivateKey)
		return ok && xx.Equal(yy)
	case *rsa.PublicKey:
		yy, ok := y.(*rsa.PublicKey)
		return ok && xx.Equal(yy)
	case *rsa.PrivateKey:
		yy, ok := y.(*rsa.PrivateKey)
		return ok && xx.Equal(yy)
	case ed25519.PublicKey:
		yy, ok := y.(ed25519.PublicKey)
		return ok && xx.Equal(yy)
	case ed25519.PrivateKey:
		yy, ok := y.(ed25519.PrivateKey)
		return ok && xx.Equal(yy)
	case x25519.PublicKey:
		yy, ok := y.(x25519.PublicKey)
		return ok && xx.Equal(yy)
	case x25519.PrivateKey:
		yy, ok := y.(x25519.PrivateKey)
		return ok && xx.Equal(yy)
	case []byte: // special case for symmetric keys
		yy, ok := y.([]byte)
		return ok && bytes.Equal(xx, yy)
	default:
		return false
	}
}

func generateECKey(crv string) (crypto.Signer, error) {
	var c elliptic.Curve
	switch crv {
	case "P-256":
		c = elliptic.P256()
	case "P-384":
		c = elliptic.P384()
	case "P-521":
		c = elliptic.P521()
	default:
		return nil, errors.Errorf("invalid value for argument crv (crv: '%s')", crv)
	}

	key, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "error generating EC key")
	}

	return key, nil
}

func generateRSAKey(bits int) (crypto.Signer, error) {
	if minBits := MinRSAKeyBytes * 8; !insecureMode.isSet() && bits < minBits {
		return nil, errors.Errorf("the size of the RSA key should be at least %d bits", minBits)
	}

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, errors.Wrap(err, "error generating RSA key")
	}

	return key, nil
}

func generateOKPKey(crv string) (crypto.Signer, error) {
	switch crv {
	case "Ed25519":
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, errors.Wrap(err, "error generating Ed25519 key")
		}
		return key, nil
	case "X25519":
		_, key, err := x25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, errors.Wrap(err, "error generating X25519 key")
		}
		return key, nil
	default:
		return nil, errors.Errorf("missing or invalid value for argument 'crv'. "+
			"expected 'Ed25519' or 'X25519', but got '%s'", crv)
	}
}

func generateOctKey(size int) (interface{}, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, size)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return nil, err
		}
		result[i] = chars[num.Int64()]
	}
	return result, nil
}
