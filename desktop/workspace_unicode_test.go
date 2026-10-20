// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestSelectWorkspaceUsesPublicUnicodeDisplayNameBound(t *testing.T) {
	profile := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(profile, "config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(profile, "local"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(profile, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(profile, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(profile, "cache"))
	for _, tc := range []struct{ name, label string }{
		{"cjk-86", strings.Repeat("漢", 86)},
		{"cjk-256", strings.Repeat("漢", 256)},
		{"emoji-256", strings.Repeat("😀", 256)},
		{"combining-256", strings.Repeat("e\u0301", 128)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "library")
			bridge := &Bridge{}
			response := bridge.SelectWorkspace(path, tc.label, true)
			if response.Error != nil || !contracts.ValidID(response.WorkspaceID) {
				t.Fatalf("valid Unicode name rejected: %v", response.Error)
			}
			selected := SelectedWorkspace(bridge)
			if selected == nil || selected.Config.DisplayName != tc.label {
				t.Fatal("selected name changed")
			}
			reopened, err := workspace.Open(path)
			if err != nil || reopened.Config.DisplayName != tc.label {
				t.Fatalf("persisted name changed: %v", err)
			}
		})
	}
	for _, name := range []string{strings.Repeat("漢", 257), strings.Repeat("😀", 257), strings.Repeat("e\u0301", 128) + "a", string([]byte{0xff}), " \n "} {
		path := filepath.Join(t.TempDir(), "must-not-exist")
		bridge := &Bridge{}
		response := bridge.SelectWorkspace(path, name, true)
		if response.Error == nil || response.Error.Code != "invalid_request" {
			t.Fatalf("invalid name accepted: %v", response.Error)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("invalid name initialized a directory")
		}
		if SelectedWorkspace(bridge) != nil {
			t.Fatal("invalid name selected a workspace")
		}
	}
}
