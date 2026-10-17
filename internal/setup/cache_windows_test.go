//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"strings"
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
	if err := workspace.SecureDirectory(dir, true); err != nil {
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

func TestProgramDirectoryAliasRetainsLogicalLocationAndPinsCanonicalFiles(t *testing.T) {
	dir := t.TempDir()
	physical := filepath.Join(dir, "physical")
	alias := filepath.Join(dir, "selected alias")
	if err := os.Mkdir(physical, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(physical, alias); err != nil {
		t.Skip("fixture directory alias capability unavailable")
	}
	name := filepath.Join(physical, "owned.bin")
	if err := os.WriteFile(name, []byte("owned"), 0600); err != nil {
		t.Fatal(err)
	}
	c := testChoices
	c.Directory = alias
	target, err := filePath(c, File{Path: "owned.bin", SHA256: strings.Repeat("a", 64), Size: 5})
	expected, resolveErr := filepath.EvalSymlinks(name)
	if err != nil || resolveErr != nil || target != expected || c.Directory != alias {
		t.Fatal(target, err, expected, resolveErr)
	}
	external := filepath.Join(dir, "external.bin")
	if err := os.WriteFile(external, []byte("unrelated"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(physical, "outside.bin")); err != nil {
		t.Fatal(err)
	}
	if _, err := filePath(c, File{Path: "outside.bin", SHA256: strings.Repeat("a", 64), Size: 9}); err == nil {
		t.Fatal("accepted a file outside the program tree")
	}
}
