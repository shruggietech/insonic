// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/packageenv"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func TestDesktopSettingsResetPreservesCASAndRemovesOverrides(t *testing.T) {
	for _, section := range []string{"media_tools", "processing_tools", "appearance"} {
		t.Run(section, func(t *testing.T) {
			a := configuredApp(t)
			path := filepath.Join(a.Workspace.Control, settingFiles[section])
			// A reset must also repair malformed explicit overrides without
			// interpreting their value or copying package defaults into the file.
			raw := []byte(`{"invalid":"explicit override"}`)
			if e := os.WriteFile(path, raw, 0600); e != nil {
				t.Fatal(e)
			}
			out := realRequest(a, "settings.set", "", map[string]any{"section": section, "revision": strings.Repeat("0", 64), "value": nil})
			if out.Error == nil || out.Error.Code != "conflict" {
				t.Fatal("stale reset removed another override", out)
			}
			if got, e := os.ReadFile(path); e != nil || string(got) != string(raw) {
				t.Fatal("rejected reset modified configuration", string(got), e)
			}
			out = realRequest(a, "settings.set", "", map[string]any{"section": section, "revision": settingRevision(raw), "value": nil})
			if out.Error != nil {
				t.Fatal("reset refused current explicit revision", out.Error)
			}
			result := out.Result.(map[string]any)
			if result["revision"] != settingRevision(nil) || result["value"] != nil || result["origin"] == "workspace" {
				t.Fatal("reset did not return default authority", result)
			}
			if _, e := os.Stat(path); !os.IsNotExist(e) {
				t.Fatal("reset persisted a defaults placeholder", e)
			}
			out = realRequest(a, "settings.show", "", nil)
			if out.Error != nil {
				t.Fatal(out.Error)
			}
			shown := out.Result.(map[string]any)["sections"].(map[string]any)[section].(map[string]any)
			if shown["revision"] != settingRevision(nil) || shown["origin"] == "workspace" || shown["error"] != nil {
				t.Fatal("reset defaults are not readable", shown)
			}
			out = realRequest(a, "settings.set", "", map[string]any{"section": section, "revision": settingRevision(raw), "value": nil})
			if out.Error != nil {
				t.Fatal("completed reset replay is not idempotent", out.Error)
			}
		})
	}
}

func TestDesktopSettingsResetInstalledPackageDefaults(t *testing.T) {
	if os.Getenv("INSONIC_TEST_PACKAGE_SETTINGS_RESET") == "1" {
		a := configuredApp(t)
		for _, name := range []string{"media_tools", "processing_tools"} {
			read := func() map[string]any {
				out := realRequest(a, "settings.show", "", nil)
				if out.Error != nil {
					t.Fatal(out.Error)
				}
				return out.Result.(map[string]any)["sections"].(map[string]any)[name].(map[string]any)
			}
			initial := read()
			if initial["origin"] != "package" || initial["revision"] != settingRevision(nil) {
				t.Fatal("installed default authority missing", name, initial)
			}
			value, _ := json.Marshal(initial["value"])
			out := realRequest(a, "settings.set", "", map[string]any{"section": name, "revision": initial["revision"], "value": json.RawMessage(value)})
			if out.Error != nil {
				t.Fatal("explicit override", out.Error)
			}
			explicit := read()
			if explicit["origin"] != "workspace" || explicit["revision"] != settingRevision(value) {
				t.Fatal("override did not own its explicit content revision", explicit)
			}
			out = realRequest(a, "settings.set", "", map[string]any{"section": name, "revision": explicit["revision"], "value": nil})
			if out.Error != nil || out.Result.(map[string]any)["origin"] != "package" {
				t.Fatal("reset did not restore package authority", out)
			}
			current := read()
			currentValue, _ := json.Marshal(current["value"])
			if current["origin"] != "package" || current["revision"] != settingRevision(nil) || string(currentValue) != string(value) {
				t.Fatal("reset copied or changed installed defaults", current)
			}
			if _, e := os.Stat(filepath.Join(a.Workspace.Control, settingFiles[name])); !os.IsNotExist(e) {
				t.Fatal("installation paths remain persisted after reset", name, e)
			}
		}
		return
	}
	// A relocated test binary plus relative companion manifest exercises the
	// real executable-relative resolver without changing this process's defaults.
	dir := t.TempDir()
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	bytes, e := os.ReadFile(executable)
	if e != nil {
		t.Fatal(e)
	}
	copy := filepath.Join(dir, filepath.Base(executable))
	if e = os.WriteFile(copy, bytes, 0755); e != nil {
		t.Fatal(e)
	}
	pin := func(name string) library.Tool {
		content := []byte("installed companion fixture: " + name)
		if e := os.WriteFile(filepath.Join(dir, name), content, 0600); e != nil {
			t.Fatal(e)
		}
		digest := sha256.Sum256(content)
		return library.Tool{Path: name, SHA256: hex.EncodeToString(digest[:]), Version: "fixture"}
	}
	cueson := pin("cueson")
	manifest := packageenv.Manifest{Kind: "installed-companions", Version: contracts.Version, Platform: runtime.GOOS + "_" + runtime.GOARCH,
		Media:  library.Tools{Kind: "media-tools", Version: contracts.Version, ExifTool: pin("exiftool"), FFprobe: pin("ffprobe"), FFmpeg: pin("ffmpeg")},
		Cueson: subtitles.Tool{Executable: cueson.Path, ExecutableSHA256: cueson.SHA256}, Perl: runtime.GOOS != "windows"}
	raw, _ := json.Marshal(manifest)
	if e = os.WriteFile(filepath.Join(dir, packageenv.ManifestName), raw, 0600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, e := process.Capture(ctx, process.Spec{Executable: copy, Args: []string{"-test.run=^TestDesktopSettingsResetInstalledPackageDefaults$", "-test.v"},
		CleanEnv: true, Env: append(process.LocalEnvironment(), "INSONIC_TEST_PACKAGE_SETTINGS_RESET=1"), MaxOutput: 64 << 10})
	if e != nil {
		t.Fatal("installed reset qualification", e, string(result.Stdout), string(result.Stderr))
	}
}
