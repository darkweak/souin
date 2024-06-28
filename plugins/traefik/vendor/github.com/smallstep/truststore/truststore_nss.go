// Copyright (c) 2018 The truststore Authors. All rights reserved.
// Copyright (c) 2018 The mkcert Authors. All rights reserved.

package truststore

import (
	"crypto/x509"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var nssDB = filepath.Join(os.Getenv("HOME"), ".pki", "nssdb")

// NSSTrust implements a Trust for Firefox or other NSS based applications.
type NSSTrust struct {
	certutilPath string
}

// NewNSSTrust creates a new NSSTrust.
func NewNSSTrust() (*NSSTrust, error) {
	var err error
	var certutilPath string
	switch runtime.GOOS {
	case "darwin":
		certutilPath, err = exec.LookPath("certutil")
		if err != nil {
			cmd := exec.Command("brew", "--prefix", "nss")
			out, err1 := cmd.Output()
			if err1 != nil {
				return nil, NewCmdError(err1, cmd, out)
			}
			certutilPath = filepath.Join(strings.TrimSpace(string(out)), "bin", "certutil")
			if _, err = os.Stat(certutilPath); err != nil {
				return nil, err
			}
		}
	case "linux":
		if certutilPath, err = exec.LookPath("certutil"); err != nil {
			return nil, err
		}
	default:
		return nil, ErrTrustNotSupported
	}

	return &NSSTrust{
		certutilPath: certutilPath,
	}, nil
}

// Name implements the Trust interface.
func (t *NSSTrust) Name() string {
	return "nss"
}

// Install implements the Trust interface.
func (t *NSSTrust) Install(filename string, cert *x509.Certificate) error {
	// install certificate in all profiles
	if forEachNSSProfile(func(profile string) {
		//nolint:gosec // tolerable risk necessary for function
		cmd := exec.Command(t.certutilPath, "-A", "-d", profile, "-t", "C,,", "-n", uniqueName(cert), "-i", filename)
		out, err := cmd.CombinedOutput()
		if err != nil {
			debug("failed to execute \"certutil -A\": %s\n\n%s", err, out)
		}
	}) == 0 {
		return fmt.Errorf("not NSS security databases found")
	}

	// check for the cert in all profiles
	if !t.Exists(cert) {
		return fmt.Errorf("certificate cannot be installed in NSS security databases")
	}

	debug("certificate installed properly in NSS security databases")
	return nil
}

// Uninstall implements the Trust interface.
func (t *NSSTrust) Uninstall(_ string, cert *x509.Certificate) (err error) {
	forEachNSSProfile(func(profile string) {
		if err != nil {
			return
		}
		// skip if not found
		//nolint:gosec // tolerable risk necessary for function
		if err := exec.Command(t.certutilPath, "-V", "-d", profile, "-u", "L", "-n", uniqueName(cert)).Run(); err != nil {
			return
		}
		// delete certificate
		//nolint:gosec // tolerable risk necessary for function
		cmd := exec.Command(t.certutilPath, "-D", "-d", profile, "-n", uniqueName(cert))
		out, err1 := cmd.CombinedOutput()
		if err1 != nil {
			err = NewCmdError(err1, cmd, out)
		}
	})
	if err == nil {
		debug("certificate uninstalled properly from NSS security databases")
	}
	return
}

// Exists implements the Trust interface. Exists checks if the certificate is
// already installed.
func (t *NSSTrust) Exists(cert *x509.Certificate) bool {
	success := true
	if forEachNSSProfile(func(profile string) {
		//nolint:gosec // tolerable risk necessary for function
		err := exec.Command(t.certutilPath, "-V", "-d", profile, "-u", "L", "-n", uniqueName(cert)).Run()
		if err != nil {
			success = false
		}
	}) == 0 {
		success = false
	}
	return success
}

// PreCheck implements the Trust interface.
func (t *NSSTrust) PreCheck() error {
	if t != nil {
		if forEachNSSProfile(func(_ string) {}) == 0 {
			return fmt.Errorf("not NSS security databases found")
		}
		return nil
	}

	if CertutilInstallHelp == "" {
		return fmt.Errorf("note: NSS support is not available on your platform")
	}

	return fmt.Errorf(`warning: "certutil" is not available, install "certutil" with "%s" and try again`, CertutilInstallHelp)
}

func forEachNSSProfile(f func(profile string)) (found int) {
	profiles, _ := filepath.Glob(NSSProfile)
	if _, err := os.Stat(nssDB); err == nil {
		profiles = append(profiles, nssDB)
	}
	if len(profiles) == 0 {
		return
	}
	for _, profile := range profiles {
		if stat, err := os.Stat(profile); err != nil || !stat.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(profile, "cert9.db")); err == nil {
			f("sql:" + profile)
			found++
			continue
		}
		if _, err := os.Stat(filepath.Join(profile, "cert8.db")); err == nil {
			f("dbm:" + profile)
			found++
		}
	}
	return
}
