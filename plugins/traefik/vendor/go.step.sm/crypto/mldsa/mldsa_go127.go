// Package mldsa is a temporary package that allows this module to compile with
// Go 1.26, where crypto/mldsa is not available yet. On Go 1.27 and later it is
// a thin alias over crypto/mldsa; on older toolchains it provides stubs that
// report ML-DSA as unsupported.
//
// This package will disappear once the lowest supported Go version is 1.27.

//go:build go1.27

package mldsa

import (
	"crypto/mldsa"
)

// Supported reports whether ML-DSA is available in the current build. It is
// true when compiled with Go 1.27 or later.
const Supported = true

const (
	PrivateKeySize       = mldsa.PrivateKeySize
	MLDSA44PublicKeySize = mldsa.MLDSA44PublicKeySize
	MLDSA65PublicKeySize = mldsa.MLDSA65PublicKeySize
	MLDSA87PublicKeySize = mldsa.MLDSA87PublicKeySize
	MLDSA44SignatureSize = mldsa.MLDSA44SignatureSize
	MLDSA65SignatureSize = mldsa.MLDSA65SignatureSize
	MLDSA87SignatureSize = mldsa.MLDSA87SignatureSize
)

type (
	Options    = mldsa.Options
	Parameters = mldsa.Parameters
	PrivateKey = mldsa.PrivateKey
	PublicKey  = mldsa.PublicKey
)

var (
	GenerateKey   = mldsa.GenerateKey
	NewPrivateKey = mldsa.NewPrivateKey
	NewPublicKey  = mldsa.NewPublicKey
	Verify        = mldsa.Verify
	MLDSA44       = mldsa.MLDSA44
	MLDSA65       = mldsa.MLDSA65
	MLDSA87       = mldsa.MLDSA87
)
