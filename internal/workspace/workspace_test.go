package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestInitializationDiscoveryAndOwnership(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace with spaces")
	w, err := Init(root, "Example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, "overwrite"); err == nil {
		t.Fatal("overwrote existing workspace")
	}
	child := filepath.Join(root, "nested")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	found, err := Discover(child)
	if err != nil || found.Config.WorkspaceID != w.Config.WorkspaceID {
		t.Fatalf("discovery: %v", err)
	}
	lock, err := w.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := found.Lock(); err == nil {
		t.Fatal("second writer admitted")
	}
	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	lock, err = found.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
}

func TestConcurrentInitPreservesIdentity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "race")
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Init(root, "race"); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("initializers: %d", successes)
	}
	if _, err := Open(root); err != nil {
		t.Fatal(err)
	}
}

func TestRejectIncompatibleAndMalformedConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, "test"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, ".insonic", "workspace.json")
	if err := os.WriteFile(file, []byte(`{"schema_version":"99.0.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); err == nil {
		t.Fatal("incompatible config accepted")
	}
}

func TestAliasAndConfiguredControlDirectory(t *testing.T) {
	root := t.TempDir()
	w, err := Init(root, "alias")
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err == nil {
		other, err := Open(alias)
		if err != nil || other.Root != w.Root {
			t.Fatal("alias not canonical")
		}
		lock, err := w.Lock()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := other.Lock(); err == nil {
			t.Fatal("alias acquired second writer")
		}
		lock.Unlock()
	}
	selected := filepath.Join(root, "selected-control")
	if err := os.Mkdir(selected, 0700); err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectory(selected, true); err != nil {
		t.Fatal(err)
	}
	w.Config.ControlDirectory = selected
	data, _ := json.Marshal(w.Config)
	if err := os.WriteFile(filepath.Join(root, ".insonic", "workspace.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(root)
	canonicalSelected, _ := canonical(selected)
	if err != nil || opened.Control != canonicalSelected {
		t.Fatalf("configured control directory: %v", err)
	}
}

func TestSchemaRejectsBadProfileAndSecretOptions(t *testing.T) {
	root := t.TempDir()
	w, err := Init(root, "schema")
	if err != nil {
		t.Fatal(err)
	}
	w.Config.Profiles.Storage.Configuration["password"] = "fixture-secret"
	data, _ := json.Marshal(w.Config)
	os.WriteFile(filepath.Join(root, ".insonic", "workspace.json"), data, 0600)
	if _, err := Open(root); err == nil {
		t.Fatal("secret-bearing config admitted")
	}
}
