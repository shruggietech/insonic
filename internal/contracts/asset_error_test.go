// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetErrorSurvivesRedactionAndPersistence(t *testing.T) {
	for _, reason := range []string{"asset_missing", "asset_unreadable", "asset_not_regular", "asset_size", "asset_hash", "asset_tool_missing"} {
		err := AssetFailure(reason, "model_file", "weights/model.bin")
		var restored Error
		raw, _ := json.Marshal(err)
		if e := json.Unmarshal(raw, &restored); e != nil {
			t.Fatal(e)
		}
		got := SafeError(fmt.Errorf("private path: %w", &restored))
		if got.Asset == nil || got.Asset.Name != "weights/model.bin" || got.Reason != reason || !strings.Contains(got.Message, "weights/model.bin") {
			t.Fatalf("lost generated observation: %+v", got)
		}
		if strings.Contains(got.Message, "private") {
			t.Fatal(got)
		}
	}
}

func TestAssetErrorRejectsUntrustedMetadata(t *testing.T) {
	for _, name := range []string{"", "../secret", "/root/key", `C:\owner\key`, "a/../b", "a//b", "NUL.txt", "a/COM1", "line\nsecret", "a/é", strings.Repeat("a", 129)} {
		got := SafeError(&Error{Code: "checksum_mismatch", Reason: "asset_hash", Message: "secret", Asset: &AssetObservation{Role: "model_file", Name: name}})
		if got.Asset != nil || strings.Contains(got.Message, "secret") {
			t.Fatalf("accepted %q: %+v", name, got)
		}
	}
	for _, e := range []*Error{
		{Code: "conflict", Reason: "asset_hash", Asset: &AssetObservation{Role: "model_file", Name: "model.bin"}},
		{Code: "checksum_mismatch", Reason: "asset_hash", Asset: &AssetObservation{Role: "private", Name: "model.bin"}},
	} {
		if SafeError(e).Asset != nil {
			t.Fatalf("accepted mismatched observation: %+v", e)
		}
	}
}

func TestUnreadableAssetUsesGeneratedLogicalCorrection(t *testing.T) {
	f, e := os.Create(filepath.Join(t.TempDir(), "private-owner-path"))
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
	err := VerifyAssetFile(context.Background(), f, "model_file", "weights.bin", "", 0)
	got := SafeError(err)
	if got.Reason != "asset_unreadable" || got.Asset == nil || strings.Contains(got.Message, "private-owner-path") {
		t.Fatal(got)
	}
}
