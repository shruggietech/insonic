// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), "insonic & %TEMP% é")
	if e := os.Mkdir(d, 0700); e != nil {
		t.Fatal(e)
	}
	for _, n := range []string{"insonic.exe", "insonic-desktop.exe", "insonic-session.cmd"} {
		if e := os.WriteFile(filepath.Join(d, n), []byte("fixture"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	return d
}

func directoryAlias(t *testing.T, target, name string) {
	t.Helper()
	if e := os.Symlink(target, name); e != nil {
		if runtime.GOOS == "windows" && errors.Is(e, syscall.Errno(1314)) {
			t.Skip("Windows symbolic-link privilege is unavailable")
		}
		t.Fatal(e)
	}
}

func TestDestinationRejectsAliasIntoLibraryBeforeCreatingMissingSuffix(t *testing.T) {
	library := t.TempDir()
	control := filepath.Join(library, ".insonic")
	nested := filepath.Join(library, "media", "existing")
	if e := os.MkdirAll(control, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(control, "workspace.json"), []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(nested, 0700); e != nil {
		t.Fatal(e)
	}
	alias := filepath.Join(t.TempDir(), "alias outside lexical library")
	directoryAlias(t, nested, alias)
	destination := filepath.Join(alias, "missing parent", "program")
	if e := ValidateDestination(destination); e == nil || !strings.Contains(e.Error(), "outside a library") {
		t.Fatalf("accepted aliased library ancestry: %v", e)
	}
	if _, e := os.Stat(filepath.Join(nested, "missing parent")); !os.IsNotExist(e) {
		t.Fatal("validation created a directory", e)
	}
}

func TestDestinationPreservesNonLibraryAliasesAndRejectsAliasedVolumeRoot(t *testing.T) {
	real := t.TempDir()
	alias := filepath.Join(t.TempDir(), "valid alias é")
	directoryAlias(t, real, alias)
	for _, destination := range []string{alias, filepath.Join(alias, "missing", "program")} {
		if e := ValidateDestination(destination); e != nil {
			t.Fatal("rejected non-library alias", e)
		}
	}
	resolved, e := canonicalPath(filepath.Join(alias, "missing", "program"))
	realPhysical, physicalError := filepath.EvalSymlinks(real)
	if e != nil || physicalError != nil || resolved != filepath.Join(realPhysical, "missing", "program") {
		t.Fatal("missing suffix lost physical ancestry", resolved, e)
	}
	rootAlias := filepath.Join(t.TempDir(), "volume alias")
	directoryAlias(t, filepath.VolumeName(real)+string(os.PathSeparator), rootAlias)
	if e := ValidateDestination(rootAlias); e == nil || !strings.Contains(e.Error(), "volume root") {
		t.Fatal("accepted aliased volume root", e)
	}
}

func TestActiveImageComparisonUsesCanonicalDirectoriesAndRejectsSiblingPrefixes(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "program")
	other := filepath.Join(base, "program-other")
	for _, d := range []string{real, other} {
		if e := os.Mkdir(d, 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(d, "insonic.exe"), []byte("never executed"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	alias := filepath.Join(t.TempDir(), "program alias")
	directoryAlias(t, real, alias)
	physical, e := canonicalPath(alias)
	if e != nil {
		t.Fatal(e)
	}
	for _, image := range []string{filepath.Join(real, "insonic.exe"), filepath.Join(alias, "insonic.exe")} {
		resolved, e := canonicalProcessImage(image)
		if e != nil || !containsPath(physical, resolved) {
			t.Fatal("missed active image under aliased installation", resolved, e)
		}
	}
	resolved, e := canonicalProcessImage(filepath.Join(other, "insonic.exe"))
	if e != nil || containsPath(physical, resolved) {
		t.Fatal("matched unrelated sibling prefix", resolved, e)
	}
}

func TestDanglingAliasCannotBecomeAnAssumedFreshDirectory(t *testing.T) {
	alias := filepath.Join(t.TempDir(), "dangling")
	directoryAlias(t, filepath.Join(t.TempDir(), "missing target"), alias)
	if _, e := canonicalPath(filepath.Join(alias, "program")); e == nil {
		t.Fatal("accepted dangling alias")
	}
	if e := ValidateDestination(filepath.Join(alias, "program")); e == nil {
		t.Fatal("accepted unresolved installation path")
	}
}
func TestCLIUsesOnlyFixedCommandAndPrivatePathEnvironment(t *testing.T) {
	d := fixture(t)
	p, e := Build(d, "cli", filepath.Join(t.TempDir(), "System32"))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(p.CommandLine, d) || !strings.Contains(p.CommandLine, "%INSONIC_LAUNCH_SCRIPT%") || p.Environment["INSONIC_LAUNCH_CLI"] != filepath.Join(d, "insonic.exe") {
		t.Fatalf("unsafe shell path interpolation: %#v", p)
	}
	if p.Executable != filepath.Join(filepath.Dir(p.Executable), "cmd.exe") {
		t.Fatal(p.Executable)
	}
}
func TestDesktopHasNoShellAndMissingClientsFail(t *testing.T) {
	d := fixture(t)
	p, e := Build(d, "desktop", t.TempDir())
	if e != nil || p.Executable != filepath.Join(d, "insonic-desktop.exe") || p.CommandLine != "" {
		t.Fatalf("%#v %v", p, e)
	}
	os.Remove(p.Executable)
	if _, e = Build(d, "desktop", t.TempDir()); e == nil {
		t.Fatal("missing GUI accepted")
	}
	if _, e = Build(d, "unknown", t.TempDir()); e == nil {
		t.Fatal("unknown mode accepted")
	}
}
func TestDestinationRejectsLibraryAncestorRootAndUnownedFiles(t *testing.T) {
	base := t.TempDir()
	control := filepath.Join(base, ".insonic")
	os.Mkdir(control, 0700)
	os.WriteFile(filepath.Join(control, "workspace.json"), []byte("{}"), 0600)
	if e := ValidateDestination(filepath.Join(base, "program")); e == nil {
		t.Fatal("library descendant accepted")
	}
	if e := ValidateDestination(filepath.VolumeName(base) + string(os.PathSeparator)); e == nil {
		t.Fatal("volume root accepted")
	}
	other := t.TempDir()
	os.WriteFile(filepath.Join(other, "user.txt"), []byte("user"), 0600)
	if e := ValidateDestination(other); e == nil {
		t.Fatal("unowned directory accepted")
	}
	if e := ValidateDestination(filepath.Join(t.TempDir(), "new directory é")); e != nil {
		t.Fatal(e)
	}
}
func TestOwnedDirectoryAndSelection(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "insonic-msi-ownership.json"), []byte(`{"kind":"windows-msi-installation","family":"insonic"}`), 0600)
	if e := ValidateDestination(d); e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{"cli", "desktop", "both", "Desktop,CLI"} {
		if e := ValidateSelection(s); e != nil {
			t.Fatal(s, e)
		}
	}
	for _, s := range []string{"", "none", "Desktop,Unknown"} {
		if e := ValidateSelection(s); e == nil {
			t.Fatal("invalid selection accepted", s)
		}
	}
}

func TestMaintenanceIncludesRegisteredRootWhenUpgradeMovesProgramFiles(t *testing.T) {
	previous := t.TempDir()
	target := filepath.Join(t.TempDir(), "new program location")
	roots, e := maintenanceRoots(target, previous)
	if e != nil || len(roots) != 2 {
		t.Fatal("moving upgrade lost an installation root", roots, e)
	}
	oldImage, e := canonicalProcessImage(filepath.Join(previous, "active-owned-fixture.exe"))
	if e != nil {
		t.Fatal(e)
	}
	if containsPath(roots[0], oldImage) || !containsPath(roots[1], oldImage) {
		t.Fatal("previous active image was not covered", roots)
	}
	alias := filepath.Join(t.TempDir(), "previous alias")
	directoryAlias(t, previous, alias)
	roots, e = maintenanceRoots(alias, previous)
	if e != nil || len(roots) != 1 {
		t.Fatal("same physical root was not deduplicated", roots, e)
	}
	for _, invalid := range []string{"relative-registration", filepath.VolumeName(previous) + string(os.PathSeparator)} {
		if _, e := maintenanceRoots(target, invalid); e == nil {
			t.Fatal("invalid previous registration accepted", invalid)
		}
	}
}

func TestShortcutCollisionsRequireExistingMSIOwnership(t *testing.T) {
	link := filepath.Join(t.TempDir(), "insonic CLI.lnk")
	if e := shortcutCollision(link, false); e != nil {
		t.Fatal("absent integration was blocked", e)
	}
	if e := os.WriteFile(link, []byte("unrelated shortcut fixture, never executed"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := shortcutCollision(link, false); e == nil {
		t.Fatal("unrelated exact-name shortcut would be overwritten")
	}
	if e := shortcutCollision(link, true); e != nil {
		t.Fatal("registered MSI integration cannot be maintained", e)
	}
	data, e := os.ReadFile(link)
	if e != nil || string(data) != "unrelated shortcut fixture, never executed" {
		t.Fatal("read-only shortcut check changed bytes", e)
	}
	directory := filepath.Join(t.TempDir(), "directory.lnk")
	if e := os.Mkdir(directory, 0700); e != nil {
		t.Fatal(e)
	}
	if e := shortcutCollision(directory, true); e == nil {
		t.Fatal("non-file collision was accepted")
	}
}
