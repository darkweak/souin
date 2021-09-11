package api

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func TestInitialize(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(config)

	endpoints := Initialize(prs, config, ykeys.InitializeYKeys(config.Ykeys))

	if len(endpoints) != 2 {
		errors.GenerateError(t, fmt.Sprintf("Endpoints length should be 1, %d received", len(endpoints)))
	}
	if !endpoints[0].IsEnabled() {
		errors.GenerateError(t, "Endpoint should be enabled")
	}
}
