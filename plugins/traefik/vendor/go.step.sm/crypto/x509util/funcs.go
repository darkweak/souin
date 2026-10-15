package x509util

import (
	"text/template"

	"go.step.sm/crypto/internal/templates"
)

// templateFuncs holds the functions registered by the application. The reserved
// set is this package's own function map, so a registration cannot shadow one.
var templateFuncs = templates.NewRegistry(func() map[string]struct{} {
	names := map[string]struct{}{}
	for name := range builtinFuncMap(new(TemplateError)) {
		names[name] = struct{}{}
	}
	return names
})

// RegisterTemplateFunc adds fn to the functions available to X.509 certificate
// templates. It returns an error if name is already registered or built in.
//
// Register during start-up. "text/template" resolves function names when it
// parses, so a template rendered before the call will fail to parse.
//
// A function receives only its own arguments. One that needs the template data
// takes it as a parameter, which the template passes as "$" rather than ".",
// as the dot is rebound inside a range block:
//
//	{{ cel "device.serial" $ | toJson }}
func RegisterTemplateFunc(name string, fn any) error {
	return templateFuncs.Register(name, fn)
}

// ReplaceTemplateFunc adds fn to the functions available to X.509 certificate
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
	funcMap := templates.GetFuncMap(&err.Message)
	// asn1 methods
	funcMap["asn1Enc"] = asn1Encode
	funcMap["asn1Marshal"] = asn1Marshal
	funcMap["asn1Seq"] = asn1Sequence
	funcMap["asn1Set"] = asn1Set
	return funcMap
}
