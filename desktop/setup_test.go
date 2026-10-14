// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/firstsetup"
	"path/filepath"
	"testing"
)

func TestNativeSetupDraftUsesSharedStore(t *testing.T) {
	profile := t.TempDir()
	t.Setenv("APPDATA", profile)
	t.Setenv("XDG_CONFIG_HOME", profile)
	b := &Bridge{}
	target := filepath.Join(t.TempDir(), "fresh library")
	raw, _ := json.Marshal(firstsetup.BootstrapInput{Path: target, Plan: json.RawMessage(`{"display_name":"Fixture"}`), Stage: "workspace", Status: "active"})
	id := contracts.ID()
	r := b.SetupDraft("save", string(raw), id)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	first := r.Result.(firstsetup.Draft)
	replayed := b.SetupDraft("save", string(raw), id)
	if replayed.Error != nil || replayed.Result.(firstsetup.Draft).Plan.WorkspaceID != first.Plan.WorkspaceID {
		t.Fatalf("%+v", replayed)
	}
	show := b.SetupDraft("show", "{}", contracts.ID())
	if show.Error != nil || show.Result.(firstsetup.Draft).Revision != 1 {
		t.Fatalf("%+v", show)
	}
	if r := b.SetupDraft("location", `{"path":"","password":"SECRET_SENTINEL"}`, contracts.ID()); r.Error == nil {
		t.Fatal("unknown field accepted")
	}
}
