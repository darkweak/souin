// Package pemutil implements utilities to parse keys and certificates. It also
// includes a method to serialize keys, X.509 certificates and certificate
// requests to PEM.
package pemutil

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	fileutils "go.step.sm/crypto/internal/utils/file"
	"go.step.sm/crypto/keyutil"
	"go.step.sm/crypto/x25519"
)

// DefaultEncCipher is the default algorithm used when encrypting sensitive
// data in the PEM format.
var DefaultEncCipher = x509.PEMCipherAES256

// PasswordPrompter defines the function signature for the PromptPassword
// callback.
type PasswordPrompter func(s string) ([]byte, error)

// FileWriter defines the function signature for the WriteFile callback.
type FileWriter func(filename string, data []byte, perm os.FileMode) error

// PromptPassword is a method used to prompt for a password to decode encrypted
// keys. If this method is not defined and the key or password are not passed,
// the parse of the key will fail.
var PromptPassword PasswordPrompter

// WriteFile is a method used to write a file, by default it uses a wrapper over
// ioutil.WriteFile, but it can be set to a custom method, that for example can
// check if a file exists and prompts the user if it should be overwritten.
var WriteFile FileWriter = fileutils.WriteFile

// PEMBlockHeader is the expected header for any PEM formatted block.
var PEMBlockHeader = []byte("-----BEGIN ")

// context add options to the pem methods.
type context struct {
	filename         string
	perm             os.FileMode
	password         []byte
	pkcs8            bool
	openSSH          bool
	comment          string
	firstBlock       bool
	passwordPrompt   string
	passwordPrompter PasswordPrompter
}

// newContext initializes the context with a filename.
func newContext(name string) *context {
	return &context{
		filename: name,
		perm:     0o600,
	}
}

// apply the context options and return the first error if exists.
func (c *context) apply(opts []Options) error {
	for _, fn := range opts {
		if err := fn(c); err != nil {
			return err
		}
	}
	return nil
}

// promptPassword returns the password or prompts for one.
func (c *context) promptPassword() ([]byte, error) {
	switch {
	case len(c.password) > 0:
		return c.password, nil
	case c.passwordPrompter != nil:
		return c.passwordPrompter(c.passwordPrompt)
	case PromptPassword != nil:
		return PromptPassword(fmt.Sprintf("Please enter the password to decrypt %s", c.filename))
	default:
		return nil, errors.Errorf("error decoding %s: key is password protected", c.filename)
	}
}

// promptEncryptPassword returns the password or prompts for one if
// WithPassword, WithPasswordFile or WithPasswordPrompt have been used. This
// method is used to encrypt keys, and it will only use the options passed, it
// will not use the global PromptPassword.
func (c *context) promptEncryptPassword() ([]byte, error) {
	switch {
	case len(c.password) > 0:
		return c.password, nil
	case c.passwordPrompter != nil:
		return c.passwordPrompter(c.passwordPrompt)
	default:
		return nil, nil
	}
}

// Options is the type to add attributes to the context.
type Options func(o *context) error

// withContext replaces the context with the given one.
func withContext(c *context) Options {
	return func(ctx *context) error {
		*ctx = *c
		return nil
	}
}

// WithFilename is a method that adds the given filename to the context.
func WithFilename(name string) Options {
	return func(ctx *context) error {
		ctx.filename = name
		// Default perm mode if not set
		if ctx.perm == 0 {
			ctx.perm = 0o600
		}
		return nil
	}
}

// ToFile is a method that adds the given filename and permissions to the
// context. It is used in the Serialize to store PEM in disk.
func ToFile(name string, perm os.FileMode) Options {
	return func(ctx *context) error {
		ctx.filename = name
		ctx.perm = perm
		return nil
	}
}

// WithPassword is a method that adds the given password to the context.
func WithPassword(pass []byte) Options {
	return func(ctx *context) error {
		ctx.password = pass
		return nil
	}
}

// WithPasswordFile is a method that adds the password in a file to the context.
func WithPasswordFile(filename string) Options {
	return func(ctx *context) error {
		b, err := fileutils.ReadPasswordFromFile(filename)
		if err != nil {
			return err
		}
		ctx.password = b
		return nil
	}
}

// WithMinLengthPasswordFile is a method that adds the password in a file to the
// context. If the password does not meet the minimum length requirement an
// error is returned. If minimum length input is <=0 then the requirement is
// ignored.
func WithMinLengthPasswordFile(filename string, minLength int) Options {
	return func(ctx *context) error {
		if err := WithPasswordFile(filename)(ctx); err != nil {
			return err
		}

		if minLength > 0 && len(ctx.password) < minLength {
			return fmt.Errorf("password does not meet minimum length requirement; must be at least %v characters", minLength)
		}
		return nil
	}
}

// WithPasswordPrompt ask the user for a password and adds it to the context.
func WithPasswordPrompt(prompt string, fn PasswordPrompter) Options {
	return func(ctx *context) error {
		ctx.passwordPrompt = prompt
		ctx.passwordPrompter = fn
		return nil
	}
}

// WithPKCS8 with v set to true returns an option used in the Serialize method
// to use the PKCS#8 encoding form on the private keys. With v set to false
// default form will be used.
func WithPKCS8(v bool) Options {
	return func(ctx *context) error {
		ctx.pkcs8 = v
		return nil
	}
}

// WithOpenSSH is an option used in the Serialize method to use OpenSSH encoding
// form on the private keys. With v set to false default form will be used.
func WithOpenSSH(v bool) Options {
	return func(ctx *context) error {
		ctx.openSSH = v
		return nil
	}
}

// WithComment is an option used in the Serialize method to add a comment in the
// OpenSSH private keys. WithOpenSSH must be set to true too.
func WithComment(comment string) Options {
	return func(ctx *context) error {
		ctx.comment = comment
		return nil
	}
}

// WithFirstBlock will avoid failing if a PEM contains more than one block or
// certificate and it will only look at the first.
func WithFirstBlock() Options {
	return func(ctx *context) error {
		ctx.firstBlock = true
		return nil
	}
}

// ParseCertificate extracts the first certificate from the given pem.
func ParseCertificate(pemData []byte) (*x509.Certificate, error) {
	var block *pem.Block
	for len(pemData) > 0 {
		block, pemData = pem.Decode(pemData)
		if block == nil {
			return nil, errors.New("error decoding pem block")
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing certificate")
		}
		return cert, nil
	}

	return nil, errors.New("error parsing certificate: no certificate found")
}

// ParseCertificateBundle returns a list of *x509.Certificate parsed from
// the given bytes.
//
// - supports PEM and DER certificate formats
//   - If a DER-formatted file is given only one certificate will be returned.
func ParseCertificateBundle(data []byte) ([]*x509.Certificate, error) {
	var err error

	// PEM format
	if bytes.Contains(data, PEMBlockHeader) {
		var block *pem.Block
		var bundle []*x509.Certificate
		for len(data) > 0 {
			block, data = pem.Decode(data)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
				continue
			}
			var crt *x509.Certificate
			crt, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, &InvalidPEMError{
					Err:  err,
					Type: PEMTypeCertificate,
				}
			}
			bundle = append(bundle, crt)
		}
		if len(bundle) == 0 {
			return nil, &InvalidPEMError{
				Type: PEMTypeCertificate,
			}
		}
		return bundle, nil
	}

	// DER format (binary)
	crt, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, &InvalidPEMError{
			Message: fmt.Sprintf("error parsing certificate as DER format: %v", err),
			Type:    PEMTypeCertificate,
		}
	}
	return []*x509.Certificate{crt}, nil
}

// ParseCertificateRequest extracts the first *x509.CertificateRequest
// from the given data.
//
// - supports PEM and DER certificate formats
//   - If a DER-formatted file is given only one certificate will be returned.
func ParseCertificateRequest(data []byte) (*x509.CertificateRequest, error) {
	// PEM format
	if bytes.Contains(data, PEMBlockHeader) {
		var block *pem.Block
		for len(data) > 0 {
			block, data = pem.Decode(data)
			if block == nil {
				break
			}
			if !strings.HasSuffix(block.Type, "CERTIFICATE REQUEST") {
				continue
			}
			csr, err := x509.ParseCertificateRequest(block.Bytes)
			if err != nil {
				return nil, &InvalidPEMError{
					Type: PEMTypeCertificateRequest,
					Err:  err,
				}
			}

			return csr, nil
		}
	}

	// DER format (binary)
	csr, err := x509.ParseCertificateRequest(data)
	if err != nil {
		return nil, &InvalidPEMError{
			Message: fmt.Sprintf("error parsing certificate request as DER format: %v", err),
			Type:    PEMTypeCertificateRequest,
		}
	}
	return csr, nil
}

// PEMType represents a PEM block type. (e.g., CERTIFICATE, CERTIFICATE REQUEST, etc.)
type PEMType int

func (pt PEMType) String() string {
	switch pt {
	case PEMTypeCertificate:
		return "certificate"
	case PEMTypeCertificateRequest:
		return "certificate request"
	default:
		return "undefined"
	}
}

const (
	// PEMTypeUndefined undefined
	PEMTypeUndefined = iota
	// PEMTypeCertificate CERTIFICATE
	PEMTypeCertificate
	// PEMTypeCertificateRequest CERTIFICATE REQUEST
	PEMTypeCertificateRequest
)

// InvalidPEMError represents an error that occurs when parsing a file with
// PEM encoded data.
type InvalidPEMError struct {
	Type    PEMType
	File    string
	Message string
	Err     error
}

func (e *InvalidPEMError) Error() string {
	switch {
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return fmt.Sprintf("error decoding PEM data: %v", e.Err)
	default:
		if e.Type == PEMTypeUndefined {
			return "does not contain valid PEM encoded data"
		}
		return fmt.Sprintf("does not contain a valid PEM encoded %s", e.Type)
	}
}

func (e *InvalidPEMError) Unwrap() error {
	return e.Err
}

// ReadCertificate returns a *x509.Certificate from the given filename. It
// supports certificates formats PEM and DER.
func ReadCertificate(filename string, opts ...Options) (*x509.Certificate, error) {
	// Populate options
	ctx := newContext(filename)
	if err := ctx.apply(opts); err != nil {
		return nil, err
	}

	bundle, err := ReadCertificateBundle(filename)
	switch {
	case err != nil:
		return nil, err
	case len(bundle) == 0:
		return nil, errors.Errorf("file %s does not contain a valid PEM or DER formatted certificate", filename)
	case len(bundle) > 1 && !ctx.firstBlock:
		return nil, errors.Errorf("error decoding %s: contains more than one PEM encoded block", filename)
	default:
		return bundle[0], nil
	}
}

// ReadCertificateBundle reads the given filename and returns a list of
// *x509.Certificate.
//
// - supports PEM and DER certificate formats
//   - If a DER-formatted file is given only one certificate will be returned.
func ReadCertificateBundle(filename string) ([]*x509.Certificate, error) {
	b, err := fileutils.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	bundle, err := ParseCertificateBundle(b)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", filename, err)
	}
	return bundle, nil
}

// ReadCertificateRequest reads the given filename and returns a
// *x509.CertificateRequest.
//
// - supports PEM and DER Certificate formats.
// - supports reading from STDIN with filename `-`.
func ReadCertificateRequest(filename string) (*x509.CertificateRequest, error) {
	b, err := fileutils.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cr, err := ParseCertificateRequest(b)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", filename, err)
	}
	return cr, nil
}

// Parse returns the key or certificate PEM-encoded in the given bytes.
func Parse(b []byte, opts ...Options) (interface{}, error) {
	// Populate options
	ctx := newContext("PEM")
	if err := ctx.apply(opts); err != nil {
		return nil, err
	}

	block, rest := pem.Decode(b)
	switch {
	case block == nil:
		return nil, errors.Errorf("error decoding %s: not a valid PEM encoded block", ctx.filename)
	case len(bytes.TrimSpace(rest)) > 0 && !ctx.firstBlock:
		return nil, errors.Errorf("error decoding %s: contains more than one PEM encoded block", ctx.filename)
	}

	// PEM is encrypted: ask for password
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" || block.Type == "ENCRYPTED PRIVATE KEY" {
		pass, err := ctx.promptPassword()
		if err != nil {
			return nil, err
		}

		block.Bytes, err = DecryptPEMBlock(block, pass)
		if err != nil {
			return nil, errors.Wrapf(err, "error decrypting %s", ctx.filename)
		}
	}

	switch block.Type {
	case "PUBLIC KEY":
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		return pub, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "RSA PRIVATE KEY":
		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		return priv, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "EC PRIVATE KEY":
		priv, err := x509.ParseECPrivateKey(block.Bytes)
		return priv, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "PRIVATE KEY", "ENCRYPTED PRIVATE KEY":
		priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		return priv, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "OPENSSH PRIVATE KEY":
		priv, err := ParseOpenSSHPrivateKey(b, withContext(ctx))
		return priv, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "CERTIFICATE":
		crt, err := x509.ParseCertificate(block.Bytes)
		return crt, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "CERTIFICATE REQUEST", "NEW CERTIFICATE REQUEST":
		csr, err := x509.ParseCertificateRequest(block.Bytes)
		return csr, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "ENCRYPTED COSIGN PRIVATE KEY":
		pass, err := ctx.promptPassword()
		if err != nil {
			return nil, err
		}
		priv, err := ParseCosignPrivateKey(block.Bytes, pass)
		return priv, errors.Wrapf(err, "error parsing %s", ctx.filename)
	case "NEBULA X25519 PUBLIC KEY":
		if len(block.Bytes) != x25519.PublicKeySize {
			return nil, errors.Errorf("error parsing %s: key is not 32 bytes", ctx.filename)
		}
		return x25519.PublicKey(block.Bytes), nil
	case "NEBULA X25519 PRIVATE KEY":
		if len(block.Bytes) != x25519.PrivateKeySize {
			return nil, errors.Errorf("error parsing %s: key is not 32 bytes", ctx.filename)
		}
		return x25519.PrivateKey(block.Bytes), nil
	default:
		return nil, errors.Errorf("error decoding %s: contains an unexpected header '%s'", ctx.filename, block.Type)
	}
}

// ParseKey returns the key or the public key of a certificate or certificate
// signing request in the given PEM-encoded bytes.
func ParseKey(b []byte, opts ...Options) (interface{}, error) {
	k, err := Parse(b, opts...)
	if err != nil {
		return nil, err
	}
	return keyutil.ExtractKey(k)
}

// Read returns the key or certificate encoded in the given PEM file.
// If the file is encrypted it will ask for a password and it will try
// to decrypt it.
//
// Supported keys algorithms are RSA and EC. Supported standards for private
// keys are PKCS#1, PKCS#8, RFC5915 for EC, and base64-encoded DER for
// certificates and public keys.
func Read(filename string, opts ...Options) (interface{}, error) {
	b, err := fileutils.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// force given filename
	opts = append(opts, WithFilename(filename))
	return Parse(b, opts...)
}

// Serialize will serialize the input to a PEM formatted block and apply
// modifiers.
func Serialize(in interface{}, opts ...Options) (*pem.Block, error) {
	ctx := new(context)
	if err := ctx.apply(opts); err != nil {
		return nil, err
	}

	var p *pem.Block
	var isPrivateKey bool
	switch k := in.(type) {
	case *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey:
		b, err := x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		p = &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: b,
		}
	case *rsa.PrivateKey:
		isPrivateKey = true
		switch {
		case ctx.pkcs8:
			b, err := x509.MarshalPKCS8PrivateKey(k)
			if err != nil {
				return nil, err
			}
			p = &pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: b,
			}
		case ctx.openSSH:
			return SerializeOpenSSHPrivateKey(k, withContext(ctx))
		default:
			p = &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(k),
			}
		}
	case *ecdsa.PrivateKey:
		isPrivateKey = true
		switch {
		case ctx.pkcs8:
			b, err := x509.MarshalPKCS8PrivateKey(k)
			if err != nil {
				return nil, err
			}
			p = &pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: b,
			}
		case ctx.openSSH:
			return SerializeOpenSSHPrivateKey(k, withContext(ctx))
		default:
			b, err := x509.MarshalECPrivateKey(k)
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal private key")
			}
			p = &pem.Block{
				Type:  "EC PRIVATE KEY",
				Bytes: b,
			}
		}
	case ed25519.PrivateKey:
		isPrivateKey = true
		switch {
		case !ctx.pkcs8 && ctx.openSSH:
			return SerializeOpenSSHPrivateKey(k, withContext(ctx))
		default: // Ed25519 keys will use pkcs8 by default
			ctx.pkcs8 = true
			b, err := x509.MarshalPKCS8PrivateKey(k)
			if err != nil {
				return nil, err
			}
			p = &pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: b,
			}
		}
	case *x509.Certificate:
		p = &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: k.Raw,
		}
	case *x509.CertificateRequest:
		p = &pem.Block{
			Type:  "CERTIFICATE REQUEST",
			Bytes: k.Raw,
		}
	default:
		return nil, errors.Errorf("cannot serialize type '%T', value '%v'", k, k)
	}

	if isPrivateKey {
		// Request password if needed.
		password, err := ctx.promptEncryptPassword()
		if err != nil {
			return nil, err
		}

		// Apply options on the PEM blocks.
		if password != nil {
			if ctx.pkcs8 {
				var err error
				p, err = EncryptPKCS8PrivateKey(rand.Reader, p.Bytes, password, DefaultEncCipher)
				if err != nil {
					return nil, err
				}
			} else {
				var err error
				p, err = x509.EncryptPEMBlock(rand.Reader, p.Type, p.Bytes, password, DefaultEncCipher) //nolint:staticcheck // support legacy use cases
				if err != nil {
					return nil, errors.Wrap(err, "failed to serialize to PEM")
				}
			}
		}
	}

	if ctx.filename != "" {
		if err := WriteFile(ctx.filename, pem.EncodeToMemory(p), ctx.perm); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// ParseDER parses the given DER-encoded bytes and results the public or private
// key encoded.
func ParseDER(b []byte) (interface{}, error) {
	// Try private keys
	key, err := x509.ParsePKCS8PrivateKey(b)
	if err != nil {
		if key, err = x509.ParseECPrivateKey(b); err != nil {
			key, err = x509.ParsePKCS1PrivateKey(b)
		}
	}

	// Try public key
	if err != nil {
		if key, err = x509.ParsePKIXPublicKey(b); err != nil {
			if key, err = x509.ParsePKCS1PublicKey(b); err != nil {
				return nil, errors.New("error decoding DER; bad format")
			}
		}
	}

	return key, nil
}

// ParseSSH parses parses a public key from an authorized_keys file used in
// OpenSSH according to the sshd(8) manual page.
func ParseSSH(b []byte) (interface{}, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey(b)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing OpenSSH key")
	}

	if cert, ok := key.(*ssh.Certificate); ok {
		key = cert.Key
	}

	switch key.Type() {
	case ssh.KeyAlgoRSA:
		var w struct {
			Name string
			E    *big.Int
			N    *big.Int
		}
		if err := ssh.Unmarshal(key.Marshal(), &w); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling key")
		}

		if w.E.BitLen() > 24 {
			return nil, errors.New("error unmarshaling key: exponent too large")
		}
		e := w.E.Int64()
		if e < 3 || e&1 == 0 {
			return nil, errors.New("error unmarshaling key: incorrect exponent")
		}

		key := new(rsa.PublicKey)
		key.E = int(e)
		key.N = w.N
		return key, nil

	case ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521:
		var w struct {
			Name     string
			ID       string
			KeyBytes []byte
		}
		if err := ssh.Unmarshal(key.Marshal(), &w); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling key")
		}

		var c ecdh.Curve
		switch w.Name {
		case ssh.KeyAlgoECDSA256:
			c = ecdh.P256()
		case ssh.KeyAlgoECDSA384:
			c = ecdh.P384()
		case ssh.KeyAlgoECDSA521:
			c = ecdh.P521()
		default:
			return nil, errors.Errorf("unsupported ecdsa curve %s", w.Name)
		}

		var p *ecdh.PublicKey
		if p, err = c.NewPublicKey(w.KeyBytes); err != nil {
			return nil, errors.Wrapf(err, "failed decoding %s key", w.Name)
		}

		// convert ECDH public key to ECDSA public key to keep
		// the returned type backwards compatible.
		rawKey := p.Bytes()
		switch p.Curve() {
		case ecdh.P256():
			return &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     big.NewInt(0).SetBytes(rawKey[1:33]),
				Y:     big.NewInt(0).SetBytes(rawKey[33:]),
			}, nil
		case ecdh.P384():
			return &ecdsa.PublicKey{
				Curve: elliptic.P384(),
				X:     big.NewInt(0).SetBytes(rawKey[1:49]),
				Y:     big.NewInt(0).SetBytes(rawKey[49:]),
			}, nil
		case ecdh.P521():
			return &ecdsa.PublicKey{
				Curve: elliptic.P521(),
				X:     big.NewInt(0).SetBytes(rawKey[1:67]),
				Y:     big.NewInt(0).SetBytes(rawKey[67:]),
			}, nil
		default:
			return nil, errors.New("cannot convert non-NIST *ecdh.PublicKey to *ecdsa.PublicKey")
		}
	case ssh.KeyAlgoED25519:
		var w struct {
			Name     string
			KeyBytes []byte
		}
		if err := ssh.Unmarshal(key.Marshal(), &w); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling key")
		}
		return ed25519.PublicKey(w.KeyBytes), nil
	case ssh.InsecureKeyAlgoDSA: //nolint:staticcheck // just using the constant; no dependent logic
		return nil, errors.Errorf("DSA keys not supported")
	default:
		return nil, errors.Errorf("unsupported key type %T", key)
	}
}

// BundleCertificate adds PEM-encoded certificates to a PEM-encoded certificate
// bundle if not already in the bundle.
func BundleCertificate(bundlePEM []byte, certsPEM ...[]byte) ([]byte, bool, error) {
	bundle, err := ParseCertificateBundle(bundlePEM)
	if err != nil {
		return nil, false, fmt.Errorf("invalid bundle: %w", err)
	}

	sums := make(map[[sha256.Size224]byte]bool, len(bundle)+len(certsPEM))
	for i := range bundle {
		sums[sha256.Sum224(bundle[i].Raw)] = true
	}

	modified := false

	for i := range certsPEM {
		cert, err := ParseCertificate(certsPEM[i])
		if err != nil {
			return nil, false, fmt.Errorf("invalid certificate %d: %w", i, err)
		}
		certSum := sha256.Sum224(cert.Raw)
		if sums[certSum] {
			continue
		}
		sums[certSum] = true
		bundlePEM = append(bundlePEM, certsPEM[i]...)
		modified = true
	}

	return bundlePEM, modified, nil
}

// UnbundleCertificate removes PEM-encoded certificates from a PEM-encoded
// certificate bundle.
func UnbundleCertificate(bundlePEM []byte, certsPEM ...[]byte) ([]byte, bool, error) {
	if len(certsPEM) == 0 {
		return bundlePEM, false, nil
	}
	drop := make(map[[sha256.Size224]byte]bool, len(certsPEM))
	for i := range certsPEM {
		certs, err := ParseCertificateBundle(certsPEM[i])
		if err != nil {
			return nil, false, fmt.Errorf("invalid certificate %d: %w", i, err)
		}
		for _, cert := range certs {
			drop[sha256.Sum224(cert.Raw)] = true
		}
	}

	var modified bool
	var keep []byte

	bundle, err := ParseCertificateBundle(bundlePEM)
	if err != nil {
		return nil, false, fmt.Errorf("invalid bundle: %w", err)
	}
	for _, cert := range bundle {
		sum := sha256.Sum224(cert.Raw)
		if drop[sum] {
			modified = true
			continue
		}
		keep = append(keep, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})...)
	}

	return keep, modified, nil
}
