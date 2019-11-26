package providers

type Provider struct {
	certificates []Certificate
}

type Certificate struct {
	key     string
	payload string
}
