// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"bytes"
	"context"
	"github.com/shruggietech/insonic/internal/backendsetup"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	local "github.com/shruggietech/insonic/internal/runtime"
	"path/filepath"
	"time"
)

func (b *Bridge) beginOperation(id string, timeout time.Duration) (context.Context, func(), error) {
	if !contracts.ValidID(id) {
		return nil, nil, contracts.Fail("invalid_request")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	b.mu.Lock()
	if b.operations == nil {
		b.operations = map[string]context.CancelFunc{}
	}
	if _, ok := b.operations[id]; ok {
		b.mu.Unlock()
		cancel()
		return nil, nil, contracts.Fail("conflict")
	}
	b.operations[id] = cancel
	b.mu.Unlock()
	return ctx, func() { cancel(); b.mu.Lock(); delete(b.operations, id); b.mu.Unlock() }, nil
}
func (b *Bridge) CancelOperation(id string) bool {
	b.mu.RLock()
	cancel := b.operations[id]
	b.mu.RUnlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// BackendSetup is the shared pre-owner bootstrap contract. Protected input is separate from the nonsecret plan.
func (b *Bridge) BackendSetup(action, path, plan, protected, mode, id string) contracts.Response {
	ctx, finish, e := b.beginOperation(id, 35*time.Second)
	if e != nil {
		return b.failed(e)
	}
	defer finish()
	if action == "inspect" {
		b.mu.RLock()
		w := b.Workspace
		b.mu.RUnlock()
		if w == nil {
			return b.failed(contracts.Fail("not_found"))
		}
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, Result: map[string]any{"configuration": w.Config, "path": w.Root}}
	}
	if action != "plan" && action != "preflight" && action != "provision" {
		return b.failed(contracts.Fail("invalid_request"))
	}
	if action != "plan" && !filepath.IsAbs(path) {
		return b.failed(contracts.Fail("invalid_request"))
	}
	c, e := backendsetup.Build([]byte(plan))
	if e != nil {
		r := b.failed(contracts.SafeError(e))
		r.Result = backendsetup.Failure(e)
		return r
	}
	if action == "plan" {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: map[string]any{"configuration": c, "provisioned": false}}
	}
	input, e := credentialcmd.ReadInput(bytes.NewBufferString(protected))
	if e != nil {
		return b.failed(e)
	}
	defer input.Close()
	if action == "preflight" {
		out, e := backendsetup.Preflight(ctx, path, c, backendsetup.Transient{Input: input})
		if e != nil {
			return b.failed(e)
		}
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: out}
	}
	w, e := backendsetup.ProvisionMode(ctx, path, c, input, mode)
	if e != nil {
		return b.failed(e)
	}
	if mode == "session" || mode == "vault" {
		if e = local.EnsureInput(ctx, w, b.CLIExecutable, []byte(protected)); e != nil {
			return b.failed(e)
		}
	}
	// Select only after successful provisioning; jobs in the previous workspace retain their owner.
	b.mu.Lock()
	old := b.tickets
	b.tickets = map[string]playbackTicket{}
	b.Workspace = w
	b.mu.Unlock()
	for _, ticket := range old {
		b.closeTicket(ticket)
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, Result: map[string]any{"configuration": w.Config, "workspace_id": w.Config.WorkspaceID, "display_name": w.Config.DisplayName, "path": w.Root, "provisioned": true}}
}
