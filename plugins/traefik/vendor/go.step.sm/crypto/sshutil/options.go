package sshutil

import (
	"bytes"
	"encoding/base64"
	"os"
	"text/template"

	"github.com/pkg/errors"

	"go.step.sm/crypto/internal/templates"
)

// Options are the options that can be passed to NewCertificate.
type Options struct {
	CertBuffer *bytes.Buffer
}

func (o *Options) apply(cr CertificateRequest, opts []Option) (*Options, error) {
	for _, fn := range opts {
		if err := fn(cr, o); err != nil {
			return o, err
		}
	}
	return o, nil
}

// Option is the type used as a variadic argument in NewCertificate.
type Option func(cr CertificateRequest, o *Options) error

// GetFuncMap returns the list of functions used by the templates. It will
// return all the functions supported by "sprig.TxtFuncMap()" but exclude "env"
// and "expandenv", removed to avoid the leak of information.
func GetFuncMap() template.FuncMap {
	return getFuncMap(new(TemplateError))
}

func getFuncMap(err *TemplateError) template.FuncMap {
	return templates.GetFuncMap(&err.Message)
}

// WithTemplate is an options that executes the given template text with the
// given data.
func WithTemplate(text string, data TemplateData) Option {
	return func(cr CertificateRequest, o *Options) error {
		terr := new(TemplateError)
		funcMap := getFuncMap(terr)
		// Parse template
		tmpl, err := template.New("template").Funcs(funcMap).Parse(text)
		if err != nil {
			return errors.Wrapf(err, "error parsing template")
		}

		buf := new(bytes.Buffer)
		data.SetCertificateRequest(cr)
		if err := tmpl.Execute(buf, data); err != nil {
			if terr.Message != "" {
				return terr
			}
			return errors.Wrapf(err, "error executing template")
		}
		o.CertBuffer = buf
		return nil
	}
}

// WithTemplateBase64 is an options that executes the given template base64
// string with the given data.
func WithTemplateBase64(s string, data TemplateData) Option {
	return func(cr CertificateRequest, o *Options) error {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return errors.Wrap(err, "error decoding template")
		}
		fn := WithTemplate(string(b), data)
		return fn(cr, o)
	}
}

// WithTemplateFile is an options that reads the template file and executes it
// with the given data.
func WithTemplateFile(path string, data TemplateData) Option {
	return func(cr CertificateRequest, o *Options) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "error reading %s", path)
		}
		fn := WithTemplate(string(b), data)
		return fn(cr, o)
	}
}
