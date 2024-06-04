package apiv1

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"
)

// ProtectionLevel specifies on some KMS how cryptographic operations are
// performed.
type ProtectionLevel int

const (
	// Protection level not specified.
	UnspecifiedProtectionLevel ProtectionLevel = iota
	// Crypto operations are performed in software.
	Software
	// Crypto operations are performed in a Hardware Security Module.
	HSM
)

// PINPolicy represents PIN requirements when signing or decrypting with an
// asymmetric key in a given slot. PINPolicy is used by the YubiKey KMS.
type PINPolicy int

// PIN policies supported by this package. The values must match the ones in
// github.com/go-piv/piv-go/piv.
//
// Caching for PINPolicyOnce isn't supported on YubiKey
// versions older than 4.3.0 due to issues with verifying if a PIN is needed.
// If specified, a PIN will be required for every operation.
const (
	PINPolicyNever PINPolicy = iota + 1
	PINPolicyOnce
	PINPolicyAlways
)

// TouchPolicy represents proof-of-presence requirements when signing or
// decrypting with asymmetric key in a given slot. TouchPolicy is used by the
// YubiKey KMS.
type TouchPolicy int

// Touch policies supported by this package. The values must match the ones in
// github.com/go-piv/piv-go/piv.
const (
	TouchPolicyNever TouchPolicy = iota + 1
	TouchPolicyAlways
	TouchPolicyCached
)

// String returns a string representation of p.
func (p ProtectionLevel) String() string {
	switch p {
	case UnspecifiedProtectionLevel:
		return "unspecified"
	case Software:
		return "software"
	case HSM:
		return "hsm"
	default:
		return fmt.Sprintf("unknown(%d)", p)
	}
}

// SignatureAlgorithm used for cryptographic signing.
type SignatureAlgorithm int

const (
	// Not specified.
	UnspecifiedSignAlgorithm SignatureAlgorithm = iota
	// RSASSA-PKCS1-v1_5 key and a SHA256 digest.
	SHA256WithRSA
	// RSASSA-PKCS1-v1_5 key and a SHA384 digest.
	SHA384WithRSA
	// RSASSA-PKCS1-v1_5 key and a SHA512 digest.
	SHA512WithRSA
	// RSASSA-PSS key with a SHA256 digest.
	SHA256WithRSAPSS
	// RSASSA-PSS key with a SHA384 digest.
	SHA384WithRSAPSS
	// RSASSA-PSS key with a SHA512 digest.
	SHA512WithRSAPSS
	// ECDSA on the NIST P-256 curve with a SHA256 digest.
	ECDSAWithSHA256
	// ECDSA on the NIST P-384 curve with a SHA384 digest.
	ECDSAWithSHA384
	// ECDSA on the NIST P-521 curve with a SHA512 digest.
	ECDSAWithSHA512
	// EdDSA on Curve25519 with a SHA512 digest.
	PureEd25519
)

// String returns a string representation of s.
func (s SignatureAlgorithm) String() string {
	switch s {
	case UnspecifiedSignAlgorithm:
		return "unspecified"
	case SHA256WithRSA:
		return "SHA256-RSA"
	case SHA384WithRSA:
		return "SHA384-RSA"
	case SHA512WithRSA:
		return "SHA512-RSA"
	case SHA256WithRSAPSS:
		return "SHA256-RSAPSS"
	case SHA384WithRSAPSS:
		return "SHA384-RSAPSS"
	case SHA512WithRSAPSS:
		return "SHA512-RSAPSS"
	case ECDSAWithSHA256:
		return "ECDSA-SHA256"
	case ECDSAWithSHA384:
		return "ECDSA-SHA384"
	case ECDSAWithSHA512:
		return "ECDSA-SHA512"
	case PureEd25519:
		return "Ed25519"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// GetPublicKeyRequest is the parameter used in the kms.GetPublicKey method.
type GetPublicKeyRequest struct {
	Name string
}

// CreateKeyRequest is the parameter used in the kms.CreateKey method.
type CreateKeyRequest struct {
	// Name represents the key name or label used to identify a key.
	//
	// Used by: awskms, cloudkms, azurekms, pkcs11, yubikey, tpmkms, mackms.
	Name string

	// SignatureAlgorithm represents the type of key to create.
	SignatureAlgorithm SignatureAlgorithm

	// Bits is the number of bits on RSA keys.
	Bits int

	// ProtectionLevel specifies how cryptographic operations are performed.
	// Used by: cloudkms, azurekms.
	ProtectionLevel ProtectionLevel

	// Extractable defines if the new key may be exported from the HSM under a
	// wrap key. On pkcs11 sets the CKA_EXTRACTABLE bit.
	//
	// Used by: pkcs11
	Extractable bool

	// PINPolicy defines PIN requirements when signing or decrypting with an
	// asymmetric key.
	//
	// Used by: yubikey
	PINPolicy PINPolicy

	// TouchPolicy represents proof-of-presence requirements when signing or
	// decrypting with asymmetric key in a given slot.
	//
	// Used by: yubikey
	TouchPolicy TouchPolicy

	// DestroyRetentionPeriod is the period of time that a key spends in a
	// destroy scheduled state before transitioning to destroyed.
	//
	// Used by: cloudkms
	DestroyRetentionPeriod time.Duration
}

// CreateKeyResponse is the response value of the kms.CreateKey method.
type CreateKeyResponse struct {
	Name      string
	PublicKey crypto.PublicKey
	// PrivateKey is only used by softkms
	PrivateKey          crypto.PrivateKey
	CreateSignerRequest CreateSignerRequest
}

// CreateSignerRequest is the parameter used in the kms.CreateSigner method.
type CreateSignerRequest struct {
	Signer           crypto.Signer
	SigningKey       string
	SigningKeyPEM    []byte
	TokenLabel       string
	PublicKey        string
	PublicKeyPEM     []byte
	Password         []byte
	PasswordPrompter PasswordPrompter
}

// CreateDecrypterRequest is the parameter used in the kms.Decrypt method.
type CreateDecrypterRequest struct {
	Decrypter        crypto.Decrypter
	DecryptionKey    string
	DecryptionKeyPEM []byte
	Password         []byte
	PasswordPrompter PasswordPrompter
}

// LoadCertificateRequest is the parameter used in the LoadCertificate method of
// a CertificateManager.
type LoadCertificateRequest struct {
	Name string
}

// LoadCertificateChainRequest is the parameter used in the LoadCertificateChain method of
// a CertificateChainManager. It's an alias for LoadCertificateRequest.
type LoadCertificateChainRequest LoadCertificateRequest

// StoreCertificateRequest is the parameter used in the StoreCertificate method
// of a CertificateManager.
type StoreCertificateRequest struct {
	Name        string
	Certificate *x509.Certificate

	// Extractable defines if the new certificate may be exported from the HSM
	// under a wrap key. On pkcs11 sets the CKA_EXTRACTABLE bit.
	//
	// Used by: pkcs11
	Extractable bool
}

// StoreCertificateChainRequest is the parameter used in the StoreCertificateChain method
// of a CertificateChainManager.
type StoreCertificateChainRequest struct {
	Name             string
	CertificateChain []*x509.Certificate
}

// CreateAttestationRequest is the parameter used in the kms.CreateAttestation
// method.
//
// # Experimental
//
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later
// release.
type CreateAttestationRequest struct {
	Name              string
	AttestationClient AttestationClient // TODO(hs): a better name; Attestor perhaps, but that's already taken
}

// AttestationClient is an interface that provides a pluggable method for
// attesting Attestation Keys (AKs).
type AttestationClient interface {
	Attest(context.Context) ([]*x509.Certificate, error)
}

// CertificationParameters encapsulates the inputs for certifying an application key.
// Only TPM 2.0 is supported at this point.
//
// This struct was copied from github.com/google/go-attestation, preventing an
// additional dependency in this package.
type CertificationParameters struct {
	// Public represents the key's canonical encoding (a TPMT_PUBLIC structure).
	// It includes the public key and signing parameters.
	Public []byte
	// CreateData represents the properties of a TPM 2.0 key. It is encoded
	// as a TPMS_CREATION_DATA structure.
	CreateData []byte
	// CreateAttestation represents an assertion as to the details of the key.
	// It is encoded as a TPMS_ATTEST structure.
	CreateAttestation []byte
	// CreateSignature represents a signature of the CreateAttestation structure.
	// It is encoded as a TPMT_SIGNATURE structure.
	CreateSignature []byte
}

// CreateAttestationResponse is the response value of the kms.CreateAttestation
// method.
//
// If a non-empty CertificateChain is returned, the first x509.Certificate is
// the same as the one in the Certificate property.
//
// When an attestation is created for a TPM key, the CertificationParameters
// property will have a record of the certification parameters at the time of
// key attestation.
//
// # Experimental
//
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later
// release.
type CreateAttestationResponse struct {
	Certificate             *x509.Certificate
	CertificateChain        []*x509.Certificate
	PublicKey               crypto.PublicKey
	CertificationParameters *CertificationParameters
	PermanentIdentifier     string
}

// DeleteKeyRequest is the parameter used in the kms.DeleteKey method.
//
// # Experimental
//
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later
// release.
type DeleteKeyRequest struct {
	Name string
}

// DeleteCertificateRequest is the parameter used in the kms.DeleteCertificate
// method.
//
// # Experimental
//
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later
// release.
type DeleteCertificateRequest struct {
	Name string
}
