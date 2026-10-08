// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestWorkspaceSelectionUsesSharedIdentityAndRejectsStaleRequests(t *testing.T) {
	b := &Bridge{}
	first := b.SelectWorkspace(filepath.Join(t.TempDir(), "first"), "First", true)
	if first.Error != nil || !contracts.ValidID(first.WorkspaceID) {
		t.Fatalf("create: %#v", first)
	}
	second := b.SelectWorkspace(filepath.Join(t.TempDir(), "second"), "Second", true)
	if second.Error != nil || first.WorkspaceID == second.WorkspaceID {
		t.Fatalf("switch: %#v", second)
	}
	stale := b.Operate(contracts.Request{WorkspaceID: first.WorkspaceID, Operation: "workspace.show"})
	if stale.Error == nil || stale.Error.Code != "workspace_mismatch" {
		t.Fatalf("stale request admitted: %#v", stale)
	}
	if missing := b.SelectWorkspace(filepath.Join(t.TempDir(), "missing"), "", false); missing.Error == nil || SelectedWorkspace(b).Config.WorkspaceID != second.WorkspaceID {
		t.Fatal("failed open changed workspace")
	}
	if b.SelectWorkspace("relative", "Wrong", true).Error == nil {
		t.Fatal("relative workspace admitted")
	}
}

func TestWorkspaceSelectionAndCredentialSnapshotsAreRaceFree(t *testing.T) {
	b := &Bridge{}
	paths := []string{filepath.Join(t.TempDir(), "one"), filepath.Join(t.TempDir(), "two")}
	for _, path := range paths {
		if b.SelectWorkspace(path, "Workspace", true).Error != nil {
			t.Fatal("create")
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b.SelectWorkspace(paths[i%2], "", false)
			b.Operate(contracts.Request{WorkspaceID: contracts.ID(), Operation: "workspace.show"})
			b.Credential([]string{"invalid"}, "")
		}(i)
	}
	wg.Wait()
}

func TestNativeChooserOnlyAcceptsNamedKinds(t *testing.T) {
	b := &Bridge{}
	SetPicker(b, func(kind string) (string, error) { return kind, nil })
	if _, err := b.Choose("arbitrary"); err == nil {
		t.Fatal("unknown dialog admitted")
	}
	if value, err := b.Choose("media"); err != nil || value != "media" {
		t.Fatal("native picker unavailable")
	}
}
