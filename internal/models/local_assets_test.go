// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalModelFailuresKeepLogicalRoleAndStage(t *testing.T) {
	for _, mode := range []string{"valid", "missing", "size", "hash", "directory"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			m := fixtureManifest()
			name := filepath.Join(dir, m.Files[0].Role)
			data := []byte("model fixture")
			if mode == "size" {
				data = []byte("short")
			}
			if mode == "hash" {
				data = []byte("model fixturf")
			}
			if mode == "directory" {
				os.Mkdir(name, 0700)
			} else if mode != "missing" {
				os.WriteFile(name, data, 0600)
			}
			err := VerifyLocal(context.Background(), dir, m)
			if mode == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			want := map[string]string{"missing": "asset_missing", "size": "asset_size", "hash": "asset_hash", "directory": "asset_not_regular"}[mode]
			got := contracts.SafeError(err)
			if err == nil || got.Reason != want || got.Asset == nil || got.Asset.Role != "model_file" || got.Asset.Name != m.Files[0].Role {
				t.Fatal(mode, got)
			}
		})
	}
}
