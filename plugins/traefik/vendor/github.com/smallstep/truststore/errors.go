// Copyright (c) 2018 The truststore Authors. All rights reserved.

package truststore

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
)

var (
	// ErrNotSupported is the error to indicate that the install of the
	// certificate is not supported on the system.
	ErrNotSupported = errors.New("install is not supported on this system")

	// ErrNotFound is the error to indicate that a cert was not found.
	ErrNotFound = errors.New("no certs found")

	// ErrInvalidCertificate is the error to indicate that a cert contains bad data.
	ErrInvalidCertificate = errors.New("invalid PEM data")

	// ErrTrustExists is the error returned when a trust already exists.
	ErrTrustExists = errors.New("trust already exists")

	// ErrTrustNotFound is the error returned when a trust does not exists.
	ErrTrustNotFound = errors.New("trust does not exists")

	// ErrTrustNotSupported is the error returned when a trust is not supported.
	ErrTrustNotSupported = errors.New("trust not supported")
)

// CmdError is the error used when an executable fails.
type CmdError struct {
	err error
	cmd *exec.Cmd
	out []byte
}

// NewCmdError creates a new CmdError.
func NewCmdError(err error, cmd *exec.Cmd, out []byte) *CmdError {
	return &CmdError{
		err: err,
		cmd: cmd,

		out: out,
	}
}

// Error implements the error interface.
func (e *CmdError) Error() string {
	name := filepath.Base(e.cmd.Path)
	return fmt.Sprintf("failed to execute %s: %v", name, e.err)
}

// Err returns the internal error.
func (e *CmdError) Err() error {
	return e.err
}

// Cmd returns the command executed.
func (e *CmdError) Cmd() *exec.Cmd {
	return e.cmd
}

// Out returns the output of the command.
func (e *CmdError) Out() []byte {
	return e.out
}

func wrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
