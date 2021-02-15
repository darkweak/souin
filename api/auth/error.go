package auth

import (
	"fmt"
)

type tokenError struct {
	found bool
}

func generateMessage(s string) string {
	return fmt.Sprintf("An error occurred, %s", s)
}

func (t *tokenError) Error() string {
	if t.found {
		return generateMessage("Token not found")
	}
	return generateMessage("Invalid request")
}

type signatureError struct {}

func (s *signatureError) Error() string {
	return generateMessage("Impossible to sign the JWT")
}
