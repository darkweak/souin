package sshutil

import (
	"text/template"

	"go.step.sm/crypto/internal/templates"
)

// templateFuncs holds the functions registered by the application. It is
// separate from the X.509 registry, so registering for one kind of certificate
// does not affect the other.
var templateFuncs = templates.NewRegistry(func() map[string]struct{} {
	names := map[string]struct{}{}
	for name := range builtinFuncMap(new(TemplateError)) {
		names[name] = struct{}{}
	}
	return names
})

// RegisterTemplateFunc adds fn to the functions available to SSH certificate
// templates. It returns an error if name is already registered or built in.
//
// It behaves as [go.step.sm/crypto/x509util.RegisterTemplateFunc] does, over a
// separate registry; an application that wants a function in both calls both.
func RegisterTemplateFunc(name string, fn any) error {
	return templateFuncs.Register(name, fn)
}

// ReplaceTemplateFunc adds fn to the functions available to SSH certificate
// templates, replacing a built-in or previously registered function with the
// same name. Use [RegisterTemplateFunc] unless the replacement is intended.
func ReplaceTemplateFunc(name string, fn any) error {
	return templateFuncs.Replace(name, fn)
}

// UnregisterTemplateFunc removes a registered function. It returns true if a
// function was removed.
func UnregisterTemplateFunc(name string) bool {
	return templateFuncs.Unregister(name)
}

// builtinFuncMap returns the functions provided by this package, excluding
// those registered by the application.
func builtinFuncMap(err *TemplateError) template.FuncMap {
	return templates.GetFuncMap(&err.Message)
}
