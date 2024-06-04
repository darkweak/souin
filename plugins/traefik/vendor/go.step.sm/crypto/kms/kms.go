package kms

import (
	"context"

	"github.com/pkg/errors"
	"go.step.sm/crypto/kms/apiv1"

	// Enable default implementation
	"go.step.sm/crypto/kms/softkms"
)

// KeyManager is the interface implemented by all the KMS.
type KeyManager = apiv1.KeyManager

// CertificateManager is the interface implemented by the KMS that can load and
// store x509.Certificates.
type CertificateManager = apiv1.CertificateManager

// Attester is the interface implemented by the KMS that can respond with an
// attestation certificate or key.
//
// # Experimental
//
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later
// release.
type Attester = apiv1.Attester

// Options are the KMS options. They represent the kms object in the ca.json.
type Options = apiv1.Options

// Type represents the KMS type used.
type Type = apiv1.Type

// TypeOf returns the KMS type of the given uri.
var TypeOf = apiv1.TypeOf

// Default is the implementation of the default KMS.
var Default = &softkms.SoftKMS{}

// New initializes a new KMS from the given type.
func New(ctx context.Context, opts apiv1.Options) (KeyManager, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	typ, err := opts.GetType()
	if err != nil {
		return nil, err
	}
	fn, ok := apiv1.LoadKeyManagerNewFunc(typ)
	if !ok {
		return nil, errors.Errorf("unsupported kms type '%s'", typ)
	}
	return fn(ctx, opts)
}
