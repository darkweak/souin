// Copyright (c) 2018 The truststore Authors. All rights reserved.
// Copyright (c) 2018 The mkcert Authors. All rights reserved.

package truststore

import (
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"os"
	"os/exec"

	plist "howett.net/plist"
)

var (
	// NSSProfile is the path of the Firefox profiles.
	NSSProfile = os.Getenv("HOME") + "/Library/Application Support/Firefox/Profiles/*"

	// CertutilInstallHelp is the command to run on macOS to add NSS support.
	CertutilInstallHelp = "brew install nss"
)

// https://github.com/golang/go/issues/24652#issuecomment-399826583
var trustSettings []interface{}
var _, _ = plist.Unmarshal(trustSettingsData, &trustSettings)
var trustSettingsData = []byte(`
<array>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAED
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>sslServer</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAEC
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>basicX509</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
</array>
`)

func installPlatform(filename string, cert *x509.Certificate) error {
	cmd := exec.Command("sudo", "security", "add-trusted-cert", "-d", "-k", "/Library/Keychains/System.keychain", filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	// Make trustSettings explicit, as older Go does not know the defaults.
	// https://github.com/golang/go/issues/24652
	plistFile, err := os.CreateTemp("", "trust-settings")
	if err != nil {
		return wrapError(err, "failed to create temp file")
	}
	defer os.Remove(plistFile.Name())

	//nolint:gosec // tolerable risk necessary for function
	cmd = exec.Command("sudo", "security", "trust-settings-export", "-d", plistFile.Name())
	out, err = cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	plistData, err := os.ReadFile(plistFile.Name())
	if err != nil {
		return wrapError(err, "failed to read trust settings")
	}

	var plistRoot map[string]interface{}
	_, err = plist.Unmarshal(plistData, &plistRoot)
	if err != nil {
		return wrapError(err, "failed to parse trust settings")
	}
	if v, ok := plistRoot["trustVersion"].(uint64); v != 1 || !ok {
		return fmt.Errorf("unsupported trust settings version: %v", plistRoot["trustVersion"])
	}

	trustList := plistRoot["trustList"].(map[string]interface{})
	rootSubjectASN1, _ := asn1.Marshal(cert.Subject.ToRDNSequence())
	for key := range trustList {
		entry := trustList[key].(map[string]interface{})
		if _, ok := entry["issuerName"]; !ok {
			continue
		}
		issuerName := entry["issuerName"].([]byte)
		if !bytes.Equal(rootSubjectASN1, issuerName) {
			continue
		}
		entry["trustSettings"] = trustSettings
		break
	}

	plistData, err = plist.MarshalIndent(plistRoot, plist.XMLFormat, "\t")
	if err != nil {
		return wrapError(err, "failed to serialize trust settings")
	}

	err = os.WriteFile(plistFile.Name(), plistData, 0600)
	if err != nil {
		return wrapError(err, "failed to write trust settings")
	}

	//nolint:gosec // tolerable risk necessary for function
	cmd = exec.Command("sudo", "security", "trust-settings-import", "-d", plistFile.Name())
	out, err = cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	debug("certificate installed properly in macOS keychain")
	return nil
}

func uninstallPlatform(filename string, cert *x509.Certificate) error {
	cmd := exec.Command("sudo", "security", "remove-trusted-cert", "-d", filename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return NewCmdError(err, cmd, out)
	}

	debug("certificate uninstalled properly from macOS keychain")
	return nil
}
