package errors

import (
	"testing"
)

// GenerateError Syntactic sugar to display errors
func GenerateError(t *testing.T, text string) {
	t.Errorf("An error occurred : %s", text)
}

// CanceledRequestContextError is the error to handle request cancellation
type CanceledRequestContextError struct{}

func (c *CanceledRequestContextError) Error() string {
	return "The user canceled the request"
}
