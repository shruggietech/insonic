// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestModelReferenceAuthority(t *testing.T) { modelReferenceSuite(t, localStore(t, contracts.ID())) }
func modelReferenceSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	target, _ := json.Marshal(ModelTarget{Kind: "hosted", Operation: "transcription", Adapter: "insonic-http", ContractVersion: "1", Endpoint: "https://example.org/inference", RemoteModel: "pinned", UpstreamRevision: "r1"})
	value := ModelAlias{ID: ModelReferenceID(s.workspace, "alias", "quick"), Name: "quick", State: "active", Target: target}
	op := contracts.ID()
	got, e := s.PutModelAlias(ctx, op, 0, value)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := s.PutModelAlias(ctx, op, 0, value)
	if e != nil || replay.Revision != got.Revision {
		t.Fatal("replay", e)
	}
	if _, e = s.PutModelAlias(ctx, contracts.ID(), 0, value); e == nil {
		t.Fatal("stale write")
	}
	gone, e := s.RemoveModelAlias(ctx, contracts.ID(), got.Revision, got.ID)
	if e != nil || gone.State != "deleted" {
		t.Fatal("remove", e)
	}
	if _, e = s.PutModelAlias(ctx, contracts.ID(), 0, value); e == nil {
		t.Fatal("remove/recreate ABA")
	}
	restored, e := s.PutModelAlias(ctx, contracts.ID(), gone.Revision, value)
	if e != nil || restored.Revision <= gone.Revision {
		t.Fatal("recreate", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	other := localStore(t, s.workspace)
	if e = other.Restore(ctx, snap); e != nil {
		t.Fatal("restore", e)
	}
	match, e := other.ModelAlias(ctx, "quick")
	if e != nil || match.Revision != restored.Revision {
		t.Fatal("portable alias", e)
	}
	snap.Records.ModelAliases[0].Name = "forged"
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("forged alias proof accepted")
	}
}

func TestModelReferenceValidation(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	for _, name := range []string{"", "UPPER", "a/b", "a b", contracts.ID()} {
		_, e := s.PutModelSource(ctx, contracts.ID(), 0, ModelSource{ID: contracts.ID(), Name: name, State: "active", URL: "https://example.org/models"})
		if e == nil {
			t.Fatal("invalid name", name)
		}
	}
	source := ModelSource{ID: ModelReferenceID(s.workspace, "source", "custom"), Name: "custom", State: "active", URL: "https://example.org/models"}
	got, e := s.PutModelSource(ctx, contracts.ID(), 0, source)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.RemoveModelSource(ctx, contracts.ID(), got.Revision-1, got.ID); e == nil {
		t.Fatal("stale source remove")
	}
	foreign := *s
	foreign.workspace = contracts.ID()
	if e = foreign.ensureWorkspace(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e = foreign.ModelSource(ctx, got.ID); e == nil {
		t.Fatal("foreign source resolved")
	}
	target, _ := json.Marshal(ModelTarget{Kind: "base", ID: contracts.ID(), Operation: "transcription"})
	if _, e = s.PutModelAlias(ctx, contracts.ID(), 0, ModelAlias{Name: "missing", State: "active", Target: target}); e == nil {
		t.Fatal("missing exact target accepted")
	}
}
