// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
)

func TestImportPresetLifecycle(t *testing.T) { importPresetSuite(t, localStore) }
func importPresetSuite(t *testing.T, create func(*testing.T, string) *Store) {
	ctx := context.Background()
	s := create(t, contracts.ID())
	p := ImportPreset{Kind: "import-preset", Version: contracts.Version, ID: contracts.ID(), Name: "Daily recordings", Options: json.RawMessage(`{"timezone":"UTC","date_precedence":"owner-first"}`)}
	profile := workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "filesystem", Version: contracts.Version, Configuration: map[string]any{"root": t.TempDir()}}
	p.Options, _ = json.Marshal(map[string]any{"timezone": "UTC", "date_precedence": "owner-first", "destination": profile})
	op := contracts.ID()
	first, e := s.MutateImportPreset(ctx, op, "create", 0, p)
	if e != nil {
		t.Fatal(e)
	}
	p.Name = "Edited"
	p.Revision = first.Revision
	next, e := s.MutateImportPreset(ctx, contracts.ID(), "update", first.Revision, p)
	if e != nil || next.Revision <= first.Revision {
		t.Fatal(next, e)
	}
	original := first
	original.Revision = 0
	original.State = ""
	replay, e := s.MutateImportPreset(ctx, op, "create", 0, original)
	if e != nil || replay.Name != first.Name || replay.Revision != first.Revision {
		t.Fatal(replay, e)
	}
	if _, e = s.MutateImportPreset(ctx, contracts.ID(), "update", first.Revision, p); e == nil {
		t.Fatal("stale update accepted")
	}
	duplicate := original
	duplicate.ID = contracts.ID()
	duplicate.Name = next.Name
	if _, e = s.MutateImportPreset(ctx, contracts.ID(), "create", 0, duplicate); e == nil {
		t.Fatal("duplicate name accepted")
	}
	deleted, e := s.MutateImportPreset(ctx, contracts.ID(), "delete", next.Revision, ImportPreset{ID: p.ID})
	if e != nil || deleted.State != "deleted" {
		t.Fatal(deleted, e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest := create(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	tampered := snap
	tampered.Records.Profiles = append([]Profile(nil), snap.Records.Profiles...)
	tampered.Records.Profiles[0].Configuration = json.RawMessage(`{"root":"changed"}`)
	tampered.Digest, _ = tampered.digest()
	invalid := create(t, s.workspace)
	if invalid.Restore(ctx, tampered) == nil {
		t.Fatal("preset profile revision was silently rebound during restore")
	}
	all, e := dest.ImportPresets(ctx)
	if e != nil || len(all) != 1 || all[0].State != "deleted" {
		t.Fatal(all, e)
	}
	if _, e = s.PutNamedSetting(ctx, contracts.ID(), "import-presets", 0, json.RawMessage(`{}`)); e == nil {
		t.Fatal("reserved registry bypass")
	}
	other := create(t, contracts.ID())
	all, e = other.ImportPresets(ctx)
	if e != nil || len(all) != 0 {
		t.Fatal("workspace leak", all, e)
	}
	secret := original
	secret.ID = contracts.ID()
	secret.Options = json.RawMessage(`{"password":"private"}`)
	if _, e = s.MutateImportPreset(ctx, contracts.ID(), "create", 0, secret); e == nil {
		t.Fatal("secret accepted")
	}
}

func TestImportPresetIncompatibleRegistry(t *testing.T) {
	for _, raw := range []string{
		`{"kind":"import-preset-registry","schema_version":"99.0.0","items":[]}`,
		`{"kind":"import-preset-registry","schema_version":"1.1.0","items":[{"kind":"import-preset","schema_version":"99.0.0","id":"ed34901a-ed3c-4a60-aef4-aed07a0759f4","revision":1,"name":"Future","state":"active","options":{}}]}`,
	} {
		_, err := decodeImportPresets([]byte(raw), 1)
		if err == nil || contracts.SafeError(err).Code != "incompatible_version" {
			t.Fatalf("incompatible registry: %v", err)
		}
	}
}

func TestPreviousReleaseImportPresetsKeepTheirAuthority(t *testing.T) {
	preset := ImportPreset{Kind: "import-preset", Version: "1.0.1", ID: contracts.ID(), Revision: 1, Name: "Existing", State: "active", Options: json.RawMessage(`{"timezone":"UTC"}`)}
	raw, _ := json.Marshal(importPresetRegistry{Kind: "import-preset-registry", Version: "1.0.1", Items: []ImportPreset{preset}})
	registry, err := decodeImportPresets(raw, 1)
	if err != nil || len(registry.Items) != 1 || registry.Items[0].ID != preset.ID || registry.Items[0].Version != "1.0.1" {
		t.Fatalf("existing preset rejected: %+v %v", registry, err)
	}
	preset.Options = json.RawMessage(`{"password":"fixture-secret"}`)
	if validateImportPreset(preset) == nil {
		t.Fatal("old version bypassed option validation")
	}
}
