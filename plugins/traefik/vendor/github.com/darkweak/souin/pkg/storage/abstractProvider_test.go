package storage

import (
	"testing"

	"github.com/darkweak/souin/tests"
)

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	p := InitializeProvider(c)
	err := p.Init()
	if nil != err {
		t.Error("Init shouldn't crash")
	}
}
