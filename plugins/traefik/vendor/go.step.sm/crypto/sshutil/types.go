package sshutil

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// CertType defines the certificate type, it can be a user or a host
// certificate.
type CertType uint32

const (
	// UserCert defines a user certificate.
	UserCert CertType = ssh.UserCert

	// HostCert defines a host certificate.
	HostCert CertType = ssh.HostCert
)

const (
	userString = "user"
	hostString = "host"
)

// CertTypeFromString returns the CertType for the string "user" and "host".
func CertTypeFromString(s string) (CertType, error) {
	switch strings.ToLower(s) {
	case userString:
		return UserCert, nil
	case hostString:
		return HostCert, nil
	default:
		return 0, errors.Errorf("unknown certificate type '%s'", s)
	}
}

// String returns "user" for user certificates and "host" for host certificates.
// It will return the empty string for any other value.
func (c CertType) String() string {
	switch c {
	case UserCert:
		return userString
	case HostCert:
		return hostString
	default:
		return ""
	}
}

// MarshalJSON implements the json.Marshaler interface for CertType. UserCert
// will be marshaled as the string "user" and HostCert as "host".
func (c CertType) MarshalJSON() ([]byte, error) {
	if s := c.String(); s != "" {
		return []byte(`"` + s + `"`), nil
	}
	return nil, errors.Errorf("unknown certificate type %d", c)
}

// UnmarshalJSON implements the json.Unmarshaler interface for CertType.
func (c *CertType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.Wrap(err, "error unmarshaling certificate type")
	}
	certType, err := CertTypeFromString(s)
	if err != nil {
		return errors.Errorf("error unmarshaling '%s' as a certificate type", s)
	}
	*c = certType
	return nil
}
