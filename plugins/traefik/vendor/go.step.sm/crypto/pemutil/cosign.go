package pemutil

import (
	"crypto"
	"crypto/x509"
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

type cosignEnvelope struct {
	KDF        cosignKDF    `json:"kdf"`
	Cipher     cosignCipher `json:"cipher"`
	Ciphertext []byte       `json:"ciphertext"`
}

type cosignKDF struct {
	Name   string             `json:"name"`
	Params cosignScryptParams `json:"params"`
	Salt   []byte             `json:"salt"`
}

type cosignScryptParams struct {
	N int `json:"N"`
	R int `json:"r"`
	P int `json:"p"`
}

type cosignCipher struct {
	Name  string `json:"name"`
	Nonce []byte `json:"nonce"`
}

// ParseCosignPrivateKey returns the private key encoded using cosign envelope.
// If an incorrect password is detected an x509.IncorrectPasswordError is
// returned.
//
// Cosign keys are encrypted under a password using scrypt as a KDF and
// nacl/secretbox for encryption.
func ParseCosignPrivateKey(data, password []byte) (crypto.PrivateKey, error) {
	var env cosignEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling key")
	}
	if env.KDF.Name != "scrypt" {
		return nil, errors.Errorf("error parsing key: unsupported kdf %s", env.KDF.Name)
	}
	if env.Cipher.Name != "nacl/secretbox" {
		return nil, errors.Errorf("error parsing key: unsupported cipher %s", env.Cipher.Name)
	}
	if len(env.Cipher.Nonce) != 24 {
		return nil, errors.New("error parsing key: nonce must be 24 bytes long")
	}

	params := env.KDF.Params
	k, err := scrypt.Key(password, env.KDF.Salt, params.N, params.R, params.P, 32)
	if err != nil {
		return nil, errors.Wrap(err, "error generating key")
	}

	var nonce [24]byte
	var key [32]byte
	copy(nonce[:], env.Cipher.Nonce)
	copy(key[:], k)

	out, ok := secretbox.Open(nil, env.Ciphertext, &nonce, &key)
	if !ok {
		return nil, x509.IncorrectPasswordError
	}

	priv, err := x509.ParsePKCS8PrivateKey(out)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing pkcs8 key")
	}

	return priv, nil
}
