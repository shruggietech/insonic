// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"testing"
)

func TestPresetDestinationPublicationRemainsReadable(t *testing.T) {
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "routes")
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	profile := workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "filesystem", Version: contracts.Version, Configuration: map[string]any{"root": filepath.Join(t.TempDir(), "alternate")}}
	options, _ := json.Marshal(map[string]any{"destination": profile})
	check, _ := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "media.import", Data: json.RawMessage(append(append([]byte(`{"kind":"import-manifest","schema_version":"1.0.2","defaults":`), options...), []byte(`,"items":[{"source":"validation"}]}`)...))})
	if e = schemas.ValidateRequest(check); e != nil {
		t.Fatal(e)
	}
	p, e := db.MutateImportPreset(ctx, contracts.ID(), "create", 0, catalog.ImportPreset{Kind: "import-preset", Version: contracts.Version, ID: contracts.ID(), Name: "Alternate", Options: options})
	if e != nil {
		t.Fatal(e)
	}
	s, e := NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	s.Grace = 0
	source := bytesFile(t, []byte("destination fixture"))
	published, e := s.PublishTo(ctx, profile, contracts.ID(), source.Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	if _, e = os.Stat(filepath.Join(profile.Configuration["root"].(string), filepath.FromSlash(published.Key))); e != nil {
		t.Fatal("alternate destination not used", e)
	}
	if _, e = db.MutateImportPreset(ctx, contracts.ID(), "delete", p.Revision, catalog.ImportPreset{ID: p.ID}); e != nil {
		t.Fatal(e)
	}
	s, e = NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	s.Grace = 0
	m, e := s.Materialize(ctx, published.ID)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(m.Path)
	if e != nil || string(raw) != "destination fixture" {
		t.Fatal(string(raw), e)
	}
	if e = s.Release(ctx, published.ID, m.Lease.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Retire(ctx, published.ID); e != nil {
		t.Fatal(e)
	}
	other, e := catalog.OpenSQLite(ctx, filepath.Join(t.TempDir(), "other.sqlite"), contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	if _, e = other.Profile(ctx, profile.ID, 1); e == nil {
		t.Fatal("profile crossed workspace")
	}
}
