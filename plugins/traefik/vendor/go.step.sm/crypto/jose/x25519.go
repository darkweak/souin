package jose

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"go.step.sm/crypto/x25519"
)

const x25519ThumbprintTemplate = `{"crv":"X25519","kty":"OKP","x":%q}`

func x25519Thumbprint(key x25519.PublicKey, hash crypto.Hash) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid elliptic key")
	}
	h := hash.New()
	fmt.Fprintf(h, x25519ThumbprintTemplate, base64.RawURLEncoding.EncodeToString(key))
	return h.Sum(nil), nil
}

// X25519Signer implements the jose.OpaqueSigner using an X25519 key and XEdDSA
// as the signing algorithm.
type X25519Signer x25519.PrivateKey

// Public returns the public key of the current signing key.
func (s X25519Signer) Public() *JSONWebKey {
	return &JSONWebKey{
		Key: x25519.PrivateKey(s).Public(),
	}
}

// Algs returns a list of supported signing algorithms, in this case only
// XEdDSA.
func (s X25519Signer) Algs() []SignatureAlgorithm {
	return []SignatureAlgorithm{
		XEdDSA,
	}
}

// SignPayload signs a payload with the current signing key using the given
// algorithm, it will fail if it's not XEdDSA.
func (s X25519Signer) SignPayload(payload []byte, alg SignatureAlgorithm) ([]byte, error) {
	if alg != XEdDSA {
		return nil, errors.Errorf("x25519 key does not support the signature algorithm %s", alg)
	}
	return x25519.PrivateKey(s).Sign(rand.Reader, payload, crypto.Hash(0))
}

// X25519Verifier implements the jose.OpaqueVerifier interface using an X25519
// key and XEdDSA as a signing algorithm.
type X25519Verifier x25519.PublicKey

// VerifyPayload verifies the given signature using the X25519 public key, it
// will fail if the signature algorithm is not XEdDSA.
func (v X25519Verifier) VerifyPayload(payload, signature []byte, alg SignatureAlgorithm) error {
	if alg != XEdDSA {
		return errors.Errorf("x25519 key does not support the signature algorithm %s", alg)
	}
	if !x25519.Verify(x25519.PublicKey(v), payload, signature) {
		return errors.New("failed to verify XEdDSA signature")
	}
	return nil
}
