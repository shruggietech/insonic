// SPDX-License-Identifier: Apache-2.0
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func fixtureManifest() Manifest {
	sum := sha256.Sum256([]byte("model fixture"))
	return Manifest{Kind: "base-model-manifest", Version: "0.0.0", Name: "fixture", ModelVersion: "v1", Revision: "immutable-v1", Capabilities: []string{"transcription"}, License: "unknown", Files: []File{{Role: "weights", SHA256: hex.EncodeToString(sum[:]), Size: 13, URL: "https://example.org/fixture.bin"}}}
}
func TestManifestExactIntegrityAndNonsecretSources(t *testing.T) {
	good := fixtureManifest()
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	good.Files[0].URL = "https://example.org/fixture.bin?download=true&id=model"
	if err := good.Validate(); err != nil {
		t.Fatal("ordinary source selector rejected")
	}
	for _, url := range []string{"https://user:password@example.org/model", "https://example.org/model?token=secret", "http://example.org/model", "file:///private/model", "https://example.org/model#secret"} {
		bad := fixtureManifest()
		bad.Files[0].URL = url
		if bad.Validate() == nil {
			t.Fatalf("unapproved source accepted: %s", url)
		}
	}
	bad := fixtureManifest()
	bad.Files = append(bad.Files, bad.Files[0])
	if bad.Validate() == nil {
		t.Fatal("duplicate role admitted")
	}
	bad = fixtureManifest()
	bad.Files[0].SHA256 = "not-a-digest"
	if bad.Validate() == nil {
		t.Fatal("invalid digest admitted")
	}
	bad = fixtureManifest()
	bad.Files[0].Size = -1
	if bad.Validate() == nil {
		t.Fatal("negative size admitted")
	}
	raw, _ := json.Marshal(good)
	var decoded Manifest
	if err := DecodeManifest(append(raw, []byte(` {}`)...), &decoded); err == nil {
		t.Fatal("extra JSON admitted")
	}
	raw = []byte(`{"kind":"base-model-manifest","kind":"base-model-manifest"}`)
	if DecodeManifest(raw, &decoded) == nil {
		t.Fatal("duplicate keys admitted")
	}
}
