// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows && !js
// +build !windows,!js

package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sys/unix"
)

func init() {
	stateFileFunc = stateFileUnix
}

func statePath() string {
	switch runtime.GOOS {
	case "linux":
		if fi, err := os.Stat("/gokrazy"); err == nil && fi.IsDir() {
			return "/perm/tailscaled/tailscaled.state"
		}

		return "/var/lib/tailscale/tailscaled.state"
	case "freebsd", "openbsd":
		return "/var/db/tailscale/tailscaled.state"
	case "darwin":
		return "/Library/Tailscale/tailscaled.state"
	default:
		return ""
	}
}

func stateFileUnix() string {
	path := statePath()
	if path == "" {
		return ""
	}

	try := path
	for i := 0; i < 3; i++ { // check writability of the file, /var/lib/tailscale, and /var/lib
		err := unix.Access(try, unix.O_RDWR)
		if err == nil {
			return path
		}
		try = filepath.Dir(try)
	}

	if os.Getuid() == 0 {
		return ""
	}

	// For non-root users, fall back to $XDG_DATA_HOME/tailscale/*.
	return filepath.Join(xdgDataHome(), "tailscale", "tailscaled.state")
}

func xdgDataHome() string {
	if e := os.Getenv("XDG_DATA_HOME"); e != "" {
		return e
	}
	return filepath.Join(os.Getenv("HOME"), ".local/share")
}

func ensureStateDirPerms(dir string) error {
	if filepath.Base(dir) != "tailscale" {
		return nil
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("expected %q to be a directory; is %v", dir, fi.Mode())
	}
	const perm = 0700
	if fi.Mode().Perm() == perm {
		// Already correct.
		return nil
	}
	return os.Chmod(dir, perm)
}

// LegacyStateFilePath is not applicable to UNIX; it is just stubbed out.
func LegacyStateFilePath() string {
	return ""
}
