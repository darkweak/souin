package linkedca

import "context"

type contextKeyType int

const (
	_ contextKeyType = iota
	adminContextKey
	provisionerContextKey
	externalAccountKeyContextKey
)

// NewContextWithAdmin returns a copy of ctx which carries an Admin.
func NewContextWithAdmin(ctx context.Context, admin *Admin) context.Context {
	return context.WithValue(ctx, adminContextKey, admin)
}

// AdminFromContext returns an Admin if the ctx carries one and a
// bool indicating if an Admin is carried by the ctx.
func AdminFromContext(ctx context.Context) (a *Admin, ok bool) {
	if a, ok = ctx.Value(adminContextKey).(*Admin); a == nil {
		return nil, false
	}
	return
}

// MustAdminFromContext returns the Admin ctx carries.
//
// MustAdminFromContext panics in case ctx carries no Admin.
func MustAdminFromContext(ctx context.Context) *Admin {
	return ctx.Value(adminContextKey).(*Admin)
}

// NewContextWithProvisioner returns a copy of ctx which carries a Provisioner.
func NewContextWithProvisioner(ctx context.Context, provisioner *Provisioner) context.Context {
	return context.WithValue(ctx, provisionerContextKey, provisioner)
}

// ProvisionerFromContext returns a Provisioner if the ctx carries one and a
// bool indicating if a Provisioner is carried by the ctx.
func ProvisionerFromContext(ctx context.Context) (p *Provisioner, ok bool) {
	if p, ok = ctx.Value(provisionerContextKey).(*Provisioner); p == nil {
		return nil, false
	}
	return
}

// MustProvisionerFromContext returns the Provisioner ctx carries.
//
// MustProvisionerFromContext panics in case ctx carries no Provisioner.
func MustProvisionerFromContext(ctx context.Context) *Provisioner {
	return ctx.Value(provisionerContextKey).(*Provisioner)
}

// NewContextWithExternalAccountKey returns a copy of ctx which carries an EABKey.
func NewContextWithExternalAccountKey(ctx context.Context, k *EABKey) context.Context {
	return context.WithValue(ctx, externalAccountKeyContextKey, k)
}

// ExternalAccountKeyFromContext returns the EABKey if the ctx carries
// one and a bool indicating if an EABKey is carried by the ctx.
func ExternalAccountKeyFromContext(ctx context.Context) (k *EABKey, ok bool) {
	if k, ok = ctx.Value(externalAccountKeyContextKey).(*EABKey); k == nil {
		return nil, false
	}
	return
}

// MustExternalAccountKeyFromContext returns the EABKey ctx carries.
//
// MustExternalAccountKeyFromContext panics in case ctx carries no EABKey.
func MustExternalAccountKeyFromContext(ctx context.Context) *EABKey {
	return ctx.Value(externalAccountKeyContextKey).(*EABKey)
}
