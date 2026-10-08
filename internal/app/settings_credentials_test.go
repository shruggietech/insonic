// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/secrets"
)

func TestDesktopSettingsReadsPersistedCredentialBackend(t *testing.T) {
	for _, mode := range []string{"vault", "session"} {
		t.Run(mode, func(t *testing.T) {
			a := configuredApp(t)
			selected := realRequest(a, "credentials.select", "", map[string]any{"arguments": []string{"select", mode}})
			if selected.Error != nil {
				t.Fatal(selected.Error)
			}
			w := a.Workspace
			a.Close()
			path := filepath.Join(w.Control, "secrets", "selection.json")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := New(w)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			for range 2 {
				shown := realRequest(reopened, "settings.show", "", nil)
				if shown.Error != nil {
					t.Fatal(shown.Error)
				}
				credentials, ok := shown.Result.(map[string]any)["credentials"].(map[string]any)
				if !ok || credentials["backend"] != mode {
					t.Fatalf("persisted %s selection missing from settings: %#v", mode, credentials)
				}
			}
			after, err := os.ReadFile(path)
			persisted, readErr := secrets.ReadSelection(w)
			if err != nil || readErr != nil || persisted != mode || !bytes.Equal(before, after) {
				t.Fatal("settings read changed credential selection", err, readErr, persisted)
			}
		})
	}
}

func TestDesktopSettingsReadPreservesLiveCredentialManager(t *testing.T) {
	a := configuredApp(t)
	shown := realRequest(a, "settings.show", "", nil)
	if shown.Error != nil || shown.Result.(map[string]any)["credentials"].(map[string]any)["backend"] != "native" {
		t.Fatal("default backend unavailable", shown.Error)
	}
	id := contracts.ID()
	loaded := realRequest(a, "credentials.select", "", map[string]any{"arguments": []string{"select", "session"}, "input": map[string]any{"session": map[string]any{id: "fixture value"}}})
	if loaded.Error != nil {
		t.Fatal(loaded.Error)
	}
	shown = realRequest(a, "settings.show", "", nil)
	if shown.Error != nil || shown.Result.(map[string]any)["credentials"].(map[string]any)["backend"] != "session" {
		t.Fatal("current backend unavailable", shown.Error)
	}
	status := realRequest(a, "credentials.status", "", map[string]any{"arguments": []string{"status", id}})
	if status.Error != nil || status.Result.(secrets.CredentialStatus).State != "configured" {
		t.Fatal("settings read reset the live session manager", status.Error)
	}
}
