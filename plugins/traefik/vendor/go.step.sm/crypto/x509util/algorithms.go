package x509util

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"strings"

	"github.com/pkg/errors"
)

// List of signature algorithms.
const (
	MD2WithRSA       = "MD2-RSA"
	MD5WithRSA       = "MD5-RSA"
	SHA1WithRSA      = "SHA1-RSA"
	SHA256WithRSA    = "SHA256-RSA"
	SHA384WithRSA    = "SHA384-RSA"
	SHA512WithRSA    = "SHA512-RSA"
	DSAWithSHA1      = "DSA-SHA1"
	DSAWithSHA256    = "DSA-SHA256"
	ECDSAWithSHA1    = "ECDSA-SHA1"
	ECDSAWithSHA256  = "ECDSA-SHA256"
	ECDSAWithSHA384  = "ECDSA-SHA384"
	ECDSAWithSHA512  = "ECDSA-SHA512"
	SHA256WithRSAPSS = "SHA256-RSAPSS"
	SHA384WithRSAPSS = "SHA384-RSAPSS"
	SHA512WithRSAPSS = "SHA512-RSAPSS"
	PureEd25519      = "Ed25519"
)

var (
	oidSignatureMD2WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 2}
	oidSignatureMD5WithRSA      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 4}
	oidSignatureSHA1WithRSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5}
	oidSignatureSHA256WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidSignatureSHA384WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidSignatureSHA512WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
	oidSignatureRSAPSS          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
	oidSignatureDSAWithSHA1     = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 3}
	oidSignatureDSAWithSHA256   = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 2}
	oidSignatureECDSAWithSHA1   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 1}
	oidSignatureECDSAWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidSignatureECDSAWithSHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	oidSignatureECDSAWithSHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
	oidSignatureEd25519         = asn1.ObjectIdentifier{1, 3, 101, 112}

	// oidISOSignatureSHA1WithRSA means the same as oidSignatureSHA1WithRSA but
	// it's specified by ISO. Microsoft's makecert.exe has been known to produce
	// certificates with this OID.
	oidISOSignatureSHA1WithRSA = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 29}
)

var signatureAlgorithmMapping = []struct {
	name  string
	value x509.SignatureAlgorithm
	oid   asn1.ObjectIdentifier
	hash  crypto.Hash
}{
	{"", x509.UnknownSignatureAlgorithm, nil, crypto.Hash(0)},
	{MD2WithRSA, x509.MD2WithRSA, oidSignatureMD2WithRSA, crypto.Hash(0) /* no value for MD2 */},
	{MD5WithRSA, x509.MD5WithRSA, oidSignatureMD5WithRSA, crypto.MD5},
	{SHA1WithRSA, x509.SHA1WithRSA, oidSignatureSHA1WithRSA, crypto.SHA1},
	{SHA1WithRSA, x509.SHA1WithRSA, oidISOSignatureSHA1WithRSA, crypto.SHA1},
	{SHA256WithRSA, x509.SHA256WithRSA, oidSignatureSHA256WithRSA, crypto.SHA256},
	{SHA384WithRSA, x509.SHA384WithRSA, oidSignatureSHA384WithRSA, crypto.SHA384},
	{SHA512WithRSA, x509.SHA512WithRSA, oidSignatureSHA512WithRSA, crypto.SHA512},
	{SHA256WithRSAPSS, x509.SHA256WithRSAPSS, oidSignatureRSAPSS, crypto.SHA256},
	{SHA384WithRSAPSS, x509.SHA384WithRSAPSS, oidSignatureRSAPSS, crypto.SHA384},
	{SHA512WithRSAPSS, x509.SHA512WithRSAPSS, oidSignatureRSAPSS, crypto.SHA512},
	{DSAWithSHA1, x509.DSAWithSHA1, oidSignatureDSAWithSHA1, crypto.SHA1},
	{DSAWithSHA256, x509.DSAWithSHA256, oidSignatureDSAWithSHA256, crypto.SHA256},
	{ECDSAWithSHA1, x509.ECDSAWithSHA1, oidSignatureECDSAWithSHA1, crypto.SHA1},
	{ECDSAWithSHA256, x509.ECDSAWithSHA256, oidSignatureECDSAWithSHA256, crypto.SHA256},
	{ECDSAWithSHA384, x509.ECDSAWithSHA384, oidSignatureECDSAWithSHA384, crypto.SHA384},
	{ECDSAWithSHA512, x509.ECDSAWithSHA512, oidSignatureECDSAWithSHA512, crypto.SHA512},
	{PureEd25519, x509.PureEd25519, oidSignatureEd25519, crypto.Hash(0) /* no pre-hashing */},
}

// SignatureAlgorithm is the JSON representation of the X509 signature algorithms
type SignatureAlgorithm x509.SignatureAlgorithm

// Set sets the signature algorithm in the given certificate.
func (s SignatureAlgorithm) Set(c *x509.Certificate) {
	c.SignatureAlgorithm = x509.SignatureAlgorithm(s)
}

// MarshalJSON implements the json.Marshaller interface.
func (s SignatureAlgorithm) MarshalJSON() ([]byte, error) {
	if s == SignatureAlgorithm(x509.UnknownSignatureAlgorithm) {
		return []byte(`""`), nil
	}
	return []byte(`"` + x509.SignatureAlgorithm(s).String() + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshal interface and unmarshals and
// validates a string as a SignatureAlgorithm.
func (s *SignatureAlgorithm) UnmarshalJSON(data []byte) error {
	name, err := unmarshalString(data)
	if err != nil {
		return err
	}

	for _, m := range signatureAlgorithmMapping {
		if strings.EqualFold(name, m.name) {
			*s = SignatureAlgorithm(m.value)
			return nil
		}
	}

	return errors.Errorf("unsupported signatureAlgorithm %s", name)
}
