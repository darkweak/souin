package souin

import (
	"github.com/darkweak/souin/plugins"
)

type overrideWriter struct {
	*plugins.CustomWriter
}

func (o *overrideWriter) Send() {}
