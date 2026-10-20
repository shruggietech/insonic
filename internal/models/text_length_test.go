// SPDX-License-Identifier: Apache-2.0
package models

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestManifestUnicodeCharacterLimits(t *testing.T) {
	for _, alphabet := range []string{"a", "é", "漢", "😀", "e\u0301"} {
		unit := len([]rune(alphabet))
		m := Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: strings.Repeat(alphabet, 256/unit), ModelVersion: strings.Repeat(alphabet, 256/unit), Revision: strings.Repeat(alphabet, 512/unit), License: "fixture", Capabilities: []string{strings.Repeat(alphabet, 128/unit)}, Files: []File{{Role: strings.Repeat(alphabet, 128/unit), SHA256: strings.Repeat("a", 64), URL: "https://example.org/model", Size: 0}}}
		raw, _ := json.Marshal(m)
		var got Manifest
		if err := DecodeManifest(raw, &got); err != nil {
			t.Errorf("%q at scalar limit: %v", alphabet, err)
			continue
		}
		if got.Digest() != m.Digest() || InstallationID(got) != InstallationID(m) {
			t.Fatal("identity bytes changed")
		}
		got.Name += "a"
		if got.Validate() == nil {
			t.Fatal("257 characters accepted")
		}
	}
	m := Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: string([]byte{0xff}), ModelVersion: "v", Revision: "r", License: "fixture", Capabilities: []string{"x"}, Files: []File{{Role: "x", SHA256: strings.Repeat("a", 64), URL: "https://example.org/model", Size: 0}}}
	if m.Validate() == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	if !modelText(strings.Repeat("漢", 128), 128) || modelText(strings.Repeat("漢", 129), 128) {
		t.Fatal("compatibility text counts bytes")
	}
}
