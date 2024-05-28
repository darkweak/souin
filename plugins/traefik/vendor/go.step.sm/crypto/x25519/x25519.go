package x25519

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"io"
	"strconv"

	"filippo.io/edwards25519"
	"filippo.io/edwards25519/field"
	"golang.org/x/crypto/curve25519"
)

const (
	// PrivateKeySize is the size in bytes of a X25519 private key.
	PrivateKeySize = 32

	// PublicKeySize is the size in bytes of a X25519 public key.
	PublicKeySize = 32

	SignatureSize = 64
)

var one = (&field.Element{}).One()

// PrivateKey is the type used to represent a X25519 private key.
type PrivateKey []byte

// PublicKey is the type used to represent a X25519 public key.
type PublicKey []byte

// GenerateKey generates a public/private key pair using entropy from rand.
func GenerateKey(rand io.Reader) (PublicKey, PrivateKey, error) {
	priv := make([]byte, PrivateKeySize)
	if _, err := io.ReadFull(rand, priv); err != nil {
		return nil, nil, err
	}

	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	return pub, priv, err
}

// ToEd25519 converts the public key p into a ed25519 key.
//
// (x, y) = (sqrt(-486664)*u/v, (u-1)/(u+1))
func (p PublicKey) ToEd25519() (ed25519.PublicKey, error) {
	a, err := convertMont(p)
	if err != nil {
		return nil, err
	}
	return a.Bytes(), nil
}

// Equal reports whether p and x have the same value.
func (p PublicKey) Equal(x crypto.PublicKey) bool {
	xx, ok := x.(PublicKey)
	if !ok {
		return false
	}
	return bytes.Equal(p, xx)
}

// Public returns the public key using scalar multiplication (scalar * point)
// using the Curve25519 basepoint. It will return nil if the private key is not
// a valid one.
func (p PrivateKey) Public() crypto.PublicKey {
	pub, _ := p.PublicKey()
	return pub
}

// Equal reports whether p and x have the same value.
func (p PrivateKey) Equal(x crypto.PrivateKey) bool {
	xx, ok := x.(PrivateKey)
	if !ok {
		return false
	}
	return bytes.Equal(p, xx)
}

// Public returns the public key using scalar multiplication (scalar * point)
// using the Curve25519 basepoint.
func (p PrivateKey) PublicKey() (PublicKey, error) {
	pub, err := curve25519.X25519(p, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

// SharedKey returns the result of the scalar multiplication (scalar * point),
// using the PrivateKey as the scalar value and the given key as the point. Both
// scalar and point must be slices of 32 bytes.
func (p PrivateKey) SharedKey(peerPublicKey []byte) ([]byte, error) {
	sharedKey, err := curve25519.X25519(p, peerPublicKey)
	if err != nil {
		return nil, err
	}
	return sharedKey, nil
}

// Sign signs the given message with the private key p and returns a signature.
//
// It implements the XEdDSA sign method defined in
// https://signal.org/docs/specifications/xeddsa/#xeddsa
//
// XEdDSA performs two passes over messages to be signed and therefore cannot
// handle pre-hashed messages. Thus opts.HashFunc() must return zero to indicate
// the message hasn't been hashed. This can be achieved by passing
// crypto.Hash(0) as the value for opts.
func (p PrivateKey) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("x25519: cannot sign hashed message")
	}

	return Sign(rand, p, message)
}

// Sign signs the message with privateKey and returns a signature. It will panic
// if len(privateKey) is not PrivateKeySize.
//
// It implements the XEdDSA sign method defined in
// https://signal.org/docs/specifications/xeddsa/#xeddsa
//
//	xeddsa_sign(k, M, Z):
//	    A, a = calculate_key_pair(k)
//	    r = hash1(a || M || Z) (mod q)
//	    R = rB
//	    h = hash(R || A || M) (mod q)
//	    s = r + ha (mod q)
//	    return R || s
func Sign(rand io.Reader, p PrivateKey, message []byte) (signature []byte, err error) {
	if l := len(p); l != PrivateKeySize {
		panic("x25519: bad private key length: " + strconv.Itoa(l))
	}

	pub, priv, err := p.calculateKeyPair()
	if err != nil {
		return nil, err
	}

	random := make([]byte, 64)
	if _, err := io.ReadFull(rand, random); err != nil {
		return nil, err
	}

	// Using same prefix in libsignal-protocol-c implementation, but can be any
	// 32 byte prefix. Golang's ed25519 implementation uses:
	//
	//   ph := sha512.Sum512(a.Bytes())
	//   prefix := ph[32:]
	prefix := [32]byte{
		0xFE, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}

	rh := sha512.New()
	rh.Write(prefix[:])
	rh.Write(priv.Bytes())
	rh.Write(message)
	rh.Write(random)
	rDigest := make([]byte, 0, sha512.Size)
	rDigest = rh.Sum(rDigest)

	r, err := edwards25519.NewScalar().SetUniformBytes(rDigest)
	if err != nil {
		return nil, err
	}

	R := (&edwards25519.Point{}).ScalarBaseMult(r) //nolint:gocritic // variable names match crypto formulae docs

	hh := sha512.New()
	hh.Write(R.Bytes())
	hh.Write(pub)
	hh.Write(message)
	hDigest := make([]byte, 0, sha512.Size)
	hDigest = hh.Sum(hDigest)
	h, err := edwards25519.NewScalar().SetUniformBytes(hDigest)
	if err != nil {
		return nil, err
	}

	s := (&edwards25519.Scalar{}).Add(r, h.Multiply(h, priv))

	sig := make([]byte, 64)
	copy(sig[:32], R.Bytes())
	copy(sig[32:], s.Bytes())
	return sig, nil
}

// Verify reports whether sig is a valid signature of message by publicKey. It
// will panic if len(publicKey) is not PublicKeySize.
//
// It implements the XEdDSA verify method defined in
// https://signal.org/docs/specifications/xeddsa/#xeddsa
//
//	xeddsa_verify(u, M, (R || s)):
//	    if u >= p or R.y >= 2|p| or s >= 2|q|:
//	        return false
//	    A = convert_mont(u)
//	    if not on_curve(A):
//	        return false
//	    h = hash(R || A || M) (mod q)
//	    Rcheck = sB - hA
//	    if bytes_equal(R, Rcheck):
//	        return true
//	    return false
func Verify(publicKey PublicKey, message, sig []byte) bool {
	// The following code should be equivalent to:
	//
	//   pub, err := publicKey.ToEd25519()
	//   if err != nil {
	//       return false
	//   }
	//   return ed25519.Verify(pub, message, sig)

	if l := len(publicKey); l != PublicKeySize {
		panic("x25519: bad public key length: " + strconv.Itoa(l))
	}

	if len(sig) != SignatureSize || sig[63]&0xE0 != 0 {
		return false
	}

	a, err := convertMont(publicKey)

	if err != nil {
		return false
	}

	hh := sha512.New()
	hh.Write(sig[:32])
	hh.Write(a.Bytes())
	hh.Write(message)
	hDigest := make([]byte, 0, sha512.Size)
	hDigest = hh.Sum(hDigest)
	h, err := edwards25519.NewScalar().SetUniformBytes(hDigest)
	if err != nil {
		return false
	}

	s, err := edwards25519.NewScalar().SetCanonicalBytes(sig[32:])
	if err != nil {
		return false
	}

	minusA := (&edwards25519.Point{}).Negate(a)
	r := (&edwards25519.Point{}).VarTimeDoubleScalarBaseMult(h, minusA, s)
	return subtle.ConstantTimeCompare(sig[:32], r.Bytes()) == 1
}

// calculateKeyPair converts a Montgomery private key k to a twisted Edwards
// public key and private key (A, a) as defined in
// https://signal.org/docs/specifications/xeddsa/#elliptic-curve-conversions
//
//	calculate_key_pair(k):
//	    E = kB
//	    A.y = E.y
//	    A.s = 0
//	    if E.s == 1:
//	        a = -k (mod q)
//	    else:
//	        a = k (mod q)
//	    return A, a
func (p PrivateKey) calculateKeyPair() ([]byte, *edwards25519.Scalar, error) {
	var pA edwards25519.Point
	var sa edwards25519.Scalar

	k, err := (&edwards25519.Scalar{}).SetBytesWithClamping(p)
	if err != nil {
		return nil, nil, err
	}

	pub := pA.ScalarBaseMult(k).Bytes()
	signBit := (pub[31] & 0x80) >> 7

	if signBit == 1 {
		sa.Negate(k)
		// Set sig bit to 0
		pub[31] &= 0x7F
	} else {
		sa.Set(k)
	}

	return pub, &sa, nil
}

// convertMont converts from a Montgomery u-coordinate to a twisted Edwards
// point P, according to
// https://signal.org/docs/specifications/xeddsa/#elliptic-curve-conversions
//
//	convert_mont(u):
//	  umasked = u (mod 2|p|)
//	  P.y = u_to_y(umasked)
//	  P.s = 0
//	  return P
func convertMont(u PublicKey) (*edwards25519.Point, error) {
	um, err := (&field.Element{}).SetBytes(u)
	if err != nil {
		return nil, err
	}

	// y = (u - 1)/(u + 1)
	a := new(field.Element).Subtract(um, one)
	b := new(field.Element).Add(um, one)
	y := new(field.Element).Multiply(a, b.Invert(b)).Bytes()

	// Set sign to 0
	y[31] &= 0x7F

	return (&edwards25519.Point{}).SetBytes(y)
}
