// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/launcher"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HelpAssetPolicy keeps bundled help independent of network availability. It
// restricts resources; the frame navigation handler routes explicit web links
// through the native system-browser dispatcher.
func HelpAssetPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/help/") {
			w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self'; script-src 'self'; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'")
		}
		next.ServeHTTP(w, r)
	})
}

// OpenHelp validates the installed build's embedded documentation, without a workspace.
func (b *Bridge) OpenHelp() contracts.Response {
	b.mu.RLock()
	files := b.helpAssets
	b.mu.RUnlock()
	if files == nil {
		var err error
		files, err = fs.Sub(Assets, "assets/help")
		if err != nil {
			return helpFailure()
		}
	}
	raw, err := fs.ReadFile(files, "offline-manifest.json")
	if err != nil {
		return helpFailure()
	}
	entry, err := launcher.DocumentationManifest(raw)
	if err != nil {
		return helpFailure()
	}
	info, err := fs.Stat(files, entry)
	if err != nil || !info.Mode().IsRegular() {
		return helpFailure()
	}
	target := (&url.URL{Path: "/assets/help/" + entry}).String()
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: map[string]any{"opened": true, "url": target, "version": contracts.Version}}
}
func helpFailure() contracts.Response {
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Condition("unavailable", "documentation_open")}
}

func (b *Bridge) OpenHelpReference(target string) contracts.Response {
	if !launcher.ValidDocumentationReference(target) {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("invalid_request")}
	}
	b.mu.Lock()
	opener := b.referenceOpen
	if b.qualification != nil {
		// The explicit native fixture observes dispatch without opening an
		// internet site or changing the runner's web association.
		b.qualification["help_reference"] = target
		opener = func(string) error { return nil }
	}
	b.mu.Unlock()
	if opener == nil {
		executable, err := os.Executable()
		if err != nil {
			return helpFailure()
		}
		opener = func(value string) error { return launcher.OpenDocumentationReference(filepath.Dir(executable), value) }
	}
	if opener(target) != nil {
		return helpFailure()
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: map[string]any{"opened": true}}
}

// ConfigureHelpQualification elects only the explicit no-workspace native fixture.
func ConfigureHelpQualification(b *Bridge) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.qualification = map[string]any{"help_only": true}
	b.qualificationProgress = newQualificationProgress("help", os.Stderr, time.Now)
}
