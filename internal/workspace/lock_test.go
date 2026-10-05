package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnerSurvivesControlChangeAndReadOnlyMetadata(t *testing.T) {
	w, err := Init(t.TempDir(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(w.Root, "state")
	if err := os.Mkdir(selected, 0700); err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectory(selected, true); err != nil {
		t.Fatal(err)
	}
	w.Config.ControlDirectory = selected
	data, _ := json.Marshal(w.Config)
	metadata := filepath.Join(w.Root, ".insonic")
	if err := os.WriteFile(filepath.Join(metadata, "workspace.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(metadata, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(metadata, 0700)
	opened, err := Open(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := opened.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	if _, err := os.Lstat(filepath.Join(metadata, "owner.lock")); !os.IsNotExist(err) {
		t.Fatal("owner lock mutated metadata")
	}
	opened.Control = metadata
	if _, err := opened.Lock(); err == nil {
		t.Fatal("control edit admitted a second owner")
	}
}
