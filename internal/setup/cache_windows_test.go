//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPinnedSourceCannotBeReplacedWhileInstallerConsumesIt(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.msi")
	if err := os.WriteFile(source, []byte("original source"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := pinFile(source)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.WriteFile(source, []byte("substituted source"), 0600); err == nil {
		t.Fatal("pinned source remained writable")
	}
	if err := os.Rename(source, filepath.Join(dir, "moved.msi")); err == nil {
		t.Fatal("pinned source remained replaceable")
	}
}
func TestCacheCleanupRemovesOnlyItsKnownSourceFiles(t *testing.T) {
	c := cache{root: t.TempDir()}
	dir := filepath.Join(c.root, testPackage.Product)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := atomicJSON(filepath.Join(dir, "source.json"), testPackage); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source.msi"), []byte("owned source"), 0600); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, "operator notes.txt")
	if err := os.WriteFile(foreign, []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.clean(testPackage.Product); err == nil {
		t.Fatal("nonempty directory cleanup should report retained extra data")
	}
	data, err := os.ReadFile(foreign)
	if err != nil || string(data) != "retain" {
		t.Fatal("cleanup changed unrelated data", err)
	}
}
func TestAtomicStateRefusesRedirectedTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	link := filepath.Join(dir, "link.json")
	if err := os.WriteFile(target, []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("fixture symlink capability unavailable")
	}
	if err := atomicJSON(link, map[string]bool{"overwrite": true}); err == nil {
		t.Fatal("accepted redirected state")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "retain" {
		t.Fatal("redirected state changed target", err)
	}
}
