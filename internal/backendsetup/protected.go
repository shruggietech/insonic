// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
)

// Transient provides newly entered protected values for non-mutating preflight.
// It creates no store, lock file or registration and never serializes values.
type Transient struct{ Input *credentialcmd.Input }

func (t Transient) Resolve(ctx context.Context, id string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if t.Input != nil {
		if value, ok := t.Input.Session[id]; ok {
			return append([]byte(nil), value...), nil
		}
	}
	return nil, contracts.Condition("unavailable", "credential_bootstrap")
}
func (t Transient) Status(ctx context.Context, id string) (string, error) {
	value, e := t.Resolve(ctx, id)
	clear(value)
	if e != nil {
		return "missing", e
	}
	return "transient", nil
}
