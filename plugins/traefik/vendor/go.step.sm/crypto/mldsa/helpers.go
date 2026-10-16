package mldsa

import (
	"crypto/sha3"
	"errors"

	"go.step.sm/crypto/internal/utils/convert"
)

var (
	errContextTooLong = errors.New("mldsa: context too long")
)

func PublicKeyHash(pub *PublicKey) [64]byte {
	h := sha3.NewSHAKE256()
	h.Write(pub.Bytes())
	var tr [64]byte
	h.Read(tr[:])
	return tr
}

func MessageHash(pub *PublicKey, msg []byte, opts *Options) ([64]byte, error) {
	if opts == nil {
		opts = &Options{}
	}

	tr := PublicKeyHash(pub)
	if len(opts.Context) > 255 {
		return [64]byte{}, errContextTooLong
	}

	h := sha3.NewSHAKE256()
	h.Write(tr[:])
	h.Write([]byte{0}) // ML-DSA / HashML-DSA domain separator
	h.Write([]byte{convert.MustUint8(len(opts.Context))})
	h.Write([]byte(opts.Context))
	h.Write(msg)
	var mu [64]byte
	h.Read(mu[:])
	return mu, nil
}
