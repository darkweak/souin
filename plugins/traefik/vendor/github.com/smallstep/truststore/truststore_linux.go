// Copyright (c) 2018 The truststore Authors. All rights reserved.
// Copyright (c) 2018 The mkcert Authors. All rights reserved.

package truststore

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	// NSSProfile is the path of the Firefox profiles.
	NSSProfile = os.Getenv("HOME") + "/.mozilla/firefox/*"

	// CertutilInstallHelp is the command to run on linux to add NSS support.
	CertutilInstallHelp = `apt install libnss3-tools" or "yum install nss-tools`

	// SystemTrustFilename is the format used to name the root certificates.
	SystemTrustFilename string

	// SystemTrustCommand is the command used to update the system truststore.
	SystemTrustCommand []string
)

func init() {
	switch {
	case pathExists("/etc/pki/ca-trust/source/anchors/"):
		SystemTrustFilename = "/etc/pki/ca-trust/source/anchors/%s.pem"
		SystemTrustCommand = []string{"update-ca-trust", "extract"}
	case pathExists("/usr/local/share/ca-certificates/"):
		SystemTrustFilename = "/usr/local/share/ca-certificates/%s.crt"
		SystemTrustCommand = []string{"update-ca-certificates"}
	case pathExists("/usr/share/pki/trust/anchors/"):
		SystemTrustFilename = "/usr/share/pki/trust/anchors/%s.crt"
		SystemTrustCommand = []string{"update-ca-certificates"}
	case pathExists("/etc/ca-certificates/trust-source/anchors/"):
		SystemTrustFilename = "/etc/ca-certificates/trust-source/anchors/%s.crt"
		SystemTrustCommand = []string{"trust", "extract-compat"}
	case pathExists("/etc/ssl/certs/"):
		SystemTrustFilename = "/etc/ssl/certs/%s.crt"
		SystemTrustCommand = []string{"trust", "extract-compat"}
	}
	if SystemTrustCommand != nil {
		_, err := exec.LookPath(SystemTrustCommand[0])
		if err != nil {
			SystemTrustCommand = nil
		}
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func systemTrustFilename(cert *x509.Certificate) string {
	return fmt.Sprintf(SystemTrustFilename, strings.ReplaceAll(uniqueName(cert), " ", "_"))
}

func installPlatform(filename string, cert *x509.Certificate) error {
	if SystemTrustCommand == nil {
		return ErrNotSupported
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	cmd := CommandWithSudo("tee", systemTrustFilename(cert))
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	cmd = CommandWithSudo(SystemTrustCommand...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	debug("certificate installed properly in linux trusts")
	return nil
}

func uninstallPlatform(filename string, cert *x509.Certificate) error {
	if SystemTrustCommand == nil {
		return ErrNotSupported
	}

	cmd := CommandWithSudo("rm", "-f", systemTrustFilename(cert))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	cmd = CommandWithSudo(SystemTrustCommand...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	debug("certificate uninstalled properly from linux trusts")
	return nil
}

func CommandWithSudo(cmd ...string) *exec.Cmd {
	if _, err := exec.LookPath("sudo"); err != nil {
		//nolint:gosec // tolerable risk necessary for function
		return exec.Command(cmd[0], cmd[1:]...)
	}
	//nolint:gosec // tolerable risk necessary for function
	return exec.Command("sudo", append([]string{"--"}, cmd...)...)
}
