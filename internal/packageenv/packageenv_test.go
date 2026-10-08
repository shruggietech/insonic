// SPDX-License-Identifier: Apache-2.0
package packageenv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func fixture(t *testing.T, directory string) Manifest {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(directory, "companions"), 0o700); err != nil {
		t.Fatal(err)
	}
	content := []byte("controlled companion bytes")
	file := "companions/tool"
	if err := os.WriteFile(filepath.Join(directory, filepath.FromSlash(file)), content, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	tool := library.Tool{Path: file, SHA256: digest, Version: "fixture", SupportFiles: []library.PinnedFile{{Path: file, SHA256: digest}}}
	return Manifest{Kind: "installed-companions", Version: contracts.Version, Platform: runtime.GOOS + "_" + runtime.GOARCH,
		Media:  library.Tools{Kind: "media-tools", Version: contracts.Version, ExifTool: tool, FFprobe: tool, FFmpeg: tool},
		Cueson: subtitles.Tool{Executable: file, ExecutableSHA256: digest}, Perl: runtime.GOOS != "windows"}
}

func writeManifest(t *testing.T, directory string, manifest Manifest) {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ManifestName), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRelocatedCompanionsDoNotRetainOldAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "original installation")
	manifest := fixture(t, original)
	writeManifest(t, original, manifest)
	relocated := filepath.Join(root, "relocated installation with spaces")
	if err := os.Rename(original, relocated); err != nil {
		t.Fatal(err)
	}
	relocated, err := filepath.EvalSymlinks(relocated)
	if err != nil {
		t.Fatal(err)
	}
	tools, found, err := Read(relocated)
	if err != nil || !found {
		t.Fatalf("missing relocated defaults: %v", err)
	}
	for _, path := range []string{tools.Media.ExifTool.Path, tools.Media.FFmpeg.Path, tools.Media.FFprobe.Path, tools.Cueson.Executable, tools.Media.ExifTool.SupportFiles[0].Path} {
		if !strings.HasPrefix(path, relocated+string(filepath.Separator)) || strings.Contains(path, original) {
			t.Fatalf("tool retained wrong installation: %s", path)
		}
	}
	if runtime.GOOS != "windows" && tools.Media.ExifTool.Interpreter == nil {
		t.Fatal("missing explicit native Perl prerequisite")
	}
	if _, err := os.Stat(filepath.Join(relocated, ".insonic")); !os.IsNotExist(err) {
		t.Fatal("discovery persisted workspace state")
	}
}

func TestCompanionTamperAndEscapeNeverBecomeDefaults(t *testing.T) {
	for _, change := range []string{"bytes", "support", "escape", "absolute", "separator", "platform"} {
		t.Run(change, func(t *testing.T) {
			directory := t.TempDir()
			manifest := fixture(t, directory)
			switch change {
			case "bytes":
				os.WriteFile(filepath.Join(directory, "companions/tool"), []byte("changed"), 0o700)
			case "support":
				manifest.Media.ExifTool.SupportFiles[0].SHA256 = strings.Repeat("0", 64)
			case "escape":
				manifest.Cueson.Executable = "../tool"
			case "absolute":
				manifest.Cueson.Executable = filepath.Join(directory, "companions/tool")
			case "separator":
				manifest.Cueson.Executable = "companions\\tool"
			case "platform":
				manifest.Platform = "unqualified_arm64"
			}
			writeManifest(t, directory, manifest)
			if _, found, err := Read(directory); !found || err == nil {
				t.Fatalf("invalid installed manifest silently accepted: found=%v error=%v", found, err)
			}
		})
	}
}

func TestAbsentManifestMeansSourceBuildButInvalidManifestFails(t *testing.T) {
	directory := t.TempDir()
	if _, found, err := Read(directory); found || err != nil {
		t.Fatalf("source build discovery: %v %v", found, err)
	}
	if err := os.WriteFile(filepath.Join(directory, ManifestName), []byte(`{"kind":"installed-companions","unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := Read(directory); !found || err == nil {
		t.Fatal("invalid installed metadata fell back to source build")
	}
}

func TestCompanionSymlinkCannotEscapeInstallation(t *testing.T) {
	directory := t.TempDir()
	manifest := fixture(t, directory)
	external := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(external, []byte("controlled companion bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "companions/link")
	if err := os.Symlink(external, link); err != nil {
		t.Skip("host lacks symlink creation permission")
	}
	manifest.Cueson.Executable = "companions/link"
	writeManifest(t, directory, manifest)
	if _, found, err := Read(directory); !found || err == nil {
		t.Fatal("outside symlink accepted")
	}
}
