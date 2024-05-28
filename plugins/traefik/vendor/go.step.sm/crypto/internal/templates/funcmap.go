package templates

import (
	"errors"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// GetFuncMap returns the list of functions provided by sprig. It changes the
// function "fail" to set the given string, this way we can report template
// errors directly to the template without having the wrapper that text/template
// adds.
//
// sprig "env" and "expandenv" functions are removed to avoid the leak of
// information.
func GetFuncMap(failMessage *string) template.FuncMap {
	m := sprig.TxtFuncMap()
	delete(m, "env")
	delete(m, "expandenv")
	m["fail"] = func(msg string) (string, error) {
		*failMessage = msg
		return "", errors.New(msg)
	}
	return m
}
