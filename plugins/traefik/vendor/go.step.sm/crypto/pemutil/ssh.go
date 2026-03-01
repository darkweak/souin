// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pemutil

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"encoding/pem"
	"math/big"

	"github.com/pkg/errors"
	bcryptpbkdf "go.step.sm/crypto/internal/bcrypt_pbkdf"
	"go.step.sm/crypto/randutil"
	"golang.org/x/crypto/ssh"
)

const (
	sshMagic             = "openssh-key-v1\x00"
	sshDefaultKdf        = "bcrypt"
	sshDefaultCiphername = "aes256-ctr"
	sshDefaultKeyLength  = 32
	sshDefaultSaltLength = 16
	sshDefaultRounds     = 16
)

type openSSHPrivateKey struct {
	CipherName   string
	KdfName      string
	KdfOpts      string
	NumKeys      uint32
	PubKey       []byte
	PrivKeyBlock []byte
}

type openSSHPrivateKeyBlock struct {
	Check1  uint32
	Check2  uint32
	Keytype string
	Rest    []byte `ssh:"rest"`
}

// ParseOpenSSHPrivateKey parses a private key in OpenSSH PEM format.
//
// Implemented based on the documentation at
// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.key
//
// This method is based on the implementation at
// https://github.com/golang/crypto/blob/master/ssh/keys.go
func ParseOpenSSHPrivateKey(pemBytes []byte, opts ...Options) (crypto.PrivateKey, error) {
	// Populate options
	ctx := newContext("PEM")
	if err := ctx.apply(opts); err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.Errorf("error decoding %s: not a valid PEM encoded block", ctx.filename)
	}

	if len(block.Bytes) < len(sshMagic) || string(block.Bytes[:len(sshMagic)]) != sshMagic {
		return nil, errors.New("invalid openssh private key format")
	}
	remaining := block.Bytes[len(sshMagic):]

	var w openSSHPrivateKey
	if err := ssh.Unmarshal(remaining, &w); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling private key")
	}

	var err error
	var key crypto.PrivateKey
	if w.KdfName != "none" || w.CipherName != "none" {
		password, err := ctx.promptPassword()
		if err != nil {
			return nil, err
		}
		key, err = ssh.ParseRawPrivateKeyWithPassphrase(pemBytes, password)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing private key")
		}
	} else {
		key, err = ssh.ParseRawPrivateKey(pemBytes)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing private key")
		}
	}

	// Convert *ed25519.PrivateKey to ed25519.PrivateKey:
	switch k := key.(type) {
	case *ed25519.PrivateKey:
		return *k, nil
	default:
		return k, nil
	}
}

// SerializeOpenSSHPrivateKey serialize a private key in the OpenSSH PEM format.
func SerializeOpenSSHPrivateKey(key crypto.PrivateKey, opts ...Options) (*pem.Block, error) {
	ctx := new(context)
	if err := ctx.apply(opts); err != nil {
		return nil, err
	}

	// Random check bytes.
	var check uint32
	if err := binary.Read(rand.Reader, binary.BigEndian, &check); err != nil {
		return nil, errors.Wrap(err, "error generating random check ")
	}

	w := openSSHPrivateKey{
		NumKeys: 1,
	}
	pk1 := openSSHPrivateKeyBlock{
		Check1: check,
		Check2: check,
	}

	password, err := ctx.promptEncryptPassword()
	if err != nil {
		return nil, err
	}

	var blockSize int
	if password == nil {
		w.CipherName = "none"
		w.KdfName = "none"
		blockSize = 8
	} else {
		w.CipherName = sshDefaultCiphername
		w.KdfName = sshDefaultKdf
		blockSize = aes.BlockSize
	}

	switch k := key.(type) {
	case *rsa.PrivateKey:
		e := new(big.Int).SetInt64(int64(k.PublicKey.E))
		// Marshal public key:
		// E and N are in reversed order in the public and private key.
		pubKey := struct {
			KeyType string
			E       *big.Int
			N       *big.Int
		}{
			ssh.KeyAlgoRSA,
			e, k.PublicKey.N,
		}
		w.PubKey = ssh.Marshal(pubKey)

		// Marshal private key.
		key := struct {
			N       *big.Int
			E       *big.Int
			D       *big.Int
			Iqmp    *big.Int
			P       *big.Int
			Q       *big.Int
			Comment string
		}{
			k.PublicKey.N, e,
			k.D, k.Precomputed.Qinv, k.Primes[0], k.Primes[1],
			ctx.comment,
		}
		pk1.Keytype = ssh.KeyAlgoRSA
		pk1.Rest = ssh.Marshal(key)
	case *ecdsa.PrivateKey:
		var curve, keyType string
		switch k.Curve.Params().Name {
		case "P-256":
			curve = "nistp256"
			keyType = ssh.KeyAlgoECDSA256
		case "P-384":
			curve = "nistp384"
			keyType = ssh.KeyAlgoECDSA384
		case "P-521":
			curve = "nistp521"
			keyType = ssh.KeyAlgoECDSA521
		default:
			return nil, errors.Errorf("error serializing key: unsupported curve %s", k.Curve.Params().Name)
		}

		p, err := k.PublicKey.ECDH()
		if err != nil {
			return nil, errors.Wrapf(err, "failed converting *ecdsa.PublicKey to *ecdh.PublicKey")
		}

		// Marshal public key.
		pubKey := struct {
			KeyType string
			Curve   string
			Pub     []byte
		}{
			keyType, curve, p.Bytes(),
		}
		w.PubKey = ssh.Marshal(pubKey)

		// Marshal private key.
		key := struct {
			Curve   string
			Pub     []byte
			D       *big.Int
			Comment string
		}{
			curve, p.Bytes(), k.D,
			ctx.comment,
		}
		pk1.Keytype = keyType
		pk1.Rest = ssh.Marshal(key)
	case ed25519.PrivateKey:
		pub := make([]byte, ed25519.PublicKeySize)
		priv := make([]byte, ed25519.PrivateKeySize)
		copy(pub, k[ed25519.PublicKeySize:])
		copy(priv, k)

		// Marshal public key.
		pubKey := struct {
			KeyType string
			Pub     []byte
		}{
			ssh.KeyAlgoED25519, pub,
		}
		w.PubKey = ssh.Marshal(pubKey)

		// Marshal private key.
		key := struct {
			Pub     []byte
			Priv    []byte
			Comment string
		}{
			pub, priv,
			ctx.comment,
		}
		pk1.Keytype = ssh.KeyAlgoED25519
		pk1.Rest = ssh.Marshal(key)
	default:
		return nil, errors.Errorf("unsupported key type %T", k)
	}

	w.PrivKeyBlock = ssh.Marshal(pk1)

	// Add padding until the private key block matches the block size,
	// 16 with AES encryption, 8 without.
	for i, l := 0, len(w.PrivKeyBlock); (l+i)%blockSize != 0; i++ {
		w.PrivKeyBlock = append(w.PrivKeyBlock, byte(i+1))
	}

	if password != nil {
		// Create encryption key derivation the password.
		salt, err := randutil.Salt(sshDefaultSaltLength)
		if err != nil {
			return nil, err
		}
		kdfOpts := struct {
			Salt   []byte
			Rounds uint32
		}{salt, sshDefaultRounds}
		w.KdfOpts = string(ssh.Marshal(kdfOpts))

		// Derive key to encrypt the private key block.
		k, err := bcryptpbkdf.Key(password, salt, sshDefaultRounds, sshDefaultKeyLength+aes.BlockSize)
		if err != nil {
			return nil, errors.Wrap(err, "error deriving decryption key")
		}

		// Encrypt the private key using the derived secret.
		dst := make([]byte, len(w.PrivKeyBlock))
		iv := k[sshDefaultKeyLength : sshDefaultKeyLength+aes.BlockSize]
		block, err := aes.NewCipher(k[:sshDefaultKeyLength])
		if err != nil {
			return nil, errors.Wrap(err, "error creating cipher")
		}

		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(dst, w.PrivKeyBlock)
		w.PrivKeyBlock = dst
	}

	b := ssh.Marshal(w)
	block := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: append([]byte(sshMagic), b...),
	}

	if ctx.filename != "" {
		if err := WriteFile(ctx.filename, pem.EncodeToMemory(block), ctx.perm); err != nil {
			return nil, err
		}
	}

	return block, nil
}
