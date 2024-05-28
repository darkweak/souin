// Copyright (c) 2022 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tscert fetches HTTPS certs from the local machine's
// Tailscale daemon (tailscaled).
package tscert

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tailscale/tscert/internal/paths"
	"github.com/tailscale/tscert/internal/safesocket"
)

var (
	// TailscaledSocket is the tailscaled Unix socket. It's used by the TailscaledDialer.
	TailscaledSocket = paths.DefaultTailscaledSocket()

	// TailscaledSocketSetExplicitly reports whether the user explicitly set TailscaledSocket.
	TailscaledSocketSetExplicitly bool

	// TailscaledDialer is the DialContext func that connects to the local machine's
	// tailscaled or equivalent.
	TailscaledDialer = defaultDialer
)

func defaultDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	if addr != "local-tailscaled.sock:80" {
		return nil, fmt.Errorf("unexpected URL address %q", addr)
	}
	// TODO: make this part of a safesocket.ConnectionStrategy
	if !TailscaledSocketSetExplicitly {
		// On macOS, when dialing from non-sandboxed program to sandboxed GUI running
		// a TCP server on a random port, find the random port. For HTTP connections,
		// we don't send the token. It gets added in an HTTP Basic-Auth header.
		if port, _, err := safesocket.LocalTCPPortAndToken(); err == nil {
			var d net.Dialer
			return d.DialContext(ctx, "tcp", "localhost:"+strconv.Itoa(port))
		}
	}
	s := safesocket.DefaultConnectionStrategy(TailscaledSocket)
	// The user provided a non-default tailscaled socket address.
	// Connect only to exactly what they provided.
	s.UseFallback(false)
	return safesocket.Connect(s)
}

var (
	// tsClient does HTTP requests to the local Tailscale daemon.
	// We lazily initialize the client in case the caller wants to
	// override TailscaledDialer.
	tsClient     *http.Client
	tsClientOnce sync.Once
)

// DoLocalRequest makes an HTTP request to the local machine's Tailscale daemon.
//
// URLs are of the form http://local-tailscaled.sock/localapi/v0/whois?ip=1.2.3.4.
//
// The hostname must be "local-tailscaled.sock", even though it
// doesn't actually do any DNS lookup. The actual means of connecting to and
// authenticating to the local Tailscale daemon vary by platform.
//
// DoLocalRequest may mutate the request to add Authorization headers.
func DoLocalRequest(req *http.Request) (*http.Response, error) {
	tsClientOnce.Do(func() {
		tsClient = &http.Client{
			Transport: &http.Transport{
				DialContext: TailscaledDialer,
			},
		}
	})
	if _, token, err := safesocket.LocalTCPPortAndToken(); err == nil {
		req.SetBasicAuth("", token)
	}
	return tsClient.Do(req)
}

func doLocalRequestNiceError(req *http.Request) (*http.Response, error) {
	res, err := DoLocalRequest(req)
	if err == nil {
		if res.StatusCode == 403 {
			all, _ := ioutil.ReadAll(res.Body)
			return nil, &AccessDeniedError{errors.New(errorMessageFromBody(all))}
		}
		return res, nil
	}
	return nil, err
}

type errorJSON struct {
	Error string
}

// AccessDeniedError is an error due to permissions.
type AccessDeniedError struct {
	err error
}

func (e *AccessDeniedError) Error() string { return fmt.Sprintf("Access denied: %v", e.err) }
func (e *AccessDeniedError) Unwrap() error { return e.err }

// IsAccessDeniedError reports whether err is or wraps an AccessDeniedError.
func IsAccessDeniedError(err error) bool {
	var ae *AccessDeniedError
	return errors.As(err, &ae)
}

// bestError returns either err, or if body contains a valid JSON
// object of type errorJSON, its non-empty error body.
func bestError(err error, body []byte) error {
	var j errorJSON
	if err := json.Unmarshal(body, &j); err == nil && j.Error != "" {
		return errors.New(j.Error)
	}
	return err
}

func errorMessageFromBody(body []byte) string {
	var j errorJSON
	if err := json.Unmarshal(body, &j); err == nil && j.Error != "" {
		return j.Error
	}
	return strings.TrimSpace(string(body))
}

func send(ctx context.Context, method, path string, wantStatus int, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://local-tailscaled.sock"+path, body)
	if err != nil {
		return nil, err
	}
	res, err := doLocalRequestNiceError(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	slurp, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != wantStatus {
		return nil, bestError(err, slurp)
	}
	return slurp, nil
}

func get200(ctx context.Context, path string) ([]byte, error) {
	return send(ctx, "GET", path, 200, nil)
}

// Status is a stripped down version of tailscale.com/ipn/ipnstate.Status
// for the tscert package.
type Status struct {
	// Version is the daemon's long version (see version.Long).
	Version string

	// BackendState is an ipn.State string value:
	//  "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped",
	//  "Starting", "Running".
	BackendState string

	// Health contains health check problems.
	// Empty means everything is good. (or at least that no known
	// problems are detected)
	Health []string

	// TailscaleIPs are the Tailscale IP(s) assigned to this node
	TailscaleIPs []string

	// MagicDNSSuffix is the network's MagicDNS suffix for nodes
	// in the network such as "userfoo.tailscale.net".
	// There are no surrounding dots.
	// MagicDNSSuffix should be populated regardless of whether a domain
	// has MagicDNS enabled.
	MagicDNSSuffix string

	// CertDomains are the set of DNS names for which the control
	// plane server will assist with provisioning TLS
	// certificates. See SetDNSRequest for dns-01 ACME challenges
	// for e.g. LetsEncrypt. These names are FQDNs without
	// trailing periods, and without any "_acme-challenge." prefix.
	CertDomains []string
}

// GetStatus returns a stripped down status from tailscaled. For a full
// version, use tailscale.com/client/tailscale.Status.
func GetStatus(ctx context.Context) (*Status, error) {
	body, err := get200(ctx, "/localapi/v0/status")
	if err != nil {
		return nil, err
	}
	st := new(Status)
	if err := json.Unmarshal(body, st); err != nil {
		return nil, err
	}
	return st, nil
}

// CertPair returns a cert and private key for the provided DNS domain.
//
// It returns a cached certificate from disk if it's still valid.
func CertPair(ctx context.Context, domain string) (certPEM, keyPEM []byte, err error) {
	res, err := send(ctx, "GET", "/localapi/v0/cert/"+domain+"?type=pair", 200, nil)
	if err != nil {
		return nil, nil, err
	}
	// with ?type=pair, the response PEM is first the one private
	// key PEM block, then the cert PEM blocks.
	i := bytes.Index(res, []byte("--\n--"))
	if i == -1 {
		return nil, nil, fmt.Errorf("unexpected output: no delimiter")
	}
	i += len("--\n")
	keyPEM, certPEM = res[:i], res[i:]
	if bytes.Contains(certPEM, []byte(" PRIVATE KEY-----")) {
		return nil, nil, fmt.Errorf("unexpected output: key in cert")
	}
	return certPEM, keyPEM, nil
}

// GetCertificate fetches a TLS certificate for the TLS ClientHello in hi.
//
// It returns a cached certificate from disk if it's still valid.
//
// It's the right signature to use as the value of
// tls.Config.GetCertificate.
func GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hi == nil || hi.ServerName == "" {
		return nil, errors.New("no SNI ServerName")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	name := hi.ServerName
	if !strings.Contains(name, ".") {
		if v, ok := ExpandSNIName(ctx, name); ok {
			name = v
		}
	}
	certPEM, keyPEM, err := CertPair(ctx, name)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// ExpandSNIName expands bare label name into the the most likely actual TLS cert name.
func ExpandSNIName(ctx context.Context, name string) (fqdn string, ok bool) {
	st, err := GetStatus(ctx)
	if err != nil {
		return "", false
	}
	for _, d := range st.CertDomains {
		if len(d) > len(name)+1 && strings.HasPrefix(d, name) && d[len(name)] == '.' {
			return d, true
		}
	}
	return "", false
}
