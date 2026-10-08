// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"path/filepath"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

// SetPicker binds the native host's dialogs without coupling core operations to a GUI.
func SetPicker(b *Bridge, pick func(string) (string, error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pick = pick
}

// Choose returns only an explicitly selected path; cancellation is an empty result.
func (b *Bridge) Choose(kind string) (string, error) {
	if kind != "workspace" && kind != "media" && kind != "subtitle" && kind != "output" {
		return "", contracts.Fail("invalid_request")
	}
	b.mu.RLock()
	pick := b.pick
	b.mu.RUnlock()
	if pick == nil {
		return "", contracts.Fail("unavailable")
	}
	return pick(kind)
}

// SelectWorkspace uses the same workspace contract as the CLI. Selecting a
// workspace changes the client view, not durable jobs in the previously selected one.
func (b *Bridge) SelectWorkspace(path, name string, create bool) contracts.Response {
	if !filepath.IsAbs(path) || len(path) > 4096 || create && (strings.TrimSpace(name) == "" || len(name) > 256) {
		return (&Bridge{}).failed(contracts.Fail("invalid_request"))
	}
	var w *workspace.Workspace
	var err error
	if create {
		w, err = workspace.Init(path, name)
	} else {
		w, err = workspace.Open(path)
	}
	if err != nil {
		return (&Bridge{}).failed(err)
	}
	b.mu.Lock()
	old := b.tickets
	b.tickets = make(map[string]playbackTicket)
	b.Workspace = w
	b.mu.Unlock()
	for _, ticket := range old {
		b.closeTicket(ticket)
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID,
		Result: map[string]any{"workspace_id": w.Config.WorkspaceID, "display_name": w.Config.DisplayName, "path": w.Root}}
}

// Selected snapshots client state for host integration without racing a switch.
func SelectedWorkspace(b *Bridge) *workspace.Workspace {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Workspace
}
