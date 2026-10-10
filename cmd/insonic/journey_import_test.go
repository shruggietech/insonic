// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
)

func TestJourneyNanosecondOriginImport(t *testing.T) {
	op, item, data, err := parseDomain([]string{"media", "import", filepath.Join(t.TempDir(), "speech.flac"), "--originated-at", "2020-01-01T00:00:00.123456789Z"})
	if err != nil {
		t.Fatal(err)
	}
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, ItemID: item, Data: data}
	raw, _ := json.Marshal(request)
	if err = schemas.ValidateRequest(raw); err != nil {
		t.Fatal(err)
	}
}
