// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentationEntryUsesInstalledIdentity(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "program é & spaces")
	if err := os.MkdirAll(filepath.Join(dir, "help", "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "help", "docs", "index.html")
	os.WriteFile(entry, []byte("<h1>installed help</h1>"), 0600)
	manifest := filepath.Join(dir, "help", "offline-manifest.json")
	os.WriteFile(manifest, []byte(`{"version":"`+contracts.Version+`","entrypoint":"docs/index.html","public_landing_included":false}`), 0600)
	got, err := DocumentationEntry(dir)
	wantInfo, wantErr := os.Stat(entry)
	gotInfo, gotErr := os.Stat(got)
	if err != nil || wantErr != nil || gotErr != nil || !os.SameFile(wantInfo, gotInfo) {
		t.Fatal(got, err)
	}
	for _, bad := range []string{`{"version":"0.0.0","entrypoint":"docs/index.html"}`, `{"version":"` + contracts.Version + `","entrypoint":"../../private.html"}`, `{"version":"` + contracts.Version + `","entrypoint":"https://example.org"}`, `{"version":"` + contracts.Version + `","version":"` + contracts.Version + `","entrypoint":"docs/index.html"}`, strings.Repeat(" ", 65537)} {
		os.WriteFile(manifest, []byte(bad), 0600)
		if _, err := DocumentationEntry(dir); err == nil || !strings.Contains(err.Error(), "Repair") {
			t.Fatal("missing repair correction", err)
		}
	}
	os.Remove(manifest)
	if _, err := DocumentationEntry(dir); err == nil {
		t.Fatal("missing help accepted")
	}
}

func TestDocumentationRejectsEscapedRootAndAcceptsProgramAlias(t *testing.T) {
	program := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "offline-manifest.json"), []byte(`{"version":"`+contracts.Version+`","entrypoint":"index.html"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "index.html"), []byte("outside installation"), 0600); err != nil {
		t.Fatal(err)
	}
	directoryAlias(t, outside, filepath.Join(program, "help"))
	if _, err := DocumentationEntry(program); err == nil {
		t.Fatal("accepted a help root outside the installed program")
	}
	if err := os.Remove(filepath.Join(program, "help")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(program, "help"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"offline-manifest.json", "index.html"} {
		data, _ := os.ReadFile(filepath.Join(outside, name))
		if err := os.WriteFile(filepath.Join(program, "help", name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(t.TempDir(), "program alias é")
	directoryAlias(t, program, alias)
	if _, err := DocumentationEntry(alias); err != nil {
		t.Fatal("legitimate program relocation alias rejected", err)
	}
	manifest := filepath.Join(program, "help", "offline-manifest.json")
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	directoryAlias(t, filepath.Join(outside, "offline-manifest.json"), manifest)
	if _, err := DocumentationEntry(program); err == nil {
		t.Fatal("accepted a manifest outside the installed help root")
	}
}

func TestDocumentationReferencesExcludePrivateOriginsAndExecutableSchemes(t *testing.T) {
	for _, value := range []string{"file:///private.html", "javascript:alert(1)", "http://wails.localhost/assets/help/index.html", "https://name:secret@example.org/", "//example.org/", "https://example.org/\nargument"} {
		if ValidDocumentationReference(value) {
			t.Fatal("accepted invalid external reference", value)
		}
	}
	if !ValidDocumentationReference("https://learn.microsoft.com/en-us/windows/?a=one&b=two#topic") {
		t.Fatal("rejected explicit web reference")
	}
}
