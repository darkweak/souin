package x509util

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"

	"github.com/pkg/errors"
)

// attributeTypeNames are the subject attributes managed by Go and this package.
// newExtraNames will populate .Insecure.CR.ExtraNames with the attributes not
// present on this map.
var attributeTypeNames = map[string]string{
	"2.5.4.6":  "C",
	"2.5.4.10": "O",
	"2.5.4.11": "OU",
	"2.5.4.3":  "CN",
	"2.5.4.5":  "SERIALNUMBER",
	"2.5.4.7":  "L",
	"2.5.4.8":  "ST",
	"2.5.4.9":  "STREET",
	"2.5.4.17": "POSTALCODE",
}

// oidEmailAddress is the oid of the deprecated emailAddress in the subject.
var oidEmailAddress = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}

// Name is the JSON representation of X.501 type Name, used in the X.509 subject
// and issuer fields.
type Name struct {
	Country            MultiString         `json:"country,omitempty"`
	Organization       MultiString         `json:"organization,omitempty"`
	OrganizationalUnit MultiString         `json:"organizationalUnit,omitempty"`
	Locality           MultiString         `json:"locality,omitempty"`
	Province           MultiString         `json:"province,omitempty"`
	StreetAddress      MultiString         `json:"streetAddress,omitempty"`
	PostalCode         MultiString         `json:"postalCode,omitempty"`
	SerialNumber       string              `json:"serialNumber,omitempty"`
	CommonName         string              `json:"commonName,omitempty"`
	ExtraNames         []DistinguishedName `json:"extraNames,omitempty"`
}

func newName(n pkix.Name) Name {
	return Name{
		Country:            n.Country,
		Organization:       n.Organization,
		OrganizationalUnit: n.OrganizationalUnit,
		Locality:           n.Locality,
		Province:           n.Province,
		StreetAddress:      n.StreetAddress,
		PostalCode:         n.PostalCode,
		SerialNumber:       n.SerialNumber,
		CommonName:         n.CommonName,
		ExtraNames:         newExtraNames(n.Names),
	}
}

// goValue converts Name to its Go representation.
func (n Name) goValue() pkix.Name {
	return pkix.Name{
		Country:            n.Country,
		Organization:       n.Organization,
		OrganizationalUnit: n.OrganizationalUnit,
		Locality:           n.Locality,
		Province:           n.Province,
		StreetAddress:      n.StreetAddress,
		PostalCode:         n.PostalCode,
		SerialNumber:       n.SerialNumber,
		CommonName:         n.CommonName,
		ExtraNames:         fromDistinguishedNames(n.ExtraNames),
	}
}

// UnmarshalJSON implements the json.Unmarshal interface and unmarshals a JSON
// object in the Name struct or a string as just the subject common name.
func (n *Name) UnmarshalJSON(data []byte) error {
	if cn, ok := maybeString(data); ok {
		n.CommonName = cn
		return nil
	}

	type nameAlias Name
	var nn nameAlias
	if err := json.Unmarshal(data, &nn); err != nil {
		return errors.Wrap(err, "error unmarshaling json")
	}
	*n = Name(nn)
	return nil
}

// Subject is the JSON representation of the X.509 subject field.
type Subject Name

func newSubject(n pkix.Name) Subject {
	return Subject(newName(n))
}

// UnmarshalJSON implements the json.Unmarshal interface and unmarshals a JSON
// object in the Subject struct or a string as just the subject common name.
func (s *Subject) UnmarshalJSON(data []byte) error {
	var name Name
	if err := name.UnmarshalJSON(data); err != nil {
		return err
	}
	*s = Subject(name)
	return nil
}

// Set sets the subject in the given certificate.
func (s Subject) Set(c *x509.Certificate) {
	c.Subject = Name(s).goValue()
}

// IsEmpty returns if the subject is empty. Certificates with an empty subject
// must have the subjectAltName extension mark as critical.
func (s Subject) IsEmpty() bool {
	subject := Name(s).goValue()
	if asn1Subject, err := asn1.Marshal(subject.ToRDNSequence()); err == nil {
		return bytes.Equal(asn1Subject, emptyASN1Subject)
	}
	return false
}

// Issuer is the JSON representation of the X.509 issuer field.
type Issuer Name

func newIssuer(n pkix.Name) Issuer {
	return Issuer(newName(n))
}

// UnmarshalJSON implements the json.Unmarshal interface and unmarshals a JSON
// object in the Issuer struct or a string as just the subject common name.
func (i *Issuer) UnmarshalJSON(data []byte) error {
	var name Name
	if err := name.UnmarshalJSON(data); err != nil {
		return err
	}
	*i = Issuer(name)
	return nil
}

// Set sets the issuer in the given certificate.
func (i Issuer) Set(c *x509.Certificate) {
	c.Issuer = Name(i).goValue()
}

// DistinguishedName mirrors the ASN.1 structure AttributeTypeAndValue in RFC
// 5280, Section 4.1.2.4.
type DistinguishedName struct {
	Type  ObjectIdentifier `json:"type"`
	Value interface{}      `json:"value"`
}

// newExtraNames returns a list of DistinguishedName with the attributes not
// present in attributeTypeNames.
func newExtraNames(atvs []pkix.AttributeTypeAndValue) []DistinguishedName {
	var extraNames []DistinguishedName
	for _, atv := range atvs {
		if _, ok := attributeTypeNames[atv.Type.String()]; !ok {
			extraNames = append(extraNames, DistinguishedName{
				Type:  ObjectIdentifier(atv.Type),
				Value: atv.Value,
			})
		}
	}
	return extraNames
}

// fromDistinguishedNames converts a list of DistinguishedName to
// []pkix.AttributeTypeAndValue. Note that this method has a special case to
// encode the deprecated emailAddress field (1.2.840.113549.1.9.1).
func fromDistinguishedNames(dns []DistinguishedName) []pkix.AttributeTypeAndValue {
	var atvs []pkix.AttributeTypeAndValue
	for _, dn := range dns {
		typ := asn1.ObjectIdentifier(dn.Type)
		v, isString := dn.Value.(string)
		if typ.Equal(oidEmailAddress) && isString {
			atvs = append(atvs, pkix.AttributeTypeAndValue{
				Type: typ,
				Value: asn1.RawValue{
					Class: asn1.ClassUniversal,
					Tag:   asn1.TagIA5String,
					Bytes: []byte(v),
				},
			})
		} else {
			atvs = append(atvs, pkix.AttributeTypeAndValue{
				Type:  typ,
				Value: dn.Value,
			})
		}
	}
	return atvs
}
