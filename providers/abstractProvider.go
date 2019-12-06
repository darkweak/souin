package providers

import "crypto/tls"

type CommonProvider struct {
	Certificates map[string]Certificate
}

type Certificate struct {
	certificate string
	key         string
}

func InitProviders(certificates *CommonProvider, tlsconfig *tls.Config, configChannel *chan int) {
	TraefikInitProvider(certificates, tlsconfig, configChannel)
}
