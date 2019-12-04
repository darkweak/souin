package providers

type CommonProvider struct {
	certificates map[string]Certificate
}

type Certificate struct {
	certificate string
	key         string
}

func InitProviders(certificates *CommonProvider) {
	TraefikInitProvider(certificates)
}
