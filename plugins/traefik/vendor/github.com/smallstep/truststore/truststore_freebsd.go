package truststore

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

var (
	// NSSProfile is the path of the Firefox profiles.
	NSSProfile = os.Getenv("HOME") + "/.mozilla/firefox/*"

	// CertutilInstallHelp is the command to add NSS support.
	CertutilInstallHelp = ""

	// SystemTrustFilename is the format used to name the root certificates.
	SystemTrustFilename string

	// SystemTrustCommand is the command used to update the system truststore.
	SystemTrustCommand []string
)

func init() {
	if !pathExists("/usr/local/etc/ssl/certs") {
		err := os.Mkdir("/usr/local/etc/ssl/certs", 0755)
		if err != nil {
			SystemTrustCommand = nil
			debug(err.Error())
			return
		}
	}
	SystemTrustCommand = []string{"certctl", "rehash"}
	SystemTrustFilename = "/usr/local/etc/ssl/certs/%s.crt"
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func systemTrustFilename(cert *x509.Certificate) string {
	return fmt.Sprintf(SystemTrustFilename, strings.Replace(uniqueName(cert), " ", "_", -1))
}

func installPlatform(filename string, cert *x509.Certificate) error {
	if SystemTrustCommand == nil {
		return ErrNotSupported
	}

	data, err := ioutil.ReadFile(filename)
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

	debug("certificate installed properly in FreeBSD trusts")
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

	debug("certificate uninstalled properly from FreeBSD trusts")
	return nil
}

func CommandWithSudo(cmd ...string) *exec.Cmd {
	if _, err := exec.LookPath("sudo"); err != nil {
		return exec.Command(cmd[0], cmd[1:]...)
	}
	return exec.Command("sudo", append([]string{"--"}, cmd...)...)
}
