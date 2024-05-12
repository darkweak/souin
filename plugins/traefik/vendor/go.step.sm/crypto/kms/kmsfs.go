package kms

import (
	"context"
	"fmt"
	"io/fs"

	"go.step.sm/crypto/kms/apiv1"
)

// FS adds a close method to the fs.FS interface. This new method allows to
// properly close the underlying KMS.
type FS interface {
	fs.FS
	Close() error
}

type kmsfs struct {
	apiv1.KeyManager
}

func newFS(ctx context.Context, kmsuri string) (*kmsfs, error) {
	if kmsuri == "" {
		return &kmsfs{}, nil
	}
	km, err := loadKMS(ctx, kmsuri)
	if err != nil {
		return nil, err
	}
	return &kmsfs{KeyManager: km}, nil
}

func (f *kmsfs) Close() error {
	if f != nil && f.KeyManager != nil {
		return f.KeyManager.Close()
	}
	return nil
}

func (f *kmsfs) getKMS(kmsuri string) (apiv1.KeyManager, error) {
	if f.KeyManager == nil {
		return loadKMS(context.TODO(), kmsuri)
	}
	return f.KeyManager, nil
}

func loadKMS(ctx context.Context, kmsuri string) (apiv1.KeyManager, error) {
	return New(ctx, apiv1.Options{
		URI: kmsuri,
	})
}

func openError(name string, err error) *fs.PathError {
	return &fs.PathError{
		Path: name,
		Op:   "open",
		Err:  err,
	}
}

// certFS implements an io/fs to load certificates from a KMS.
type certFS struct {
	*kmsfs
}

// CertFS creates a new io/fs with the given KMS URI.
func CertFS(ctx context.Context, kmsuri string) (FS, error) {
	km, err := newFS(ctx, kmsuri)
	if err != nil {
		return nil, err
	}
	_, ok := km.KeyManager.(apiv1.CertificateManager)
	if !ok {
		return nil, fmt.Errorf("%s does not implement a CertificateManager", kmsuri)
	}
	return &certFS{kmsfs: km}, nil
}

// Open returns a file representing a certificate in an KMS.
func (f *certFS) Open(name string) (fs.File, error) {
	km, err := f.getKMS(name)
	if err != nil {
		return nil, openError(name, err)
	}
	cert, err := km.(apiv1.CertificateManager).LoadCertificate(&apiv1.LoadCertificateRequest{
		Name: name,
	})
	if err != nil {
		return nil, openError(name, err)
	}
	return &object{
		Path:   name,
		Object: cert,
	}, nil
}

// keyFS implements an io/fs to load public keys from a KMS.
type keyFS struct {
	*kmsfs
}

// KeyFS creates a new KeyFS with the given KMS URI.
func KeyFS(ctx context.Context, kmsuri string) (FS, error) {
	km, err := newFS(ctx, kmsuri)
	if err != nil {
		return nil, err
	}
	return &keyFS{kmsfs: km}, nil
}

// Open returns a file representing a public key in a KMS.
func (f *keyFS) Open(name string) (fs.File, error) {
	km, err := f.getKMS(name)
	if err != nil {
		return nil, openError(name, err)
	}
	// Attempt with a public key
	pub, err := km.GetPublicKey(&apiv1.GetPublicKeyRequest{
		Name: name,
	})
	if err != nil {
		return nil, openError(name, err)
	}
	return &object{
		Path:   name,
		Object: pub,
	}, nil
}
