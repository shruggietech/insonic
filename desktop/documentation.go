// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/launcher"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// HelpAssetPolicy keeps bundled help independent of network availability. It
// applies to both navigation and resources inside the controlled native frame.
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

// ConfigureHelpQualification elects only the explicit no-workspace native fixture.
func ConfigureHelpQualification(b *Bridge) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.qualification = map[string]any{"help_only": true}
}
