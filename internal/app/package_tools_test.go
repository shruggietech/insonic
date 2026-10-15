// SPDX-License-Identifier: Apache-2.0
package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestCatalogViewsDoNotInitializeExecutableTools(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Catalog-only views")
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := os.WriteFile(filepath.Join(w.Control, "media-tools.json"), []byte(`{"invalid":"unused executable configuration"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	response := realRequest(a, "media.list", "", map[string]any{"limit": 10})
	if response.Error != nil {
		t.Fatalf("catalog listing tried to initialize media executables: %v", response.Error)
	}
	response = realRequest(a, "media.show", contracts.ID(), nil)
	if response.Error == nil || response.Error.Code != "not_found" {
		t.Fatalf("catalog lookup was masked by executable configuration: %+v", response.Error)
	}
	if _, err := a.libraryService(); err == nil {
		t.Fatal("execution service stopped validating explicit tool configuration")
	}
}

func TestExplicitWorkspaceToolConfigurationNeverFallsBack(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Explicit tools")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"media-tools.json", "processing-tools.json"} {
		if err := os.WriteFile(filepath.Join(w.Control, name), []byte(`{"invalid":"explicit"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ReadMediaTools(w); err == nil {
		t.Fatal("invalid explicit media configuration silently replaced")
	}
	if _, err := ReadProcessingTools(w); err == nil {
		t.Fatal("invalid explicit processing configuration silently replaced")
	}
}
