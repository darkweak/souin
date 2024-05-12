package kms

import (
	"bytes"
	"encoding/pem"
	"io"
	"io/fs"
	"sync"
	"time"

	"go.step.sm/crypto/pemutil"
)

// object implements the fs.File and fs.FileMode interfaces.
type object struct {
	Path    string
	Object  interface{}
	once    sync.Once
	err     error
	pemData *bytes.Buffer
}

// FileMode implementation
func (o *object) Name() string       { return o.Path }
func (o *object) Size() int64        { return int64(o.pemData.Len()) }
func (o *object) Mode() fs.FileMode  { return 0400 }
func (o *object) ModTime() time.Time { return time.Time{} }
func (o *object) IsDir() bool        { return false }
func (o *object) Sys() interface{}   { return o.Object }

func (o *object) load() error {
	o.once.Do(func() {
		b, err := pemutil.Serialize(o.Object)
		if err != nil {
			o.err = &fs.PathError{
				Op:   "open",
				Path: o.Path,
				Err:  err,
			}
			return
		}
		o.pemData = bytes.NewBuffer(pem.EncodeToMemory(b))
	})
	return o.err
}

func (o *object) Stat() (fs.FileInfo, error) {
	if err := o.load(); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *object) Read(b []byte) (int, error) {
	if err := o.load(); err != nil {
		return 0, err
	}
	return o.pemData.Read(b)
}

func (o *object) Close() error {
	o.Object = nil
	o.pemData = nil
	if o.err == nil {
		o.err = io.EOF
		return nil
	}
	return o.err
}
