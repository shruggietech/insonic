// SPDX-License-Identifier: Apache-2.0
package models

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestManifestInputReportsConstraintWithoutChangingIdentity(t *testing.T) {
	m := Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: " \n exact identity ", ModelVersion: "  ", Revision: "revision", License: "fixture", Capabilities: []string{"transcription"}, Files: []File{{Role: "weights", SHA256: strings.Repeat("a", 64), URL: "https://example.test/model", Size: 1}}}
	raw, _ := json.Marshal(m)
	var got Manifest
	if err := DecodeManifest(raw, &got); err != nil || got.Name != m.Name || got.ModelVersion != m.ModelVersion || got.Digest() != m.Digest() {
		t.Fatal(got, err)
	}
	m.Name = strings.Repeat("é", 257)
	raw, _ = json.Marshal(m)
	err := DecodeManifest(raw, &got)
	var classified *contracts.Error
	if !errors.As(err, &classified) || classified.Reason != "input_length" || classified.Validation == nil || classified.Validation.Pointer != "/name" || classified.Validation.Bound != 256 {
		t.Fatalf("%v", err)
	}
}
