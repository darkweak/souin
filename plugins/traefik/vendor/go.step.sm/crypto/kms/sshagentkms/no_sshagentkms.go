//go:build nosshagentkms
// +build nosshagentkms

package sshagentkms

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.step.sm/crypto/kms/apiv1"
)

func init() {
	apiv1.Register(apiv1.SSHAgentKMS, func(ctx context.Context, opts apiv1.Options) (apiv1.KeyManager, error) {
		name := filepath.Base(os.Args[0])
		return nil, errors.Errorf("unsupported kms type 'sshagentkms': %s is compiled without SSH Agent KMS support", name)
	})
}
