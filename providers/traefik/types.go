package traefik

// DomainAcmeFile structure is composed of DNS Main and SANs
type DomainAcmeFile struct {
	Main string   `json:"Main"`
	SANs []string `json:"SANs"`
}

// CertificateAcmeFile structure is composed of Certificate, DomainAcmeFile and Key
type CertificateAcmeFile struct {
	Certificate string         `json:"Certificate"`
	Domain      DomainAcmeFile `json:"Domain"`
	Key         string         `json:"Key"`
}

// BodyAcmeFile structure is composed of Contact, Status
type BodyAcmeFile struct {
	Contact []string `json:"contact"`
	Status  string   `json:"status"`
}

// RegistrationAcmeFile structure is composed of Body, URI
type RegistrationAcmeFile struct {
	Body BodyAcmeFile `json:"body"`
	URI  string       `json:"uri"`
}

// AccountAcmeFile structure is composed of Email, KeyType, PrivateKey, Registration
type AccountAcmeFile struct {
	Email        string               `json:"Email"`
	KeyType      string               `json:"KeyType"`
	PrivateKey   string               `json:"PrivateKey"`
	Registration RegistrationAcmeFile `json:"Registration"`
}

// AcmeFile structure is Træfik acme.json hierachy
type AcmeFile struct {
	Account      AccountAcmeFile       `json:"Account"`
	Certificates []CertificateAcmeFile `json:"Certificates"`
}
