package x509util

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func convertName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "")
}

// Names used for key usages.
var (
	KeyUsageDigitalSignature  = convertName("DigitalSignature")
	KeyUsageContentCommitment = convertName("ContentCommitment")
	KeyUsageKeyEncipherment   = convertName("KeyEncipherment")
	KeyUsageDataEncipherment  = convertName("DataEncipherment")
	KeyUsageKeyAgreement      = convertName("KeyAgreement")
	KeyUsageCertSign          = convertName("CertSign")
	KeyUsageCRLSign           = convertName("CRLSign")
	KeyUsageEncipherOnly      = convertName("EncipherOnly")
	KeyUsageDecipherOnly      = convertName("DecipherOnly")
)

// Names used for extended key usages.
var (
	ExtKeyUsageAny                            = convertName("Any")
	ExtKeyUsageServerAuth                     = convertName("ServerAuth")
	ExtKeyUsageClientAuth                     = convertName("ClientAuth")
	ExtKeyUsageCodeSigning                    = convertName("CodeSigning")
	ExtKeyUsageEmailProtection                = convertName("EmailProtection")
	ExtKeyUsageIPSECEndSystem                 = convertName("IPSECEndSystem")
	ExtKeyUsageIPSECTunnel                    = convertName("IPSECTunnel")
	ExtKeyUsageIPSECUser                      = convertName("IPSECUser")
	ExtKeyUsageTimeStamping                   = convertName("TimeStamping")
	ExtKeyUsageOCSPSigning                    = convertName("OCSPSigning")
	ExtKeyUsageMicrosoftServerGatedCrypto     = convertName("MicrosoftServerGatedCrypto")
	ExtKeyUsageNetscapeServerGatedCrypto      = convertName("NetscapeServerGatedCrypto")
	ExtKeyUsageMicrosoftCommercialCodeSigning = convertName("MicrosoftCommercialCodeSigning")
	ExtKeyUsageMicrosoftKernelCodeSigning     = convertName("MicrosoftKernelCodeSigning")
)

// Names used and SubjectAlternativeNames types.
const (
	AutoType                = "auto"
	EmailType               = "email" // also known as 'rfc822Name' in RFC 5280
	DNSType                 = "dns"
	X400AddressType         = "x400Address"
	DirectoryNameType       = "dn"
	EDIPartyNameType        = "ediPartyName"
	URIType                 = "uri"
	IPType                  = "ip"
	RegisteredIDType        = "registeredID"
	PermanentIdentifierType = "permanentIdentifier"
	HardwareModuleNameType  = "hardwareModuleName"
)

//nolint:deadcode // ignore
const (
	// These type ids are defined in RFC 5280 page 36.
	nameTypeOtherName     = 0
	nameTypeEmail         = 1
	nameTypeDNS           = 2
	nameTypeX400          = 3
	nameTypeDirectoryName = 4
	nameTypeEDI           = 5
	nameTypeURI           = 6
	nameTypeIP            = 7
	nameTypeRegisteredID  = 8
)

// sanTypeSeparator is used to set the type of otherName SANs. The format string
// is "[type:]value", printable will be used as default type if none is
// provided.
const sanTypeSeparator = ":"

// RFC 4043 - https://datatracker.ietf.org/doc/html/rfc4043
var oidPermanentIdentifier = []int{1, 3, 6, 1, 5, 5, 7, 8, 3}

// RFC 4108 - https://www.rfc-editor.org/rfc/rfc4108
var oidHardwareModuleNameIdentifier = []int{1, 3, 6, 1, 5, 5, 7, 8, 4}

// RFC 5280 - https://datatracker.ietf.org/doc/html/rfc5280#section-4.2.1.6
//
//	OtherName ::= SEQUENCE {
//	  type-id    OBJECT IDENTIFIER,
//	  value      [0] EXPLICIT ANY DEFINED BY type-id }
type otherName struct {
	TypeID asn1.ObjectIdentifier
	Value  asn1.RawValue
}

// PermanentIdentifier is defined in RFC 4043 as an optional feature that
// may be used by a CA to indicate that two or more certificates relate to the
// same entity.
//
// In device attestation this SAN will contain the UDID (Unique Device
// IDentifier) or serial number of the device.
//
// See https://tools.ietf.org/html/rfc4043
//
//	PermanentIdentifier ::= SEQUENCE {
//	  identifierValue    UTF8String OPTIONAL,
//	  assigner           OBJECT IDENTIFIER OPTIONAL
//	}
type PermanentIdentifier struct {
	Identifier string           `json:"identifier,omitempty"`
	Assigner   ObjectIdentifier `json:"assigner,omitempty"`
}

type asn1PermanentIdentifier struct {
	IdentifierValue string                `asn1:"utf8,optional"`
	Assigner        asn1.ObjectIdentifier `asn1:"optional"`
}

func (p *PermanentIdentifier) asn1Type() asn1PermanentIdentifier {
	return asn1PermanentIdentifier{
		IdentifierValue: p.Identifier,
		Assigner:        asn1.ObjectIdentifier(p.Assigner),
	}
}

// HardwareModuleName is defined in RFC 4108 as an optional feature that by be
// used to identify a hardware module.
//
// The OID defined for this SAN is "1.3.6.1.5.5.7.8.4".
//
// See https://www.rfc-editor.org/rfc/rfc4108#section-5
//
//	HardwareModuleName ::= SEQUENCE {
//	  hwType OBJECT IDENTIFIER,
//	  hwSerialNum OCTET STRING
//	}
type HardwareModuleName struct {
	Type         ObjectIdentifier `json:"type"`
	SerialNumber []byte           `json:"serialNumber"`
}

type asn1HardwareModuleName struct {
	Type         asn1.ObjectIdentifier
	SerialNumber []byte `asn1:"tag:4"`
}

func (h *HardwareModuleName) asn1Type() asn1HardwareModuleName {
	return asn1HardwareModuleName{
		Type:         asn1.ObjectIdentifier(h.Type),
		SerialNumber: h.SerialNumber,
	}
}

// Extension is the JSON representation of a raw X.509 extensions.
type Extension struct {
	ID       ObjectIdentifier `json:"id"`
	Critical bool             `json:"critical"`
	Value    []byte           `json:"value"`
}

// newExtension creates an Extension from a standard pkix.Extension.
func newExtension(e pkix.Extension) Extension {
	return Extension{
		ID:       ObjectIdentifier(e.Id),
		Critical: e.Critical,
		Value:    e.Value,
	}
}

// newExtensions creates a slice of Extension from a slice of pkix.Extension.
func newExtensions(extensions []pkix.Extension) []Extension {
	if extensions == nil {
		return nil
	}
	ret := make([]Extension, len(extensions))
	for i, e := range extensions {
		ret[i] = newExtension(e)
	}
	return ret
}

// Set adds the extension to the given X509 certificate.
func (e Extension) Set(c *x509.Certificate) {
	c.ExtraExtensions = append(c.ExtraExtensions, pkix.Extension{
		Id:       asn1.ObjectIdentifier(e.ID),
		Critical: e.Critical,
		Value:    e.Value,
	})
}

// ObjectIdentifier represents a JSON strings that unmarshals into an ASN1
// object identifier or OID.
type ObjectIdentifier asn1.ObjectIdentifier

// Equal reports whether o and v represent the same identifier.
func (o ObjectIdentifier) Equal(v ObjectIdentifier) bool {
	if len(o) != len(v) {
		return false
	}
	for i := 0; i < len(o); i++ {
		if o[i] != v[i] {
			return false
		}
	}
	return true
}

// MarshalJSON implements the json.Marshaler interface and returns the string
// version of the asn1.ObjectIdentifier.
func (o ObjectIdentifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(asn1.ObjectIdentifier(o).String())
}

// UnmarshalJSON implements the json.Unmarshaler interface and coverts a strings
// like "2.5.29.17" into an ASN1 object identifier.
func (o *ObjectIdentifier) UnmarshalJSON(data []byte) error {
	s, err := unmarshalString(data)
	if err != nil {
		return err
	}

	oid, err := parseObjectIdentifier(s)
	if err != nil {
		return err
	}
	*o = ObjectIdentifier(oid)
	return nil
}

// SubjectAlternativeName represents a X.509 subject alternative name. Types
// supported are "dns", "email", "ip", "uri". A special type "auto" or "" can be
// used to try to guess the type of the value.
//
// ASN1Value can only be used for those types where the string value cannot
// contain enough information to encode the value.
type SubjectAlternativeName struct {
	Type      string          `json:"type"`
	Value     string          `json:"value"`
	ASN1Value json.RawMessage `json:"asn1Value,omitempty"`
}

// Set sets the subject alternative name in the given x509.Certificate.
func (s SubjectAlternativeName) Set(c *x509.Certificate) {
	switch strings.ToLower(s.Type) {
	case DNSType:
		c.DNSNames = append(c.DNSNames, s.Value)
	case EmailType:
		c.EmailAddresses = append(c.EmailAddresses, s.Value)
	case IPType:
		// The validation of the IP would happen in the unmarshaling, but just
		// to be sure we are only adding valid IPs.
		if ip := net.ParseIP(s.Value); ip != nil {
			c.IPAddresses = append(c.IPAddresses, ip)
		}
	case URIType:
		if u, err := url.Parse(s.Value); err == nil {
			c.URIs = append(c.URIs, u)
		}
	case "", AutoType:
		dnsNames, ips, emails, uris := SplitSANs([]string{s.Value})
		c.DNSNames = append(c.DNSNames, dnsNames...)
		c.IPAddresses = append(c.IPAddresses, ips...)
		c.EmailAddresses = append(c.EmailAddresses, emails...)
		c.URIs = append(c.URIs, uris...)
	default:
		panic(fmt.Sprintf("unsupported subject alternative name type %s", s.Type))
	}
}

// RawValue returns the undecoded ASN.1 object for the SAN.
func (s SubjectAlternativeName) RawValue() (asn1.RawValue, error) {
	var zero asn1.RawValue

	switch s.Type {
	case "", AutoType:
		// autotype requires us to find out what the type is.
		ip := net.ParseIP(s.Value)
		u, err := url.Parse(s.Value)
		switch {
		case ip != nil:
			return SubjectAlternativeName{Type: IPType, Value: s.Value}.RawValue()
		case err == nil && u.Scheme != "":
			return SubjectAlternativeName{Type: URIType, Value: s.Value}.RawValue()
		case strings.Contains(s.Value, "@"):
			return SubjectAlternativeName{Type: EmailType, Value: s.Value}.RawValue()
		default:
			return SubjectAlternativeName{Type: DNSType, Value: s.Value}.RawValue()
		}
	case EmailType:
		valid := isIA5String(s.Value)
		if !valid {
			return zero, fmt.Errorf("error converting %q to ia5", s.Value)
		}
		return asn1.RawValue{Tag: nameTypeEmail, Class: asn1.ClassContextSpecific, Bytes: []byte(s.Value)}, nil
	case DNSType:
		// use SanitizeName for DNS types because it will do some character
		// replacement and verify that its an acceptable hostname
		ia5String, err := SanitizeName(s.Value)
		if err != nil {
			return zero, errors.Wrapf(err, "error converting %q to ia5", s.Value)
		}
		return asn1.RawValue{Tag: nameTypeDNS, Class: asn1.ClassContextSpecific, Bytes: []byte(ia5String)}, nil
	case URIType:
		valid := isIA5String(s.Value)
		if !valid {
			return zero, fmt.Errorf("error converting %q to ia5", s.Value)
		}
		return asn1.RawValue{Tag: nameTypeURI, Class: asn1.ClassContextSpecific, Bytes: []byte(s.Value)}, nil
	case IPType:
		rawIP := net.ParseIP(s.Value)
		if rawIP == nil {
			return zero, fmt.Errorf("error converting %q to IP", s.Value)
		}
		ip := rawIP.To4()
		if ip == nil {
			ip = rawIP
		}
		return asn1.RawValue{Tag: nameTypeIP, Class: asn1.ClassContextSpecific, Bytes: ip}, nil
	case RegisteredIDType:
		if s.Value == "" {
			return zero, errors.New("error parsing RegisteredID SAN: blank value is not allowed")
		}
		oid, err := parseObjectIdentifier(s.Value)
		if err != nil {
			return zero, errors.Wrap(err, "error parsing OID for RegisteredID SAN")
		}
		rawBytes, err := asn1.MarshalWithParams(oid, "tag:8")
		if err != nil {
			return zero, errors.Wrap(err, "error marshaling RegisteredID SAN")
		}
		return asn1.RawValue{FullBytes: rawBytes}, nil
	case PermanentIdentifierType:
		var v PermanentIdentifier
		switch {
		case len(s.ASN1Value) != 0:
			if err := json.Unmarshal(s.ASN1Value, &v); err != nil {
				return zero, errors.Wrap(err, "error unmarshaling PermanentIdentifier SAN")
			}
		case s.Value != "":
			v.Identifier = s.Value
		default: // continue, both identifierValue and assigner are optional
		}
		otherName, err := marshalOtherName(oidPermanentIdentifier, v.asn1Type())
		if err != nil {
			return zero, errors.Wrap(err, "error marshaling PermanentIdentifier SAN")
		}
		return otherName, nil
	case HardwareModuleNameType:
		if len(s.ASN1Value) == 0 {
			return zero, errors.New("error parsing HardwareModuleName SAN: empty asn1Value is not allowed")
		}
		var v HardwareModuleName
		if err := json.Unmarshal(s.ASN1Value, &v); err != nil {
			return zero, errors.Wrap(err, "error unmarshaling HardwareModuleName SAN")
		}
		otherName, err := marshalOtherName(oidHardwareModuleNameIdentifier, v.asn1Type())
		if err != nil {
			return zero, errors.Wrap(err, "error marshaling HardwareModuleName SAN")
		}
		return otherName, nil
	case DirectoryNameType:
		if len(s.ASN1Value) == 0 {
			return zero, errors.New("error parsing DirectoryName SAN: empty asn1Value is not allowed")
		}
		var dn Name
		if err := json.Unmarshal(s.ASN1Value, &dn); err != nil {
			return zero, errors.Wrap(err, "error unmarshaling DirectoryName SAN")
		}
		rdn, err := asn1.Marshal(dn.goValue().ToRDNSequence())
		if err != nil {
			return zero, errors.Wrap(err, "error marshaling DirectoryName SAN")
		}
		if bytes.Equal(rdn, emptyASN1Subject) {
			return zero, errors.New("error parsing DirectoryName SAN: empty or malformed asn1Value")
		}
		return asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        nameTypeDirectoryName,
			IsCompound: true,
			Bytes:      rdn,
		}, nil
	case X400AddressType, EDIPartyNameType:
		return zero, fmt.Errorf("unimplemented SAN type %s", s.Type)
	default:
		// Assume otherName with a valid oid in type.
		oid, err := parseObjectIdentifier(s.Type)
		if err != nil {
			return zero, fmt.Errorf("unsupported SAN type %s", s.Type)
		}

		// The default type is printable, but if the value is prefixed with a
		// type, use that.
		var value, params = s.Value, "printable"
		if strings.Contains(value, sanTypeSeparator) {
			params = strings.Split(value, sanTypeSeparator)[0]
			value = value[len(params)+1:]
		}

		rawBytes, err := marshalExplicitValue(value, params)
		if err != nil {
			return zero, errors.Wrapf(err, "error marshaling ASN1 value %q", s.Value)
		}

		// use MarshalWithParams so we can set the context-specific tag - in this case 0
		otherNameBytes, err := asn1.MarshalWithParams(otherName{
			TypeID: oid,
			Value:  asn1.RawValue{FullBytes: rawBytes},
		}, "tag:0")
		if err != nil {
			return zero, errors.Wrap(err, "unable to Marshal otherName SAN")
		}
		return asn1.RawValue{FullBytes: otherNameBytes}, nil
	}
}

// marshalOtherName marshals an otherName field with the given oid and value and
// returns the raw bytes to use.
func marshalOtherName(oid asn1.ObjectIdentifier, value interface{}) (asn1.RawValue, error) {
	valueBytes, err := asn1.MarshalWithParams(value, "explicit,tag:0")
	if err != nil {
		return asn1.RawValue{}, err
	}
	b, err := asn1.MarshalWithParams(otherName{
		TypeID: oid,
		Value:  asn1.RawValue{FullBytes: valueBytes},
	}, "tag:0")
	if err != nil {
		return asn1.RawValue{}, err
	}
	return asn1.RawValue{FullBytes: b}, nil
}

type asn1Params struct {
	Type   string
	Params string
}

func parseFieldParameters(str string) (p asn1Params) {
	var part string
	var params []string
	for len(str) > 0 {
		part, str, _ = strings.Cut(str, ",")
		switch part {
		// string types
		case "utf8", "ia5", "numeric", "printable":
			p.Type = part
			params = append(params, part)
		// types that are parsed from the string.
		// int and oid are not a type that can be set in a tag.
		case "int", "oid":
			p.Type = part
		// types parsed from the string as a time
		case "utc", "generalized":
			p.Type = part
			params = append(params, part)
		// base64 encoded asn1 value
		case "raw":
			p.Type = part
		case "":
			// skip
		default:
			params = append(params, part)
		}
	}
	p.Params = strings.Join(params, ",")
	return p
}

// marshalValue marshals the given value with the given params.
//
// The return value value can be any type depending on the OID. ASN supports a
// great number of formats, but Golang's asn1 package supports much fewer -- for
// now support anything the Golang asn1 marshaller supports.
//
// See https://www.openssl.org/docs/man1.0.2/man3/ASN1_generate_nconf.html
func marshalValue(value, params string) ([]byte, error) {
	p := parseFieldParameters(params)
	// Marshal types without a tag support.
	switch p.Type {
	case "int":
		i, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.Wrap(err, "invalid int value")
		}
		return asn1.MarshalWithParams(i, p.Params)
	case "oid":
		oid, err := parseObjectIdentifier(value)
		if err != nil {
			return nil, errors.Wrap(err, "invalid oid value")
		}
		return asn1.MarshalWithParams(oid, p.Params)
	case "raw":
		// the raw type accepts a base64 encoded byte array which is passed unaltered into the ASN
		// marshaller. By using this type users can add ASN1 data types manually into templates
		// to support some unsupported types like BMPString, Octet String, and others
		return base64.StdEncoding.DecodeString(value)
	case "utf8":
		if !isUTF8String(value) {
			return nil, fmt.Errorf("invalid utf8 value")
		}
		return asn1.MarshalWithParams(value, p.Params)
	case "ia5":
		if !isIA5String(value) {
			return nil, fmt.Errorf("invalid ia5 value")
		}
		return asn1.MarshalWithParams(value, p.Params)
	case "numeric":
		if !isNumericString(value) {
			return nil, fmt.Errorf("invalid numeric value")
		}
		return asn1.MarshalWithParams(value, p.Params)
	case "printable":
		if !isPrintableString(value, true, true) {
			return nil, fmt.Errorf("invalid printable value")
		}
		return asn1.MarshalWithParams(value, p.Params)
	case "utc", "generalized":
		// This is the layout of Time.String() function
		const defaultLayout = "2006-01-02 15:04:05.999999999 -0700 MST"
		t, err := time.Parse(defaultLayout, value)
		if err != nil {
			var err2 error
			if t, err2 = time.Parse(time.RFC3339, value); err2 != nil {
				return nil, errors.Wrapf(err, "invalid %s value", p.Type)
			}
		}
		return asn1.MarshalWithParams(t, p.Params)
	default: // if it's an unknown type, default to printable
		if !isPrintableString(value, true, true) {
			return nil, fmt.Errorf("invalid printable value")
		}
		return asn1.MarshalWithParams(value, p.Params)
	}
}

// marshalExplicitValue marshals the given value with given type and returns the
// raw bytes to use. It will add the explicit tag to the final parameters.
func marshalExplicitValue(value, typ string) ([]byte, error) {
	return marshalValue(value, "explicit,"+typ)
}

// KeyUsage type represents the JSON array used to represent the key usages of a
// X509 certificate.
type KeyUsage x509.KeyUsage

// Set sets the key usage to the given certificate.
func (k KeyUsage) Set(c *x509.Certificate) {
	c.KeyUsage = x509.KeyUsage(k)
}

// UnmarshalJSON implements the json.Unmarshaler interface and coverts a string
// or a list of strings into a key usage.
func (k *KeyUsage) UnmarshalJSON(data []byte) error {
	ms, err := unmarshalMultiString(data)
	if err != nil {
		return err
	}

	*k = 0

	for _, s := range ms {
		var ku x509.KeyUsage
		switch convertName(s) {
		case KeyUsageDigitalSignature:
			ku = x509.KeyUsageDigitalSignature
		case KeyUsageContentCommitment:
			ku = x509.KeyUsageContentCommitment
		case KeyUsageKeyEncipherment:
			ku = x509.KeyUsageKeyEncipherment
		case KeyUsageDataEncipherment:
			ku = x509.KeyUsageDataEncipherment
		case KeyUsageKeyAgreement:
			ku = x509.KeyUsageKeyAgreement
		case KeyUsageCertSign:
			ku = x509.KeyUsageCertSign
		case KeyUsageCRLSign:
			ku = x509.KeyUsageCRLSign
		case KeyUsageEncipherOnly:
			ku = x509.KeyUsageEncipherOnly
		case KeyUsageDecipherOnly:
			ku = x509.KeyUsageDecipherOnly
		default:
			return errors.Errorf("unsupported keyUsage %s", s)
		}
		*k |= KeyUsage(ku)
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface and converts a key usage
// into a list of strings.
func (k KeyUsage) MarshalJSON() ([]byte, error) {
	var usages []string

	if x509.KeyUsage(k)&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, KeyUsageDigitalSignature)
	}
	if x509.KeyUsage(k)&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, KeyUsageContentCommitment)
	}
	if x509.KeyUsage(k)&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, KeyUsageKeyEncipherment)
	}
	if x509.KeyUsage(k)&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, KeyUsageDataEncipherment)
	}
	if x509.KeyUsage(k)&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, KeyUsageKeyAgreement)
	}
	if x509.KeyUsage(k)&x509.KeyUsageCertSign != 0 {
		usages = append(usages, KeyUsageCertSign)
	}
	if x509.KeyUsage(k)&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, KeyUsageCRLSign)
	}
	if x509.KeyUsage(k)&x509.KeyUsageEncipherOnly != 0 {
		usages = append(usages, KeyUsageEncipherOnly)
	}
	if x509.KeyUsage(k)&x509.KeyUsageDecipherOnly != 0 {
		usages = append(usages, KeyUsageDecipherOnly)
	}

	if len(usages) == 0 && k != 0 {
		return nil, fmt.Errorf("cannot marshal key usage %v", k)
	}

	return json.Marshal(usages)
}

// ExtKeyUsage represents a JSON array of extended key usages.
type ExtKeyUsage []x509.ExtKeyUsage

// Set sets the extended key usages in the given certificate.
func (k ExtKeyUsage) Set(c *x509.Certificate) {
	c.ExtKeyUsage = []x509.ExtKeyUsage(k)
}

// UnmarshalJSON implements the json.Unmarshaler interface and coverts a string
// or a list of strings into a list of extended key usages.
func (k *ExtKeyUsage) UnmarshalJSON(data []byte) error {
	ms, err := unmarshalMultiString(data)
	if err != nil {
		return err
	}

	eku := make([]x509.ExtKeyUsage, len(ms))
	for i, s := range ms {
		var ku x509.ExtKeyUsage
		switch convertName(s) {
		case ExtKeyUsageAny:
			ku = x509.ExtKeyUsageAny
		case ExtKeyUsageServerAuth:
			ku = x509.ExtKeyUsageServerAuth
		case ExtKeyUsageClientAuth:
			ku = x509.ExtKeyUsageClientAuth
		case ExtKeyUsageCodeSigning:
			ku = x509.ExtKeyUsageCodeSigning
		case ExtKeyUsageEmailProtection:
			ku = x509.ExtKeyUsageEmailProtection
		case ExtKeyUsageIPSECEndSystem:
			ku = x509.ExtKeyUsageIPSECEndSystem
		case ExtKeyUsageIPSECTunnel:
			ku = x509.ExtKeyUsageIPSECTunnel
		case ExtKeyUsageIPSECUser:
			ku = x509.ExtKeyUsageIPSECUser
		case ExtKeyUsageTimeStamping:
			ku = x509.ExtKeyUsageTimeStamping
		case ExtKeyUsageOCSPSigning:
			ku = x509.ExtKeyUsageOCSPSigning
		case ExtKeyUsageMicrosoftServerGatedCrypto:
			ku = x509.ExtKeyUsageMicrosoftServerGatedCrypto
		case ExtKeyUsageNetscapeServerGatedCrypto:
			ku = x509.ExtKeyUsageNetscapeServerGatedCrypto
		case ExtKeyUsageMicrosoftCommercialCodeSigning:
			ku = x509.ExtKeyUsageMicrosoftCommercialCodeSigning
		case ExtKeyUsageMicrosoftKernelCodeSigning:
			ku = x509.ExtKeyUsageMicrosoftKernelCodeSigning
		default:
			return errors.Errorf("unsupported extKeyUsage %s", s)
		}
		eku[i] = ku
	}

	*k = ExtKeyUsage(eku)
	return nil
}

// MarshalJSON implements the json.Marshaler interface and converts a list of
// extended key usages to a list of strings
func (k ExtKeyUsage) MarshalJSON() ([]byte, error) {
	usages := make([]string, len(k))

	for i, eku := range k {
		switch eku {
		case x509.ExtKeyUsageAny:
			usages[i] = ExtKeyUsageAny
		case x509.ExtKeyUsageServerAuth:
			usages[i] = ExtKeyUsageServerAuth
		case x509.ExtKeyUsageClientAuth:
			usages[i] = ExtKeyUsageClientAuth
		case x509.ExtKeyUsageCodeSigning:
			usages[i] = ExtKeyUsageCodeSigning
		case x509.ExtKeyUsageEmailProtection:
			usages[i] = ExtKeyUsageEmailProtection
		case x509.ExtKeyUsageIPSECEndSystem:
			usages[i] = ExtKeyUsageIPSECEndSystem
		case x509.ExtKeyUsageIPSECTunnel:
			usages[i] = ExtKeyUsageIPSECTunnel
		case x509.ExtKeyUsageIPSECUser:
			usages[i] = ExtKeyUsageIPSECUser
		case x509.ExtKeyUsageTimeStamping:
			usages[i] = ExtKeyUsageTimeStamping
		case x509.ExtKeyUsageOCSPSigning:
			usages[i] = ExtKeyUsageOCSPSigning
		case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
			usages[i] = ExtKeyUsageMicrosoftServerGatedCrypto
		case x509.ExtKeyUsageNetscapeServerGatedCrypto:
			usages[i] = ExtKeyUsageNetscapeServerGatedCrypto
		case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
			usages[i] = ExtKeyUsageMicrosoftCommercialCodeSigning
		case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
			usages[i] = ExtKeyUsageMicrosoftKernelCodeSigning
		default:
			return nil, fmt.Errorf("unsupported extKeyUsage %v", eku)
		}
	}

	return json.Marshal(usages)
}

// UnknownExtKeyUsage represents the list of OIDs of extended key usages unknown
// to crypto/x509.
type UnknownExtKeyUsage MultiObjectIdentifier

// MarshalJSON implements the json.Marshaler interface in UnknownExtKeyUsage.
func (u UnknownExtKeyUsage) MarshalJSON() ([]byte, error) {
	return MultiObjectIdentifier(u).MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface in UnknownExtKeyUsage.
func (u *UnknownExtKeyUsage) UnmarshalJSON(data []byte) error {
	var v MultiObjectIdentifier
	if err := json.Unmarshal(data, &v); err != nil {
		return errors.Wrap(err, "error unmarshaling json")
	}
	*u = UnknownExtKeyUsage(v)
	return nil
}

// Set sets the policy identifiers to the given certificate.
func (u UnknownExtKeyUsage) Set(c *x509.Certificate) {
	c.UnknownExtKeyUsage = u
}

// SubjectKeyID represents the binary value of the subject key identifier
// extension, this should be the SHA-1 hash of the public key. In JSON this
// value should be a base64-encoded string, and in most cases it should not be
// set because it will be automatically generated.
type SubjectKeyID []byte

// Set sets the subject key identifier to the given certificate.
func (id SubjectKeyID) Set(c *x509.Certificate) {
	c.SubjectKeyId = id
}

// AuthorityKeyID represents the binary value of the authority key identifier
// extension. It should be the subject key identifier of the parent certificate.
// In JSON this value should be a base64-encoded string, and in most cases it
// should not be set, as it will be automatically provided.
type AuthorityKeyID []byte

// Set sets the authority key identifier to the given certificate.
func (id AuthorityKeyID) Set(c *x509.Certificate) {
	c.AuthorityKeyId = id
}

// OCSPServer contains the list of OSCP servers that will be encoded in the
// authority information access extension.
type OCSPServer MultiString

// UnmarshalJSON implements the json.Unmarshaler interface in OCSPServer.
func (o *OCSPServer) UnmarshalJSON(data []byte) error {
	ms, err := unmarshalMultiString(data)
	if err != nil {
		return err
	}
	*o = ms
	return nil
}

// Set sets the list of OSCP servers to the given certificate.
func (o OCSPServer) Set(c *x509.Certificate) {
	c.OCSPServer = o
}

// IssuingCertificateURL contains the list of the issuing certificate url that
// will be encoded in the authority information access extension.
type IssuingCertificateURL MultiString

// UnmarshalJSON implements the json.Unmarshaler interface in IssuingCertificateURL.
func (u *IssuingCertificateURL) UnmarshalJSON(data []byte) error {
	ms, err := unmarshalMultiString(data)
	if err != nil {
		return err
	}
	*u = ms
	return nil
}

// Set sets the list of issuing certificate urls to the given certificate.
func (u IssuingCertificateURL) Set(c *x509.Certificate) {
	c.IssuingCertificateURL = u
}

// CRLDistributionPoints contains the list of CRL distribution points that will
// be encoded in the CRL distribution points extension.
type CRLDistributionPoints MultiString

// UnmarshalJSON implements the json.Unmarshaler interface in CRLDistributionPoints.
func (u *CRLDistributionPoints) UnmarshalJSON(data []byte) error {
	ms, err := unmarshalMultiString(data)
	if err != nil {
		return err
	}
	*u = ms
	return nil
}

// Set sets the CRL distribution points to the given certificate.
func (u CRLDistributionPoints) Set(c *x509.Certificate) {
	c.CRLDistributionPoints = u
}

// PolicyIdentifiers represents the list of OIDs to set in the certificate
// policies extension.
type PolicyIdentifiers MultiObjectIdentifier

// MarshalJSON implements the json.Marshaler interface in PolicyIdentifiers.
func (p PolicyIdentifiers) MarshalJSON() ([]byte, error) {
	return MultiObjectIdentifier(p).MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface in PolicyIdentifiers.
func (p *PolicyIdentifiers) UnmarshalJSON(data []byte) error {
	var v MultiObjectIdentifier
	if err := json.Unmarshal(data, &v); err != nil {
		return errors.Wrap(err, "error unmarshaling json")
	}
	*p = PolicyIdentifiers(v)
	return nil
}

// Set sets the policy identifiers to the given certificate.
func (p PolicyIdentifiers) Set(c *x509.Certificate) {
	c.PolicyIdentifiers = p
}

// BasicConstraints represents the X509 basic constraints extension and defines
// if a certificate is a CA and then maximum depth of valid certification paths
// that include the certificate. A MaxPathLen of zero indicates that no non-
// self-issued intermediate CA certificates may follow in a valid certification
// path. To do not impose a limit the MaxPathLen should be set to -1.
type BasicConstraints struct {
	IsCA       bool `json:"isCA"`
	MaxPathLen int  `json:"maxPathLen"`
}

// Set sets the basic constraints to the given certificate.
func (b BasicConstraints) Set(c *x509.Certificate) {
	c.BasicConstraintsValid = true
	c.IsCA = b.IsCA
	if c.IsCA {
		switch {
		case b.MaxPathLen == 0:
			c.MaxPathLen = 0
			c.MaxPathLenZero = true
		case b.MaxPathLen < 0:
			c.MaxPathLen = -1
			c.MaxPathLenZero = false
		default:
			c.MaxPathLen = b.MaxPathLen
			c.MaxPathLenZero = false
		}
	} else {
		c.MaxPathLen = 0
		c.MaxPathLenZero = false
	}
}

// NameConstraints represents the X509 Name constraints extension and defines a
// names space within which all subject names in subsequent certificates in a
// certificate path must be located. The name constraints extension must be used
// only in a CA.
type NameConstraints struct {
	Critical                bool        `json:"critical"`
	PermittedDNSDomains     MultiString `json:"permittedDNSDomains"`
	ExcludedDNSDomains      MultiString `json:"excludedDNSDomains"`
	PermittedIPRanges       MultiIPNet  `json:"permittedIPRanges"`
	ExcludedIPRanges        MultiIPNet  `json:"excludedIPRanges"`
	PermittedEmailAddresses MultiString `json:"permittedEmailAddresses"`
	ExcludedEmailAddresses  MultiString `json:"excludedEmailAddresses"`
	PermittedURIDomains     MultiString `json:"permittedURIDomains"`
	ExcludedURIDomains      MultiString `json:"excludedURIDomains"`
}

// Set sets the name constraints in the given certificate.
func (n NameConstraints) Set(c *x509.Certificate) {
	c.PermittedDNSDomainsCritical = n.Critical
	c.PermittedDNSDomains = n.PermittedDNSDomains
	c.ExcludedDNSDomains = n.ExcludedDNSDomains
	c.PermittedIPRanges = n.PermittedIPRanges
	c.ExcludedIPRanges = n.ExcludedIPRanges
	c.PermittedEmailAddresses = n.PermittedEmailAddresses
	c.ExcludedEmailAddresses = n.ExcludedEmailAddresses
	c.PermittedURIDomains = n.PermittedURIDomains
	c.ExcludedURIDomains = n.ExcludedURIDomains
}

// SerialNumber is the JSON representation of the X509 serial number.
type SerialNumber struct {
	*big.Int
}

// Set sets the serial number in the given certificate.
func (s SerialNumber) Set(c *x509.Certificate) {
	c.SerialNumber = s.Int
}

// MarshalJSON implements the json.Marshaler interface, and encodes a
// SerialNumber using the big.Int marshaler.
func (s *SerialNumber) MarshalJSON() ([]byte, error) {
	if s == nil || s.Int == nil {
		return []byte(`null`), nil
	}
	return s.Int.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshal interface and unmarshals an
// integer or a string into a serial number. If a string is used, a prefix of
// “0b” or “0B” selects base 2, “0”, “0o” or “0O” selects base 8, and “0x” or
// “0X” selects base 16. Otherwise, the selected base is 10 and no prefix is
// accepted.
func (s *SerialNumber) UnmarshalJSON(data []byte) error {
	if sn, ok := maybeString(data); ok {
		// Using base 0 to accept prefixes 0b, 0o, 0x but defaults as base 10.
		b, ok := new(big.Int).SetString(sn, 0)
		if !ok {
			return errors.Errorf("error unmarshaling json: serialNumber %s is not valid", sn)
		}
		*s = SerialNumber{
			Int: b,
		}
		return nil
	}

	// Assume a number.
	var i int64
	if err := json.Unmarshal(data, &i); err != nil {
		return errors.Wrap(err, "error unmarshaling json")
	}
	*s = SerialNumber{
		Int: new(big.Int).SetInt64(i),
	}
	return nil
}

func createCertificateSubjectAltNameExtension(c Certificate, subjectIsEmpty bool) (Extension, error) {
	return createSubjectAltNameExtension(c.DNSNames, c.EmailAddresses, c.IPAddresses, c.URIs, c.SANs, subjectIsEmpty)
}

func createCertificateRequestSubjectAltNameExtension(c CertificateRequest, subjectIsEmpty bool) (Extension, error) {
	return createSubjectAltNameExtension(c.DNSNames, c.EmailAddresses, c.IPAddresses, c.URIs, c.SANs, subjectIsEmpty)
}

// createSubjectAltNameExtension will construct an Extension containing all
// SubjectAlternativeNames held in a Certificate. It implements more types than
// the golang x509 library, so it is used whenever OtherName or RegisteredID
// type SANs are present in the certificate.
//
// See also https://datatracker.ietf.org/doc/html/rfc5280.html#section-4.2.1.6
//
// TODO(mariano,unreality): X400Address, DirectoryName, and EDIPartyName types
// are defined in RFC5280 but are currently unimplemented
func createSubjectAltNameExtension(dnsNames, emailAddresses MultiString, ipAddresses MultiIP, uris MultiURL, sans []SubjectAlternativeName, subjectIsEmpty bool) (Extension, error) {
	var zero Extension

	var rawValues []asn1.RawValue
	for _, dnsName := range dnsNames {
		rawValue, err := SubjectAlternativeName{
			Type: DNSType, Value: dnsName,
		}.RawValue()
		if err != nil {
			return zero, err
		}

		rawValues = append(rawValues, rawValue)
	}

	for _, emailAddress := range emailAddresses {
		rawValue, err := SubjectAlternativeName{
			Type: EmailType, Value: emailAddress,
		}.RawValue()
		if err != nil {
			return zero, err
		}

		rawValues = append(rawValues, rawValue)
	}

	for _, ip := range ipAddresses {
		rawValue, err := SubjectAlternativeName{
			Type: IPType, Value: ip.String(),
		}.RawValue()
		if err != nil {
			return zero, err
		}

		rawValues = append(rawValues, rawValue)
	}

	for _, uri := range uris {
		rawValue, err := SubjectAlternativeName{
			Type: URIType, Value: uri.String(),
		}.RawValue()
		if err != nil {
			return zero, err
		}

		rawValues = append(rawValues, rawValue)
	}

	for _, san := range sans {
		rawValue, err := san.RawValue()
		if err != nil {
			return zero, err
		}

		rawValues = append(rawValues, rawValue)
	}

	// Now marshal the rawValues into the ASN1 sequence, and create an Extension object to hold the extension
	rawBytes, err := asn1.Marshal(rawValues)
	if err != nil {
		return zero, errors.Wrap(err, "error marshaling SubjectAlternativeName extension to ASN1")
	}

	return Extension{
		ID:       oidExtensionSubjectAltName,
		Critical: subjectIsEmpty,
		Value:    rawBytes,
	}, nil
}

// SubjectAlternativeNames is a container for names extracted
// from the X.509 Subject Alternative Names extension.
type SubjectAlternativeNames struct {
	DNSNames             []string
	EmailAddresses       []string
	IPAddresses          []net.IP
	URIs                 []*url.URL
	PermanentIdentifiers []PermanentIdentifier
	HardwareModuleNames  []HardwareModuleName
	TPMHardwareDetails   TPMHardwareDetails
	//OtherNames          []OtherName // TODO(hs): unused at the moment; do we need it? what type definition to use?
}

// TPMHardwareDetails is a container for some details
// for TPM hardware.
type TPMHardwareDetails struct {
	Manufacturer string // TODO(hs): use Manufacturer from TPM package? Need to fix import cycle, though
	Model        string
	Version      string
}

var (
	oidTPMManufacturer = asn1.ObjectIdentifier{2, 23, 133, 2, 1}
	oidTPMModel        = asn1.ObjectIdentifier{2, 23, 133, 2, 2}
	oidTPMVersion      = asn1.ObjectIdentifier{2, 23, 133, 2, 3}
)

// ParseSubjectAlternativeNames parses the Subject Alternative Names
// from the X.509 certificate `c`. SAN types supported by the Go stdlib,
// including DNS names, IP addresses, email addresses and URLs, are copied
// to the result first. After that, the raw extension bytes are parsed to
// extract PermanentIdentifiers and HardwareModuleNames SANs.
func ParseSubjectAlternativeNames(c *x509.Certificate) (sans SubjectAlternativeNames, err error) {
	// the Certificate c is expected to have been processed before, so the
	// SANs known by the Go stdlib are expected to have been populated already.
	// These SANs are copied over to the result.
	sans.DNSNames = c.DNSNames
	sans.IPAddresses = c.IPAddresses
	sans.EmailAddresses = c.EmailAddresses
	sans.URIs = c.URIs

	var sanExtension pkix.Extension
	for _, ext := range c.Extensions {
		if ext.Id.Equal(oidExtensionSubjectAltName) {
			sanExtension = ext
			break
		}
	}

	if sanExtension.Value == nil {
		return
	}

	directoryNames, otherNames, err := parseSubjectAltName(sanExtension)
	if err != nil {
		return sans, fmt.Errorf("failed parsing SubjectAltName extension: %w", err)
	}

	for _, otherName := range otherNames {
		switch {
		case otherName.TypeID.Equal(oidPermanentIdentifier):
			permanentIdentifier, err := parsePermanentIdentifier(otherName.Value.FullBytes)
			if err != nil {
				return sans, fmt.Errorf("failed parsing PermanentIdentifier: %w", err)
			}
			sans.PermanentIdentifiers = append(sans.PermanentIdentifiers, permanentIdentifier)
		case otherName.TypeID.Equal(oidHardwareModuleNameIdentifier):
			hardwareModuleName, err := parseHardwareModuleName(otherName.Value.FullBytes)
			if err != nil {
				return sans, fmt.Errorf("failed parsing HardwareModuleName: %w", err)
			}
			sans.HardwareModuleNames = append(sans.HardwareModuleNames, hardwareModuleName)
		default:
			// TODO(hs): handle other types; defaulting to otherName?
		}
	}

	tpmDetails := TPMHardwareDetails{}
	for _, directoryName := range directoryNames {
		for _, name := range directoryName.Names {
			switch {
			case name.Type.Equal(oidTPMManufacturer):
				tpmDetails.Manufacturer = name.Value.(string)
			case name.Type.Equal(oidTPMModel):
				tpmDetails.Model = name.Value.(string)
			case name.Type.Equal(oidTPMVersion):
				tpmDetails.Version = name.Value.(string)
			default:
				// TODO(hs): handle other directoryNames?
			}
		}
	}
	sans.TPMHardwareDetails = tpmDetails

	return
}

// https://datatracker.ietf.org/doc/html/rfc5280#page-35
func parseSubjectAltName(ext pkix.Extension) (dirNames []pkix.Name, otherNames []otherName, err error) {
	err = forEachSAN(ext.Value, func(generalName asn1.RawValue) error {
		switch generalName.Tag {
		case 0: // otherName
			var on otherName
			if _, err := asn1.UnmarshalWithParams(generalName.FullBytes, &on, "tag:0"); err != nil {
				return fmt.Errorf("failed unmarshaling otherName: %w", err)
			}
			otherNames = append(otherNames, on)
		case 4: // directoryName
			var rdns pkix.RDNSequence
			if _, err := asn1.Unmarshal(generalName.Bytes, &rdns); err != nil {
				return fmt.Errorf("failed unmarshaling directoryName: %w", err)
			}
			var dirName pkix.Name
			dirName.FillFromRDNSequence(&rdns)
			dirNames = append(dirNames, dirName)
		default:
			// skipping the other tag values intentionally
		}
		return nil
	})
	return
}

func parsePermanentIdentifier(der []byte) (PermanentIdentifier, error) {
	var permID asn1PermanentIdentifier
	if _, err := asn1.UnmarshalWithParams(der, &permID, "explicit,tag:0"); err != nil {
		return PermanentIdentifier{}, fmt.Errorf("failed unmarshaling der data: %w", err)
	}
	return PermanentIdentifier{Identifier: permID.IdentifierValue, Assigner: ObjectIdentifier(permID.Assigner)}, nil
}

func parseHardwareModuleName(der []byte) (HardwareModuleName, error) {
	var hardwareModuleName asn1HardwareModuleName
	if _, err := asn1.UnmarshalWithParams(der, &hardwareModuleName, "explicit,tag:0"); err != nil {
		return HardwareModuleName{}, fmt.Errorf("failed unmarshaling der data: %w", err)
	}
	return HardwareModuleName{Type: ObjectIdentifier(hardwareModuleName.Type), SerialNumber: hardwareModuleName.SerialNumber}, nil
}

// Borrowed from the x509 package.
func forEachSAN(extension []byte, callback func(ext asn1.RawValue) error) error {
	var seq asn1.RawValue
	rest, err := asn1.Unmarshal(extension, &seq)
	if err != nil {
		return err
	} else if len(rest) != 0 {
		return errors.New("x509: trailing data after X.509 extension")
	}
	if !seq.IsCompound || seq.Tag != 16 || seq.Class != 0 {
		return asn1.StructuralError{Msg: "bad SAN sequence"}
	}

	rest = seq.Bytes
	for len(rest) > 0 {
		var v asn1.RawValue
		rest, err = asn1.Unmarshal(rest, &v)
		if err != nil {
			return err
		}

		if err := callback(v); err != nil {
			return err
		}
	}

	return nil
}
