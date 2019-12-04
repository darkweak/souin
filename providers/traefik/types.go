package traefik

type DomainAcmeFile struct {
	Main string   `json:"Main"`
	SANs []string `json:"SANs"`
}

type CertificateAcmeFile struct {
	Certificate string         `json:"Certificate"`
	Domain      DomainAcmeFile `json:"Domain"`
	Key         string         `json:"Key"`
}

type BodyAcmeFile struct {
	Contact []string `json:"contact"`
	Status  string   `json:"status"`
}

type RegistrationAcmeFile struct {
	Body BodyAcmeFile `json:"body"`
	Uri  string       `json:"uri"`
}

type AccountAcmeFile struct {
	Email        string               `json:"Email"`
	KeyType      string               `json:"KeyType"`
	PrivateKey   string               `json:"PrivateKey"`
	Registration RegistrationAcmeFile `json:"Registration"`
}

type AcmeFile struct {
	Account      AccountAcmeFile       `json:"Account"`
	Certificates []CertificateAcmeFile `json:"Certificates"`
}
