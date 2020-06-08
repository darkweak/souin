package providers

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
)

const MEMORYVALUE = "My first data"

func TestIShouldBeAbleToReadAndWriteDataInMemory(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	err := client.Set("Test", []byte(MEMORYVALUE))
	if err != nil {
		errors.GenerateError(t, "Impossible to set memory variable")
	}
	res, err := client.Get("Test")
	if err != nil {
		errors.GenerateError(t, "Retrieving data from memory")
	}
	if MEMORYVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, MEMORYVALUE))
	}
}
