// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"os"
	"path/filepath"
	"strings"
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
