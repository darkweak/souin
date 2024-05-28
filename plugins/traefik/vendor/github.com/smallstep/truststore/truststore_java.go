// Copyright (c) 2018 The truststore Authors. All rights reserved.
// Copyright (c) 2018 The mkcert Authors. All rights reserved.

package truststore

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // not used for cryptographic purposes
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// JavaStorePass is the default store password of the keystore.
var JavaStorePass = "changeit"

// JavaTrust implements a Trust for the Java runtime.
type JavaTrust struct {
	keytoolPath string
	cacertsPath string
}

// NewJavaTrust initializes a new JavaTrust if the environment has java installed.
func NewJavaTrust() (*JavaTrust, error) {
	home := os.Getenv("JAVA_HOME")
	if home == "" {
		return nil, ErrTrustNotFound
	}

	var keytoolPath, cacertsPath string
	if runtime.GOOS == "windows" {
		keytoolPath = filepath.Join(home, "bin", "keytool.exe")
	} else {
		keytoolPath = filepath.Join(home, "bin", "keytool")
	}

	if _, err := os.Stat(keytoolPath); err != nil {
		return nil, ErrTrustNotFound
	}

	_, err := os.Stat(filepath.Join(home, "lib", "security", "cacerts"))
	if err == nil {
		cacertsPath = filepath.Join(home, "lib", "security", "cacerts")
	}

	_, err = os.Stat(filepath.Join(home, "jre", "lib", "security", "cacerts"))
	if err == nil {
		cacertsPath = filepath.Join(home, "jre", "lib", "security", "cacerts")
	}

	return &JavaTrust{
		keytoolPath: keytoolPath,
		cacertsPath: cacertsPath,
	}, nil
}

// Name implement the Trust interface.
func (t *JavaTrust) Name() string {
	return "java"
}

// Install implements the Trust interface.
func (t *JavaTrust) Install(filename string, cert *x509.Certificate) error {
	args := []string{
		"-importcert", "-noprompt",
		"-keystore", t.cacertsPath,
		"-storepass", JavaStorePass,
		"-file", filename,
		"-alias", uniqueName(cert),
	}

	//nolint:gosec // tolerable risk necessary for function
	cmd := exec.Command(t.keytoolPath, args...)
	if out, err := execKeytool(cmd); err != nil {
		return NewCmdError(err, cmd, out)
	}

	debug("certificate installed properly in Java keystore")
	return nil
}

// Uninstall implements the Trust interface.
func (t *JavaTrust) Uninstall(filename string, cert *x509.Certificate) error {
	args := []string{
		"-delete",
		"-alias", uniqueName(cert),
		"-keystore", t.cacertsPath,
		"-storepass", JavaStorePass,
	}

	//nolint:gosec // tolerable risk necessary for function
	cmd := exec.Command(t.keytoolPath, args...)
	out, err := execKeytool(cmd)
	if bytes.Contains(out, []byte("does not exist")) {
		return nil
	}
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	debug("certificate uninstalled properly from the Java keystore")
	return nil
}

// Exists implements the Trust interface.
func (t *JavaTrust) Exists(cert *x509.Certificate) bool {
	if t == nil {
		return false
	}

	// exists returns true if the given x509.Certificate's fingerprint
	// is in the keytool -list output
	exists := func(c *x509.Certificate, h hash.Hash, keytoolOutput []byte) bool {
		h.Write(c.Raw)
		fp := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		return bytes.Contains(keytoolOutput, []byte(fp))
	}

	//nolint:gosec // tolerable risk necessary for function
	cmd := exec.Command(t.keytoolPath, "-list", "-keystore", t.cacertsPath, "-storepass", JavaStorePass)
	keytoolOutput, err := cmd.CombinedOutput()
	if err != nil {
		debug("failed to execute \"keytool -list\": %s\n\n%s", err, keytoolOutput)
		return false
	}

	// keytool outputs SHA1 and SHA256 (Java 9+) certificates in uppercase hex
	// with each octet pair delimitated by ":". Drop them from the keytool output
	keytoolOutput = bytes.ReplaceAll(keytoolOutput, []byte(":"), nil)

	// pre-Java 9 uses SHA1 fingerprints
	//nolint:gosec // not used for cryptographic purposes
	s1, s256 := sha1.New(), sha256.New()
	return exists(cert, s1, keytoolOutput) || exists(cert, s256, keytoolOutput)
}

// PreCheck implements the Trust interface.
func (t *JavaTrust) PreCheck() error {
	if t != nil {
		return nil
	}
	return fmt.Errorf("define JAVA_HOME environment variable to use the Java trust")
}

// execKeytool will execute a "keytool" command and if needed re-execute
// the command wrapped in 'sudo' to work around file permissions.
func execKeytool(cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.CombinedOutput()
	if err != nil && bytes.Contains(out, []byte("java.io.FileNotFoundException")) && runtime.GOOS != "windows" {
		origArgs := cmd.Args[1:]
		//nolint:gosec // tolerable risk necessary for function
		cmd = exec.Command("sudo", cmd.Path)
		cmd.Args = append(cmd.Args, origArgs...)
		cmd.Env = []string{
			"JAVA_HOME=" + os.Getenv("JAVA_HOME"),
		}
		out, err = cmd.CombinedOutput()
	}
	return out, err
}
