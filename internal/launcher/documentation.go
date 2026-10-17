// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"encoding/json"
	"errors"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var documentationCorrection = errors.New("Installed offline documentation could not be opened. Open help/index.html in the program directory with your HTML browser, or use installer Repair to restore the matching help files. The installation remains available.")

// DocumentationEntry resolves only the matching installed documentation tree.
func DocumentationEntry(directory string) (string, error) {
	if !filepath.IsAbs(directory) {
		return "", documentationCorrection
	}
	program, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", documentationCorrection
	}
	base, err := filepath.EvalSymlinks(filepath.Join(program, "help"))
	if err != nil || !containsPath(program, base) || base == program {
		return "", documentationCorrection
	}
	manifest, err := filepath.EvalSymlinks(filepath.Join(base, "offline-manifest.json"))
	if err != nil || !containsPath(base, manifest) || manifest == base {
		return "", documentationCorrection
	}
	f, err := os.Open(manifest)
	if err != nil {
		return "", documentationCorrection
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return "", documentationCorrection
	}
	name, err := DocumentationManifest(raw)
	if err != nil {
		return "", err
	}
	entry, err := filepath.EvalSymlinks(filepath.Join(base, filepath.FromSlash(name)))
	if err != nil || !containsPath(base, entry) || entry == base {
		return "", documentationCorrection
	}
	info, err := os.Stat(entry)
	if err != nil || !info.Mode().IsRegular() {
		return "", documentationCorrection
	}
	return entry, nil
}

func OpenDocumentation(directory string) error {
	return openDocumentation(directory, "")
}

// Online references are an explicit user election outside the offline frame.
// Private application origins and executable/non-web schemes never leave it.
func ValidDocumentationReference(value string) bool {
	if len(value) > 2048 || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	u, err := url.ParseRequestURI(value)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil && !strings.EqualFold(strings.TrimSuffix(u.Hostname(), "."), "wails.localhost") && !strings.EqualFold(strings.TrimSuffix(u.Hostname(), "."), "wails")
}

func OpenDocumentationReference(directory, target string) error {
	if !ValidDocumentationReference(target) {
		return documentationCorrection
	}
	return openDocumentation(directory, target)
}

// DocumentationManifest is shared by installed-file and embedded-view consumers.
func DocumentationManifest(raw []byte) (string, error) {
	var manifest struct {
		Version string `json:"version"`
		Entry   string `json:"entrypoint"`
		Public  bool   `json:"public_landing_included"`
	}
	if len(raw) > 65536 || contracts.ValidateJSON(raw) != nil || json.Unmarshal(raw, &manifest) != nil || manifest.Version != contracts.Version || manifest.Public {
		return "", documentationCorrection
	}
	if manifest.Entry == "" || strings.ContainsAny(manifest.Entry, "\\:\x00\r\n") || filepath.IsAbs(manifest.Entry) || filepath.Ext(manifest.Entry) != ".html" || filepath.ToSlash(filepath.Clean(manifest.Entry)) != manifest.Entry || strings.HasPrefix(manifest.Entry, "../") {
		return "", documentationCorrection
	}
	return manifest.Entry, nil
}
