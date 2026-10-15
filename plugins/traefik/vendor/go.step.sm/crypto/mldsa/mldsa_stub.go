// Package mldsa is a temporary package that allows this module to compile with
// Go 1.26, where crypto/mldsa is not available yet. On Go 1.27 and later it is
// a thin alias over crypto/mldsa; on older toolchains it provides stubs that
// report ML-DSA as unsupported.
//
// This package will disappear once the lowest supported Go version is 1.27.

//go:build !go1.27

package mldsa

import (
	"crypto"
	"errors"
	"io"
)

var errNotSupported = errors.New("mldsa is not supported")

// Supported reports whether ML-DSA is available in the current build. It is
// false when compiled with a Go toolchain older than 1.27.
const Supported = false

const (
	PrivateKeySize       = 32
	MLDSA44PublicKeySize = 1312
	MLDSA65PublicKeySize = 1952
	MLDSA87PublicKeySize = 2592
	MLDSA44SignatureSize = 2420
	MLDSA65SignatureSize = 3309
	MLDSA87SignatureSize = 4627
)

type Options struct {
	Context string
}

func (o *Options) HashFunc() crypto.Hash {
	return 0
}

type Parameters struct{}

func MLDSA44() Parameters {
	return Parameters{}
}

func MLDSA65() Parameters {
	return Parameters{}
}

func MLDSA87() Parameters {
	return Parameters{}
}

func (p Parameters) PublicKeySize() int {
	return 0
}

func (p Parameters) SignatureSize() int {
	return 0
}

func (p Parameters) String() string {
	return ""
}

type PrivateKey struct{}

func GenerateKey(params Parameters) (*PrivateKey, error) {
	return nil, errNotSupported
}

func NewPrivateKey(params Parameters, seed []byte) (*PrivateKey, error) {
	return nil, errNotSupported
}

func (sk *PrivateKey) Bytes() []byte {
	return nil
}

func (sk *PrivateKey) Equal(x crypto.PrivateKey) bool {
	return false
}

func (sk *PrivateKey) Public() crypto.PublicKey {
	return (*PublicKey)(nil)
}

func (sk *PrivateKey) PublicKey() *PublicKey {
	return (*PublicKey)(nil)
}

func (sk *PrivateKey) Sign(_ io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return nil, errNotSupported
}

func (sk *PrivateKey) SignDeterministic(message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return nil, errNotSupported
}

type PublicKey struct{}

func NewPublicKey(params Parameters, encoding []byte) (*PublicKey, error) {
	return nil, errNotSupported
}

func (pk *PublicKey) Bytes() []byte {
	return nil
}

func (pk *PublicKey) Equal(x crypto.PublicKey) bool {
	return false
}

func (pk *PublicKey) Parameters() Parameters {
	return Parameters{}
}

func Verify(pk *PublicKey, message, signature []byte, opts *Options) error {
	return errNotSupported
}
