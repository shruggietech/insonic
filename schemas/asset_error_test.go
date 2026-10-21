// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestAssetObservationUsesPortableRuntimeSchema(t *testing.T) {
	for _, name := range []string{"weights/model.bin", "../secret", "/owner/key", `C:\owner\key`, "a//b", "a/../b", "NUL.txt", "a/COM1", "a\nsecret", "a/é", strings.Repeat("a", 129)} {
		err := contracts.AssetFailure("asset_hash", "model_file", "weights/model.bin")
		err.Asset.Name = name
		raw, _ := json.Marshal(contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: err})
		got := ValidateResponse(raw)
		if (got == nil) != contracts.ValidAssetName(name) {
			t.Fatalf("runtime/schema disagreement for %q: %v", name, got)
		}
	}
}
