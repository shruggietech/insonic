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
