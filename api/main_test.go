package api

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/rfc"
	"github.com/darkweak/souin/tests"
)

func TestInitialize(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	c := tests.MockConfiguration(tests.BaseConfiguration)
	provider := providers.InitializeProvider(c)
	transport := rfc.NewTransport(provider, ykeys.InitializeYKeys(c.GetYkeys()), surrogate.InitializeSurrogate(c))

	endpoints := Initialize(transport, config)

	if len(endpoints) != 2 {
		errors.GenerateError(t, fmt.Sprintf("Endpoints length should be 1, %d received", len(endpoints)))
	}
	if !endpoints[0].IsEnabled() {
		errors.GenerateError(t, "Endpoint should be enabled")
	}
}
