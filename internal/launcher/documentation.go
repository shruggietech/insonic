// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"encoding/json"
	"errors"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
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
	base, err := filepath.EvalSymlinks(filepath.Join(directory, "help"))
	if err != nil {
		return "", documentationCorrection
	}
	f, err := os.Open(filepath.Join(base, "offline-manifest.json"))
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
	entry, err := DocumentationEntry(directory)
	if err != nil {
		return err
	}
	return openDocumentation(directory, entry)
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
