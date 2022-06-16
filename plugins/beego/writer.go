package beego

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	beegoCtx "github.com/beego/beego/v2/server/web/context"
	"github.com/darkweak/souin/plugins"
)

type (
	beegoWriterDecorator struct {
		*plugins.CustomWriter
		Response *http.Response
		ctx      *beegoCtx.Context
		buf      *bytes.Buffer
	}
)

func (b *beegoWriterDecorator) Header() http.Header {
	if b.Response.Header == nil {
		b.Response.Header = http.Header{}
	}

	return b.Response.Header
}

func (b *beegoWriterDecorator) Write(d []byte) (int, error) {
	b.ctx.Output.SetStatus(b.Response.StatusCode)
	b.buf.Write(d)
	b.Response.Body = ioutil.NopCloser(bytes.NewBuffer(d))

	if !b.ctx.ResponseWriter.Started {
		return b.Send()
	}

	return len(d), nil
}

func (b *beegoWriterDecorator) Send() (int, error) {
	for h, v := range b.Response.Header {
		if len(v) > 0 {
			b.ctx.Output.Header(h, strings.Join(v, ", "))
		}
	}
	b.ctx.Output.SetStatus(b.Response.StatusCode)
	var d []byte

	if b.Response.Body != nil {
		d, _ = ioutil.ReadAll(b.Response.Body)
	}
	b.Response.Body = ioutil.NopCloser(b.buf)
	return len(d), b.ctx.Output.Body(b.buf.Bytes())
}

func (b *beegoWriterDecorator) WriteHeader(code int) {
	if b.Response == nil {
		b.Response = &http.Response{
			Header: http.Header{},
		}
	}
	if code != 0 {
		b.Response.StatusCode = code
	}
}
