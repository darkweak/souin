package apiv1

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	"go.step.sm/crypto/kms/uri"
)

// KeyManager is the interface implemented by all the KMS.
type KeyManager interface {
	GetPublicKey(req *GetPublicKeyRequest) (crypto.PublicKey, error)
	CreateKey(req *CreateKeyRequest) (*CreateKeyResponse, error)
	CreateSigner(req *CreateSignerRequest) (crypto.Signer, error)
	Close() error
}

// Decrypter is an interface implemented by KMSes that are used
// in operations that require decryption
type Decrypter interface {
	CreateDecrypter(req *CreateDecrypterRequest) (crypto.Decrypter, error)
}

// CertificateManager is the interface implemented by the KMS that can load and
// store x509.Certificates.
type CertificateManager interface {
	LoadCertificate(req *LoadCertificateRequest) (*x509.Certificate, error)
	StoreCertificate(req *StoreCertificateRequest) error
}

// CertificateChainManager is the interface implemented by KMS implementations
// that can load certificate chains. The LoadCertificateChain method uses the
// same request object as the LoadCertificate method of the CertificateManager
// interfaces. When the LoadCertificateChain method is called, the certificate
// chain stored through the CertificateChain property in the StoreCertificateRequest
// will be returned, partially reusing the StoreCertificateRequest object.
type CertificateChainManager interface {
	LoadCertificateChain(req *LoadCertificateChainRequest) ([]*x509.Certificate, error)
	StoreCertificateChain(req *StoreCertificateChainRequest) error
}

// NameValidator is an interface that KeyManager can implement to validate a
// given name or URI.
type NameValidator interface {
	ValidateName(s string) error
}

// Attester is the interface implemented by the KMS that can respond with an
// attestation certificate or key.
//
// # Experimental
//
// Notice: This API is EXPERIMENTAL and may be changed or removed in a later
// release.
type Attester interface {
	CreateAttestation(req *CreateAttestationRequest) (*CreateAttestationResponse, error)
}

// NotImplementedError is the type of error returned if an operation is not
// implemented.
type NotImplementedError struct {
	Message string
}

func (e NotImplementedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "not implemented"
}

// AlreadyExistsError is the type of error returned if a key already exists. This
// is currently only implmented for pkcs11 and tpmkms.
type AlreadyExistsError struct {
	Message string
}

func (e AlreadyExistsError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "key already exists"
}

// Type represents the KMS type used.
type Type string

const (
	// DefaultKMS is a KMS implementation using software.
	DefaultKMS Type = ""
	// SoftKMS is a KMS implementation using software.
	SoftKMS Type = "softkms"
	// CloudKMS is a KMS implementation using Google's Cloud KMS.
	CloudKMS Type = "cloudkms"
	// AmazonKMS is a KMS implementation using Amazon AWS KMS.
	AmazonKMS Type = "awskms"
	// PKCS11 is a KMS implementation using the PKCS11 standard.
	PKCS11 Type = "pkcs11"
	// YubiKey is a KMS implementation using a YubiKey PIV.
	YubiKey Type = "yubikey"
	// SSHAgentKMS is a KMS implementation using ssh-agent to access keys.
	SSHAgentKMS Type = "sshagentkms"
	// AzureKMS is a KMS implementation using Azure Key Vault.
	AzureKMS Type = "azurekms"
	// CAPIKMS
	CAPIKMS Type = "capi"
	// TPMKMS
	TPMKMS Type = "tpmkms"
)

// Options are the KMS options. They represent the kms object in the ca.json.
type Options struct {
	// The type of the KMS to use.
	Type Type `json:"type"`

	// Path to the credentials file used in CloudKMS and AmazonKMS.
	CredentialsFile string `json:"credentialsFile,omitempty"`

	// URI is based on the PKCS #11 URI Scheme defined in
	// https://tools.ietf.org/html/rfc7512 and represents the configuration used
	// to connect to the KMS.
	//
	// Used by: pkcs11, tpmkms
	URI string `json:"uri,omitempty"`

	// Pin used to access the PKCS11 module. It can be defined in the URI using
	// the pin-value or pin-source properties.
	Pin string `json:"pin,omitempty"`

	// ManagementKey used in YubiKeys. Default management key is the hexadecimal
	// string 010203040506070801020304050607080102030405060708:
	//   []byte{
	//       0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	//       0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	//       0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	//   }
	ManagementKey string `json:"managementKey,omitempty"`

	// Region to use in AmazonKMS.
	Region string `json:"region,omitempty"`

	// Profile to use in AmazonKMS.
	Profile string `json:"profile,omitempty"`

	// StorageDirectory is the path to a directory to
	// store serialized TPM objects. Only used by the TPMKMS.
	StorageDirectory string `json:"storageDirectory,omitempty"`
}

// Validate checks the fields in Options.
func (o *Options) Validate() error {
	if o == nil {
		return nil
	}

	typ := strings.ToLower(string(o.Type))
	switch Type(typ) {
	case DefaultKMS, SoftKMS: // Go crypto based kms.
	case CloudKMS, AmazonKMS, AzureKMS: // Cloud based kms.
	case YubiKey, PKCS11, TPMKMS: // Hardware based kms.
	case SSHAgentKMS, CAPIKMS: // Others
	default:
		return fmt.Errorf("unsupported kms type %s", o.Type)
	}

	return nil
}

// GetType returns the type in the type property or the one present in the URI.
func (o *Options) GetType() (Type, error) {
	if o.Type != "" {
		return o.Type, nil
	}
	if o.URI != "" {
		u, err := uri.Parse(o.URI)
		if err != nil {
			return DefaultKMS, err
		}
		return Type(strings.ToLower(u.Scheme)), nil
	}
	return SoftKMS, nil
}

var ErrNonInteractivePasswordPrompt = errors.New("password required in non-interactive context")

var NonInteractivePasswordPrompter = func(s string) ([]byte, error) {
	return nil, ErrNonInteractivePasswordPrompt
}

type PasswordPrompter func(s string) ([]byte, error)
