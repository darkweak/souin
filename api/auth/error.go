package auth

type tokenError struct {
	found bool
}

func generateMessage(s string) string {
	return "An error occurred, " + s
}

func (t *tokenError) Error() string {
	if t.found {
		return generateMessage("Invalid request")
	}
	return generateMessage("Token not found")
}

type signatureError struct{}

func (s *signatureError) Error() string {
	return generateMessage("Impossible to sign the JWT")
}
