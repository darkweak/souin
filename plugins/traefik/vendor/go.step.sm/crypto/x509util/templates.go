package x509util

import (
	"crypto/x509"

	"go.step.sm/crypto/internal/templates"
)

// Variables used to hold template data.
const (
	SubjectKey            = "Subject"
	SANsKey               = "SANs"
	TokenKey              = "Token"
	InsecureKey           = "Insecure"
	UserKey               = "User"
	CertificateRequestKey = "CR"
	AuthorizationCrtKey   = "AuthorizationCrt"
	AuthorizationChainKey = "AuthorizationChain"
	WebhooksKey           = "Webhooks"
)

// TemplateError represents an error in a template produced by the fail
// function.
type TemplateError struct {
	Message string
}

// Error implements the error interface and returns the error string when a
// template executes the `fail "message"` function.
func (e *TemplateError) Error() string {
	return e.Message
}

// ValidateTemplate validates a text template.
func ValidateTemplate(text []byte) error {
	return templates.ValidateTemplate(text, GetFuncMap())
}

// ValidateTemplateData validates that template data is
// valid JSON.
func ValidateTemplateData(data []byte) error {
	return templates.ValidateTemplateData(data)
}

// TemplateData is an alias for map[string]interface{}. It represents the data
// passed to the templates.
type TemplateData map[string]interface{}

// NewTemplateData creates a new map for templates data.
func NewTemplateData() TemplateData {
	return TemplateData{}
}

// CreateTemplateData creates a new TemplateData with the given common name and SANs.
func CreateTemplateData(commonName string, sans []string) TemplateData {
	return TemplateData{
		SubjectKey: Subject{
			CommonName: commonName,
		},
		SANsKey: CreateSANs(sans),
	}
}

// Set sets a key-value pair in the template data.
func (t TemplateData) Set(key string, v interface{}) {
	t[key] = v
}

// SetInsecure sets a key-value pair in the insecure template data.
func (t TemplateData) SetInsecure(key string, v interface{}) {
	if m, ok := t[InsecureKey].(TemplateData); ok {
		m[key] = v
	} else {
		t[InsecureKey] = TemplateData{key: v}
	}
}

// SetSubject sets the given subject in the template data.
func (t TemplateData) SetSubject(v Subject) {
	t.Set(SubjectKey, v)
}

// SetCommonName sets the given common name in the subject in the template data.
func (t TemplateData) SetCommonName(cn string) {
	s, _ := t[SubjectKey].(Subject)
	s.CommonName = cn
	t[SubjectKey] = s
}

// SetSANs sets the given SANs in the template data.
func (t TemplateData) SetSANs(sans []string) {
	t.Set(SANsKey, CreateSANs(sans))
}

// SetSubjectAlternativeNames sets the given sans in the template data.
func (t TemplateData) SetSubjectAlternativeNames(sans ...SubjectAlternativeName) {
	t.Set(SANsKey, sans)
}

// SetToken sets the given token in the template data.
func (t TemplateData) SetToken(v interface{}) {
	t.Set(TokenKey, v)
}

// SetUserData sets the given user provided object in the insecure template
// data.
func (t TemplateData) SetUserData(v interface{}) {
	t.SetInsecure(UserKey, v)
}

// SetAuthorizationCertificate sets the given certificate in the template. This certificate
// is generally present in a token header.
func (t TemplateData) SetAuthorizationCertificate(crt interface{}) {
	t.Set(AuthorizationCrtKey, crt)
}

// SetAuthorizationCertificateChain sets a the given certificate chain in the
// template. These certificates are generally present in a token header.
func (t TemplateData) SetAuthorizationCertificateChain(chain interface{}) {
	t.Set(AuthorizationChainKey, chain)
}

// SetCertificateRequest sets the given certificate request in the insecure
// template data.
func (t TemplateData) SetCertificateRequest(cr *x509.CertificateRequest) {
	t.SetInsecure(CertificateRequestKey, NewCertificateRequestFromX509(cr))
}

// SetWebhook sets the given webhook response in the webhooks template data.
func (t TemplateData) SetWebhook(webhookName string, data interface{}) {
	if webhooksMap, ok := t[WebhooksKey].(map[string]interface{}); ok {
		webhooksMap[webhookName] = data
	} else {
		t[WebhooksKey] = map[string]interface{}{
			webhookName: data,
		}
	}
}

// DefaultLeafTemplate is the default template used to generate a leaf
// certificate.
const DefaultLeafTemplate = `{
	"subject": {{ toJson .Subject }},
	"sans": {{ toJson .SANs }},
{{- if typeIs "*rsa.PublicKey" .Insecure.CR.PublicKey }}
	"keyUsage": ["keyEncipherment", "digitalSignature"],
{{- else }}
	"keyUsage": ["digitalSignature"],
{{- end }}
	"extKeyUsage": ["serverAuth", "clientAuth"]
}`

// DefaultIIDLeafTemplate is the template used by default on instance identity
// provisioners like AWS, GCP or Azure. By default, those provisioners allow the
// SANs provided in the certificate request, but the option `DisableCustomSANs`
// can be provided to force only the verified domains, if the option is true
// `.SANs` will be set with the verified domains.
const DefaultIIDLeafTemplate = `{
	"subject": {"commonName": {{ toJson .Insecure.CR.Subject.CommonName }}},
{{- if .SANs }}
	"sans": {{ toJson .SANs }},
{{- else }}
	"dnsNames": {{ toJson .Insecure.CR.DNSNames }},
	"emailAddresses": {{ toJson .Insecure.CR.EmailAddresses }},
	"ipAddresses": {{ toJson .Insecure.CR.IPAddresses }},
	"uris": {{ toJson .Insecure.CR.URIs }},
{{- end }}
{{- if typeIs "*rsa.PublicKey" .Insecure.CR.PublicKey }}
	"keyUsage": ["keyEncipherment", "digitalSignature"],
{{- else }}
	"keyUsage": ["digitalSignature"],
{{- end }}
	"extKeyUsage": ["serverAuth", "clientAuth"]
}`

// DefaultAdminLeafTemplate is a template used by default by K8sSA and
// admin-OIDC provisioners. This template takes all the SANs and subject from
// the certificate request.
const DefaultAdminLeafTemplate = `{
	"subject": {{ toJson .Insecure.CR.Subject }},
	"dnsNames": {{ toJson .Insecure.CR.DNSNames }},
	"emailAddresses": {{ toJson .Insecure.CR.EmailAddresses }},
	"ipAddresses": {{ toJson .Insecure.CR.IPAddresses }},
	"uris": {{ toJson .Insecure.CR.URIs }},
{{- if typeIs "*rsa.PublicKey" .Insecure.CR.PublicKey }}
	"keyUsage": ["keyEncipherment", "digitalSignature"],
{{- else }}
	"keyUsage": ["digitalSignature"],
{{- end }}
	"extKeyUsage": ["serverAuth", "clientAuth"]
}`

// DefaultIntermediateTemplate is a template that can be used to generate an
// intermediate certificate.
const DefaultIntermediateTemplate = `{
	"subject": {{ toJson .Subject }},
	"keyUsage": ["certSign", "crlSign"],
	"basicConstraints": {
		"isCA": true,
		"maxPathLen": 0
	}
}`

// DefaultRootTemplate is a template that can be used to generate a root
// certificate.
const DefaultRootTemplate = `{
	"subject": {{ toJson .Subject }},
	"issuer": {{ toJson .Subject }},
	"keyUsage": ["certSign", "crlSign"],
	"basicConstraints": {
		"isCA": true,
		"maxPathLen": 1
	}
}`

// CertificateRequestTemplate is a template that will sign the given certificate
// request.
const CertificateRequestTemplate = `{{ toJson .Insecure.CR }}`

// DefaultCertificateRequestTemplate is the templated used by default when
// creating a new certificate request.
const DefaultCertificateRequestTemplate = `{
	"subject": {{ toJson .Subject }},
	"sans": {{ toJson .SANs }}
}`

// DefaultAttestedLeafTemplate is the default template used to generate a leaf
// certificate from an attestation certificate. The main difference with
// "DefaultLeafTemplate" is that the extended key usage is limited to
// "clientAuth".
const DefaultAttestedLeafTemplate = `{
	"subject": {{ toJson .Subject }},
	"sans": {{ toJson .SANs }},
{{- if typeIs "*rsa.PublicKey" .Insecure.CR.PublicKey }}
	"keyUsage": ["keyEncipherment", "digitalSignature"],
{{- else }}
	"keyUsage": ["digitalSignature"],
{{- end }}
	"extKeyUsage": ["clientAuth"]
}`
