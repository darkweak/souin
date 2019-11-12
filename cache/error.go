package cache

import (
	"testing"
)

func generateError(t *testing.T, text string) {
	t.Errorf("An error occurred : %s", text)
}
