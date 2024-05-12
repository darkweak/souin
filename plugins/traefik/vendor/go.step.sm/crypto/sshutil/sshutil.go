package sshutil

import (
	"crypto"
	"crypto/dsa" //nolint:staticcheck // support for DSA fingerprints
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"fmt"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// CryptoPublicKey returns the crypto.PublicKey version of an ssh.PublicKey or
// *agent.Key.
func CryptoPublicKey(pub interface{}) (crypto.PublicKey, error) {
	switch p := pub.(type) {
	case *ecdsa.PublicKey, *rsa.PublicKey, ed25519.PublicKey, *dsa.PublicKey:
		return pub, nil
	case ssh.CryptoPublicKey:
		return p.CryptoPublicKey(), nil
	case *agent.Key:
		sshPub, err := ssh.ParsePublicKey(p.Blob)
		if err != nil {
			return nil, err
		}
		return CryptoPublicKey(sshPub)
	case ssh.PublicKey:
		// sk keys do not implement ssh.CryptoPublicKey
		return cryptoSKPublicKey(p)
	default:
		return nil, fmt.Errorf("unsupported public key type %T", pub)
	}
}

// cryptoSKPublicKey returns the crypto.PublicKey of an SSH SK public keys.
func cryptoSKPublicKey(pub ssh.PublicKey) (crypto.PublicKey, error) {
	switch pub.Type() {
	case "sk-ecdsa-sha2-nistp256@openssh.com":
		var w struct {
			Name        string
			ID          string
			Key         []byte
			Application string
		}
		if err := ssh.Unmarshal(pub.Marshal(), &w); err != nil {
			return nil, err
		}
		key := new(ecdsa.PublicKey)
		key.Curve = elliptic.P256()
		key.X, key.Y = elliptic.Unmarshal(key.Curve, w.Key)
		if key.X == nil || key.Y == nil {
			return nil, fmt.Errorf("invalid curve point")
		}
		return key, nil
	case "sk-ssh-ed25519@openssh.com":
		var w struct {
			Name        string
			KeyBytes    []byte
			Application string
		}
		if err := ssh.Unmarshal(pub.Marshal(), &w); err != nil {
			return nil, err
		}
		if l := len(w.KeyBytes); l != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid size %d for Ed25519 public key", l)
		}
		return ed25519.PublicKey(w.KeyBytes), nil
	default:
		return nil, fmt.Errorf("unsupported public key type %s", pub.Type())
	}
}
