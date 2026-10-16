package uri

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"maps"
	"math/big"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	"go.step.sm/crypto/internal/termutil"
)

// readPIN defines the method used to read a pin, it can be changed for testing
// purposes.
var readPIN = termutil.ReadPassword

// URI implements a parser for a URI format based on the the PKCS #11 URI Scheme
// defined in https://tools.ietf.org/html/rfc7512
//
// These URIs will be used to define the key names in a KMS.
type URI struct {
	*url.URL
	Values url.Values
}

// New creates a new URI from a scheme and key-value pairs.
func New(scheme string, values url.Values) *URI {
	return &URI{
		URL: &url.URL{
			Scheme: scheme,
			Opaque: strings.ReplaceAll(values.Encode(), "&", ";"),
		},
		Values: values,
	}
}

// NewFile creates an uri for a file.
func NewFile(path string) *URI {
	return &URI{
		URL: &url.URL{
			Scheme: "file",
			Path:   path,
		},
	}
}

// NewOpaque returns a uri with the given scheme and the given opaque.
func NewOpaque(scheme, opaque string) *URI {
	return &URI{
		URL: &url.URL{
			Scheme: scheme,
			Opaque: opaque,
		},
	}
}

// HasScheme returns true if the given uri has the given scheme, false otherwise.
func HasScheme(scheme, rawuri string) bool {
	u, err := url.Parse(rawuri)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, scheme)
}

// Parse returns the URI for the given string or an error.
func Parse(rawuri string) (*URI, error) {
	u, err := url.Parse(rawuri)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing %s", rawuri)
	}
	if u.Scheme == "" {
		return nil, errors.Errorf("error parsing %s: scheme is missing", rawuri)
	}
	// Starting with Go 1.17 url.ParseQuery returns an error using semicolon as
	// separator.
	v, err := url.ParseQuery(strings.ReplaceAll(u.Opaque, ";", "&"))
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing %s", rawuri)
	}

	return &URI{
		URL:    u,
		Values: v,
	}, nil
}

// ParseWithScheme returns the URI for the given string only if it has the given
// scheme.
func ParseWithScheme(scheme, rawuri string) (*URI, error) {
	u, err := Parse(rawuri)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(u.Scheme, scheme) {
		return nil, errors.Errorf("error parsing %s: scheme not expected", rawuri)
	}
	return u, nil
}

// String returns the string representation of the URI.
func (u *URI) String() string {
	if len(u.Values) > 0 {
		u.URL.Opaque = strings.ReplaceAll(u.Values.Encode(), "&", ";")
	}
	return u.URL.String()
}

// Has checks whether a given key is set.
func (u *URI) Has(key string) bool {
	return u.Values.Has(key) || u.URL.Query().Has(key)
}

// Get returns the first value in the uri with the given key, it will return
// empty string if that field is not present.
func (u *URI) Get(key string) string {
	v := u.Values.Get(key)
	if v == "" {
		v = u.URL.Query().Get(key)
	}
	return v
}

// GetBool returns true if a given key has the value "true". It returns false
// otherwise.
func (u *URI) GetBool(key string) bool {
	v := u.Values.Get(key)
	if v == "" {
		v = u.URL.Query().Get(key)
	}
	return strings.EqualFold(v, "true")
}

// GetInt returns the first integer value in the URI with the given key. It
// returns nil if the field is not present or if the value can't be parsed
// as an integer.
func (u *URI) GetInt(key string) *int64 {
	v := u.Values.Get(key)
	if v == "" {
		v = u.URL.Query().Get(key)
	}
	if v == "" {
		return nil
	}
	if i, err := strconv.ParseInt(v, 10, 0); err == nil {
		return &i
	}
	return nil
}

// GetBigInt returns the first [*big.Int] value in the URI with the given key.
// It returns nil if the field is not present. It parses as a hexadecimal
// string if the value starts with 0x (0x12), 0X (0X1A), contains a colon
// (00:01), or contains only hex characters with at least one letter A-F
// (e.g. "0A01"); otherwise it parses as a base-10 number.
func (u *URI) GetBigInt(key string) (*big.Int, error) {
	v := u.Get(key)
	if v == "" {
		return nil, nil //nolint:nilnil // return nil value
	}

	if hx, ok, isHex := hexString(v); ok && isHex {
		if hx == "" {
			return nil, fmt.Errorf("value %q is not a valid hexadecimal number", v)
		}

		b, err := hex.DecodeString(hx)
		if err != nil {
			return nil, err
		}

		return new(big.Int).SetBytes(b), nil
	}

	bi, ok := new(big.Int).SetString(v, 10)
	if !ok {
		return nil, fmt.Errorf("value %q is not a valid number", v)
	}

	return bi, nil
}

// GetEncoded returns the first value in the uri with the given key, it will
// return empty nil if that field is not present or is empty. If the return
// value is hex encoded it will decode it and return it.
func (u *URI) GetEncoded(key string) []byte {
	v := u.Get(key)
	if v == "" {
		return nil
	}
	if hx, ok, _ := hexString(v); ok {
		if b, err := hex.DecodeString(hx); err == nil {
			return b
		}
	}
	return []byte(v)
}

// GetHexEncoded returns the first value in the uri with the given key. It
// returns nil if the field is not present or is empty. It will return an
// error if the the value is not properly hex encoded.
func (u *URI) GetHexEncoded(key string) ([]byte, error) {
	v := u.Get(key)
	if v == "" {
		return nil, nil
	}

	hx, ok, _ := hexString(v)
	if !ok || hx == "" {
		return nil, fmt.Errorf("value %q is not a valid hexadecimal number", v)
	}

	return hex.DecodeString(hx)
}

// Set sets the key to value. It replaces any existing values.
func (u *URI) Set(key, value string) {
	u.Values.Set(key, value)
}

// Pin returns the pin encoded in the url. It will read the pin from the
// pin-value or the pin-source attributes.
func (u *URI) Pin() string {
	if value := u.Get("pin-value"); value != "" {
		return value
	}
	if path := u.Get("pin-source"); path != "" {
		if b, err := readFile(path); err == nil {
			return string(bytes.TrimRightFunc(b, unicode.IsSpace))
		}
	}
	if u.Has("pin-prompt") {
		prompt := "Enter PIN:"
		if s := u.Get("pin-prompt"); s != "" {
			prompt = s
		}
		if b, err := readPIN(prompt); err == nil {
			return string(bytes.TrimRightFunc(b, unicode.IsSpace))
		}
	}
	return ""
}

// Read returns the raw content of the file in the given attribute key. This
// method will return nil if the key is missing.
func (u *URI) Read(key string) ([]byte, error) {
	path := u.Get(key)
	if path == "" {
		return nil, nil
	}
	return readFile(path)
}

// Values returns the [url.Values] merging the values in the opaque and query
// string of the given [*URI].
func Values(u *URI) url.Values {
	uv := url.Values{}
	maps.Copy(uv, u.Values)
	for k, v := range u.URL.Query() {
		if !uv.Has(k) {
			uv[k] = v
			continue
		}
		for _, s := range v {
			uv.Add(k, s)
		}
	}
	return uv
}

// hexString returns a clean hexadecimal string, a boolean indicating if s is a
// valid hexadecimal string, and a boolean indicating if s was explicitly
// identifiable as hexadecimal. If s starts with 0x (0x12), 0X (0X1A), or
// contains colons (01:1A) it will remove them. The third boolean is true if s
// had a 0x/0X prefix, contained colons, or contained at least one letter A-F
// (010A). The string is prefixed with 0 if its length is odd.
func hexString(s string) (string, bool, bool) {
	hx := strings.TrimPrefix(s, "0x")
	hx = strings.TrimPrefix(hx, "0X")
	hx = strings.ReplaceAll(hx, ":", "")
	changed := (len(s) != len(hx))

	if len(hx)%2 != 0 {
		hx = "0" + hx
	}

	valid, hasLetter := isValidHexString(hx)
	if !valid {
		return "", false, false
	}

	return hx, valid, (changed || hasLetter)
}

// isValidHexString returns two booleans, the first indicating s contains only
// hexadecimal characters and the second if at least one letter (a-f or A-F),
// indicating it should be treated as hex.
func isValidHexString(s string) (bool, bool) {
	var hasLetter bool
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
			hasLetter = true
		case c >= 'A' && c <= 'F':
			hasLetter = true
		default:
			return false, false
		}
	}
	return true, hasLetter
}

func readFile(path string) ([]byte, error) {
	u, err := url.Parse(path)
	if err == nil && (u.Scheme == "" || u.Scheme == "file") {
		switch {
		case u.Path != "":
			path = u.Path
		case u.Opaque != "":
			path = u.Opaque
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %s", path)
	}
	return b, nil
}
