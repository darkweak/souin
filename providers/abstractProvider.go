package providers

import "crypto/tls"

// CommonProvider contains a Certificate map
type CommonProvider struct {
	Certificates map[string]Certificate
}

// A Certificate is composed of a certificate and a key
type Certificate struct {
	certificate string
	key         string
}

// InitProviders function allow to init certificates and be able to exploit data as needed
func InitProviders(certificates *CommonProvider, tlsconfig *tls.Config, configChannel *chan int) {
	TraefikInitProvider(certificates, tlsconfig, configChannel)
}
