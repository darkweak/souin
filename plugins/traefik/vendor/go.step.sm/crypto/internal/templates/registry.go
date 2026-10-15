package templates

import (
	"fmt"
	"maps"
	"reflect"
	"sync"
	"text/template"
)

// Registry holds template functions added by the application, for the
// functions a template needs that this library does not provide.
type Registry struct {
	// reserved is a function so a package can name its own built-ins without
	// this file knowing them. It is resolved once, on first use.
	reserved func() map[string]struct{}
	once     sync.Once
	names    map[string]struct{}

	mu    sync.RWMutex
	funcs map[string]any
}

// NewRegistry returns a Registry that refuses any name returned by reserved.
func NewRegistry(reserved func() map[string]struct{}) *Registry {
	return &Registry{reserved: reserved, funcs: map[string]any{}}
}

// Register adds fn to the registry. It returns an error if name is already
// registered or reserved.
func (r *Registry) Register(name string, fn any) error {
	if err := validate(name, fn); err != nil {
		return err
	}

	r.once.Do(func() { r.names = r.reserved() })
	if _, ok := r.names[name]; ok {
		return fmt.Errorf("template function %q is built in and cannot be replaced", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.funcs[name]; ok {
		return fmt.Errorf("template function %q is already registered", name)
	}
	r.funcs[name] = fn
	return nil
}

// Replace adds fn to the registry, replacing any reserved or previously
// registered function with the same name.
func (r *Registry) Replace(name string, fn any) error {
	if err := validate(name, fn); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = fn
	return nil
}

func validate(name string, fn any) error {
	switch {
	case name == "":
		return fmt.Errorf("template function name is required")
	case fn == nil:
		return fmt.Errorf("template function %q is nil", name)
	case reflect.TypeOf(fn).Kind() != reflect.Func:
		return fmt.Errorf("template function %q is a %s, not a function", name, reflect.TypeOf(fn).Kind())
	case !validName(name):
		return fmt.Errorf("template function name %q is not a valid identifier", name)
	}
	return nil
}

// Unregister removes a function from the registry. It returns true if a
// function was removed.
func (r *Registry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.funcs[name]; !ok {
		return false
	}
	delete(r.funcs, name)
	return true
}

// Apply adds the registered functions to funcMap.
func (r *Registry) Apply(funcMap template.FuncMap) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	maps.Copy(funcMap, r.funcs)
}

// validName reports whether name is a valid Go identifier, as required by
// "text/template".
func validName(name string) bool {
	for i, c := range name {
		switch {
		case c == '_':
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
