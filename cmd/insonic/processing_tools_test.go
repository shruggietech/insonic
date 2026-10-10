// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/workspace"
)

func TestProcessingToolsCLIRejectsExplicitZeroBudgetBeforePublication(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "configuration")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	config := map[string]any{"kind": "processing-tools", "schema_version": "1.0.0", "cueson": map[string]any{"executable": filepath.Join(t.TempDir(), "unselected-cueson"), "executable_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, "processing": map[string]any{"threads": 0}}
	raw, _ := json.Marshal(config)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if processingToolsCommand(w, path, true) == 0 {
		t.Fatal("explicit zero budget admitted")
	}
	if _, err := os.Stat(filepath.Join(w.Control, "processing-tools.json")); !os.IsNotExist(err) {
		t.Fatal("invalid configuration was published")
	}
}
