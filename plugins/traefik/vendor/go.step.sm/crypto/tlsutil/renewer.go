package tlsutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// RenewFunc defines the type of the functions used to get a new tls
// certificate.
type RenewFunc func() (*tls.Certificate, *tls.Config, error)

// MinCertDuration is the minimum validity of a certificate.
var MinCertDuration = time.Minute

// Renewer automatically renews a tls certificate using a RenewFunc.
//
//nolint:gocritic // ignore exposedSyncMutex
type Renewer struct {
	sync.RWMutex
	RenewFunc    RenewFunc
	cert         *tls.Certificate
	config       *tls.Config
	timer        *time.Timer
	renewBefore  time.Duration
	renewJitter  time.Duration
	certNotAfter time.Time
}

type renewerOptions func(r *Renewer) error

// WithRenewBefore modifies a tls renewer by setting the renewBefore attribute.
func WithRenewBefore(b time.Duration) func(r *Renewer) error {
	return func(r *Renewer) error {
		r.renewBefore = b
		return nil
	}
}

// WithRenewJitter modifies a tls renewer by setting the renewJitter attribute.
func WithRenewJitter(j time.Duration) func(r *Renewer) error {
	return func(r *Renewer) error {
		r.renewJitter = j
		return nil
	}
}

// NewRenewer creates a TLS renewer for the given cert. It will use the given
// RenewFunc to get a new certificate when required.
func NewRenewer(cert *tls.Certificate, config *tls.Config, fn RenewFunc, opts ...renewerOptions) (*Renewer, error) {
	r := &Renewer{
		RenewFunc:    fn,
		cert:         cert,
		config:       config.Clone(),
		certNotAfter: cert.Leaf.NotAfter,
	}

	// Use renewer methods.
	if r.config.GetCertificate == nil {
		r.config.GetCertificate = r.GetCertificate
	}
	if r.config.GetClientCertificate == nil {
		r.config.GetClientCertificate = r.GetClientCertificate
	}
	if r.config.GetConfigForClient == nil {
		r.config.GetConfigForClient = r.GetConfigForClient
	}

	for _, f := range opts {
		if err := f(r); err != nil {
			return nil, fmt.Errorf("error applying options: %w", err)
		}
	}

	period := cert.Leaf.NotAfter.Sub(cert.Leaf.NotBefore)
	if period < MinCertDuration {
		return nil, fmt.Errorf("period must be greater than or equal to %s, but got %v", MinCertDuration, period)
	}
	// By default we will try to renew the cert before 2/3 of the validity
	// period have expired.
	if r.renewBefore == 0 {
		r.renewBefore = period / 3
	}
	// By default we set the jitter to 1/20th of the validity period.
	if r.renewJitter == 0 {
		r.renewJitter = period / 20
	}

	return r, nil
}

// GetConfig returns the current tls.Config.
func (r *Renewer) GetConfig() *tls.Config {
	return r.getConfigForClient()
}

// Run starts the certificate renewer for the given certificate.
func (r *Renewer) Run() {
	r.Lock()
	next := r.nextRenewDuration(r.certNotAfter)
	r.timer = time.AfterFunc(next, r.renewCertificate)
	r.Unlock()
}

// RunContext starts the certificate renewer for the given certificate.
func (r *Renewer) RunContext(ctx context.Context) {
	r.Run()
	go func() {
		<-ctx.Done()
		r.Stop()
	}()
}

// Stop prevents the renew timer from firing.
func (r *Renewer) Stop() bool {
	r.Lock()
	defer r.Unlock()
	if r.timer != nil {
		return r.timer.Stop()
	}
	return true
}

// GetCertificate returns the current server certificate.
//
// This method is set in the tls.Config GetCertificate property.
func (r *Renewer) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return r.getCertificate(), nil
}

// GetClientCertificate returns the current client certificate.
//
// This method is set in the tls.Config GetClientCertificate property.
func (r *Renewer) GetClientCertificate(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return r.getCertificate(), nil
}

// GetConfigForClient returns the tls.Config used per request.
//
// This method is set in the tls.Config GetConfigForClient property.
func (r *Renewer) GetConfigForClient(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	return r.getConfigForClient(), nil
}

// getCertificate returns the certificate using a read-only lock. It will
// automatically renew the certificate if it has expired.
func (r *Renewer) getCertificate() *tls.Certificate {
	r.RLock()
	// Force certificate renewal if the timer didn't run.
	// This is an special case that can happen after a computer sleep.
	if time.Now().After(r.certNotAfter) {
		r.RUnlock()
		r.renewCertificate()
		r.RLock()
	}
	cert := r.cert
	r.RUnlock()
	return cert
}

func (r *Renewer) getConfigForClient() *tls.Config {
	r.RLock()
	// Force certificate renewal if the timer didn't run.
	// This is an special case that can happen after a computer sleep.
	if time.Now().After(r.certNotAfter) {
		r.RUnlock()
		r.renewCertificate()
		r.RLock()
	}
	config := r.config
	r.RUnlock()
	return config
}

// setCertificate updates the certificate using a read-write lock. It also
// updates certNotAfter with 1m of delta; this will force the renewal of the
// certificate if it is about to expire.
func (r *Renewer) setCertificate(cert *tls.Certificate, config *tls.Config) {
	r.Lock()
	r.cert = cert
	r.config = config
	r.certNotAfter = cert.Leaf.NotAfter
	// Use renewer methods.
	if r.config.GetCertificate == nil {
		r.config.GetCertificate = r.GetCertificate
	}
	if r.config.GetClientCertificate == nil {
		r.config.GetClientCertificate = r.GetClientCertificate
	}
	if r.config.GetConfigForClient == nil {
		r.config.GetConfigForClient = r.GetConfigForClient
	}
	r.Unlock()
}

func (r *Renewer) renewCertificate() {
	var next time.Duration
	cert, config, err := r.RenewFunc()
	if err != nil {
		next = r.renewJitter / 2
		next += time.Duration(mathRandInt63n(int64(next)))
	} else {
		r.setCertificate(cert, config)
		next = r.nextRenewDuration(cert.Leaf.NotAfter)
	}
	r.Lock()
	r.timer.Reset(next)
	r.Unlock()
}

func (r *Renewer) nextRenewDuration(notAfter time.Time) time.Duration {
	d := time.Until(notAfter) - r.renewBefore
	n := mathRandInt63n(int64(r.renewJitter))
	d -= time.Duration(n)
	if d < 0 {
		d = 0
	}
	return d
}

//nolint:gosec // not used for security reasons
func mathRandInt63n(n int64) int64 {
	return rand.Int63n(n)
}
