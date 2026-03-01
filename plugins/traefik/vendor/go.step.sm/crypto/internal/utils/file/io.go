package fileutils

import (
	"bytes"
	"io"
	"os"
	"unicode"

	"github.com/pkg/errors"

	"go.step.sm/crypto/internal/utils/utfbom"
)

func maybeUnwrap(err error) error {
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return wrapped
	}
	return err
}

// stdinFilename is the name of the file that is used in many command
// line utilities to denote input is to be read from STDIN.
const stdinFilename = "-"

// stdin points to STDIN through os.Stdin.
var stdin = os.Stdin

// ReadFile reads the file identified by filename and returns
// the contents. If filename is equal to "-", it will read from
// STDIN.
func ReadFile(filename string) (b []byte, err error) {
	if filename == stdinFilename {
		filename = "/dev/stdin"
		b, err = io.ReadAll(stdin)
	} else {
		var contents []byte
		contents, err = os.ReadFile(filename)
		if err != nil {
			return nil, errors.Wrapf(maybeUnwrap(err), "error reading %q", filename)
		}
		b, err = io.ReadAll(utfbom.SkipOnly(bytes.NewReader(contents)))
	}
	if err != nil {
		return nil, errors.Wrapf(maybeUnwrap(err), "error reading %q", filename)
	}
	return
}

// ReadPasswordFromFile reads and returns the password from the given filename.
// The contents of the file will be trimmed at the right.
func ReadPasswordFromFile(filename string) ([]byte, error) {
	password, err := ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %s", filename)
	}
	password = bytes.TrimRightFunc(password, unicode.IsSpace)
	return password, nil
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm
// (before umask); otherwise WriteFile truncates it before writing.
//
// It wraps os.WriteFile wrapping the errors.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return errors.Wrapf(maybeUnwrap(err), "error writing %s", filename)
	}
	return nil
}
