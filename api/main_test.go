package api

import (
	"fmt"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"testing"
)

func TestInitialize(t *testing.T) {
	config := tests.MockConfiguration()
	prs := providers.InitializeProvider(config)

	endpoints := Initialize(prs, config)

	if len(endpoints) != 1 {
		errors.GenerateError(t, fmt.Sprintf("Endpoints length should be 1, %d received", len(endpoints)))
	}
	if !endpoints[0].IsEnabled() {
		errors.GenerateError(t, fmt.Sprintf("Endpoint should be enabled"))
	}
}
