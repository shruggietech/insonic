// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"testing"
)

func TestOfflineAssetsInspectOnlyElectedFilesAndExplainFailures(t *testing.T) {
	dir := t.TempDir()
	p := Profile{Runtime: Asset{Name: "runtime.tgz", Size: 3, SHA256: Digest([]byte("abc"))}, Installer: Asset{Name: "uv-fixture.zip", Size: 2, SHA256: Digest([]byte("uv"))}, Wheels: []Asset{{Name: "fixture.whl", Size: 4, SHA256: Digest([]byte("abcd"))}}}
	os.WriteFile(filepath.Join(dir, p.Runtime.Name), []byte("abc"), 0600)
	os.WriteFile(filepath.Join(dir, p.Wheels[0].Name), []byte("abce"), 0600)
	os.WriteFile(filepath.Join(dir, "unrelated"), make([]byte, 100), 0600)
	got, e := InspectOffline(context.Background(), dir, p)
	if e != nil || got.RequiredBytes != 9 || got.VerifiedBytes != 3 || got.MissingBytes != 6 || len(got.Failures) != 2 {
		t.Fatal(got, e)
	}
	for i, want := range []string{"asset_missing", "asset_hash"} {
		if got.Failures[i].Reason != want || got.Failures[i].Asset == nil {
			t.Fatal(got.Failures[i])
		}
	}
	for _, mode := range []string{"size", "directory"} {
		name := filepath.Join(dir, p.Runtime.Name)
		os.Remove(name)
		if mode == "size" {
			os.WriteFile(name, []byte("a"), 0600)
		} else {
			os.Mkdir(name, 0700)
		}
		err := verifyAsset(context.Background(), name, p.Runtime)
		reason := "asset_size"
		if mode == "directory" {
			reason = "asset_not_regular"
		}
		if err == nil || contracts.SafeError(err).Reason != reason {
			t.Fatal(mode, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := InspectOffline(ctx, dir, p); e == nil || contracts.SafeError(e).Code != "cancelled" {
		t.Fatal(e)
	}
}

func TestMissingInstallerToolNamesLogicalPrerequisite(t *testing.T) {
	_, err := findExecutable(t.TempDir(), "uv")
	got := contracts.SafeError(err)
	if got.Reason != "asset_tool_missing" || got.Asset == nil || got.Asset.Role != "worker_installer" || got.Asset.Name != "uv" {
		t.Fatal(got)
	}
}
