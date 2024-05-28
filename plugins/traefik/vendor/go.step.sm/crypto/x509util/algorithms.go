package x509util

import (
	"crypto/x509"
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

var signatureAlgorithmMapping = []struct {
	name  string
	value x509.SignatureAlgorithm
}{
	{"", x509.UnknownSignatureAlgorithm},
	{MD2WithRSA, x509.MD2WithRSA},
	{MD5WithRSA, x509.MD5WithRSA},
	{SHA1WithRSA, x509.SHA1WithRSA},
	{SHA256WithRSA, x509.SHA256WithRSA},
	{SHA384WithRSA, x509.SHA384WithRSA},
	{SHA512WithRSA, x509.SHA512WithRSA},
	{DSAWithSHA1, x509.DSAWithSHA1},
	{DSAWithSHA256, x509.DSAWithSHA256},
	{ECDSAWithSHA1, x509.ECDSAWithSHA1},
	{ECDSAWithSHA256, x509.ECDSAWithSHA256},
	{ECDSAWithSHA384, x509.ECDSAWithSHA384},
	{ECDSAWithSHA512, x509.ECDSAWithSHA512},
	{SHA256WithRSAPSS, x509.SHA256WithRSAPSS},
	{SHA384WithRSAPSS, x509.SHA384WithRSAPSS},
	{SHA512WithRSAPSS, x509.SHA512WithRSAPSS},
	{PureEd25519, x509.PureEd25519},
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
