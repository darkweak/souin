package cache

import (
	"testing"
)

// Syntactic sugar to display errors
func GenerateError(t *testing.T, text string) {
	t.Errorf("An error occurred : %s", text)
}
