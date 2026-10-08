// SPDX-License-Identifier: Apache-2.0
package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/workspace"
)

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
