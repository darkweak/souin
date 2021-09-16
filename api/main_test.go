package api

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/plugins"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func TestInitialize(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	retriever := plugins.DefaultSouinPluginInitializerFromConfiguration(config)

	endpoints := Initialize(retriever.Transport, config)

	if len(endpoints) != 2 {
		errors.GenerateError(t, fmt.Sprintf("Endpoints length should be 1, %d received", len(endpoints)))
	}
	if !endpoints[0].IsEnabled() {
		errors.GenerateError(t, "Endpoint should be enabled")
	}
}
