package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func identitySuite(t *testing.T, s *Store) {
	ctx := context.Background()
	id, alias := contracts.ID(), contracts.ID()
	identity := SpeakerIdentity{Speaker: Speaker{ID: id, Name: "Jane", State: "active"}, Aliases: []SpeakerAlias{{ID: alias, SpeakerID: id, Text: "Janie", Language: "en", State: "active"}}}
	got, e := s.PutSpeaker(ctx, contracts.ID(), 0, identity)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutSpeaker(ctx, contracts.ID(), 0, identity); e == nil {
		t.Fatal("stale create")
	}
	termID := contracts.ID()
	term, e := s.PutTerm(ctx, contracts.ID(), 0, Term{ID: termID, Canonical: "spectrogram", Variants: json.RawMessage(`["spectrograms"]`), Language: "en", SpeakerID: &id, AliasID: &alias})
	if e != nil {
		t.Fatal(e)
	}
	snap, e := s.ContextInputs(ctx, ContextFilter{Language: "en", SpeakerIDs: []string{id}})
	if e != nil || len(snap.Speakers) != 1 || len(snap.Terms) != 1 {
		t.Fatal(snap, e)
	}
	// Aliases referenced by terms cannot be silently deleted.
	got.Aliases = nil
	if _, e = s.PutSpeaker(ctx, contracts.ID(), got.Speaker.Revision, got); e == nil {
		t.Fatal("dangling alias accepted")
	}
	changed := term
	changed.Canonical = "spectral"
	if _, e = s.PutTerm(ctx, contracts.ID(), term.Revision, changed); e != nil {
		t.Fatal(e)
	}
	exported, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	old := exported.Records.Terms[0]
	exported.Records.Terms[0] = term
	exported.Digest, _ = exported.digest()
	other := localStore(t, s.workspace)
	if e = other.Restore(ctx, exported); e == nil {
		t.Fatal("rolled-back term accepted")
	}
	exported.Records.Terms[0] = old
	exported.Records.Speakers = nil
	exported.Records.Aliases = nil
	exported.Records.Terms = nil
	exported.Digest, _ = exported.digest()
	if e = localStore(t, s.workspace).Restore(ctx, exported); e == nil {
		t.Fatal("omitted speaker domain accepted")
	}
	config, _ := json.Marshal(map[string]any{"recognition": map[string]any{"adapter": "faster-whisper", "contract_version": "1", "mode": "local", "model_id": contracts.ID()}, "diarization": map[string]any{"adapter": "pyannote", "contract_version": "1", "mode": "local", "model_id": contracts.ID()}})
	pipeline, e := s.PutPipeline(ctx, contracts.ID(), 0, Pipeline{ID: contracts.ID(), Name: "Local", Preset: "local", Configuration: config})
	if e != nil {
		t.Fatal(e)
	}
	pipeline.Configuration = json.RawMessage(`{"api_key":"secret"}`)
	if _, e = s.PutPipeline(ctx, contracts.ID(), pipeline.Revision, pipeline); e == nil {
		t.Fatal("secret persisted")
	}
	valid, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, valid); e != nil {
		t.Fatal("current restore", e)
	}
}
func TestSQLiteIdentityDomains(t *testing.T) { identitySuite(t, localStore(t, contracts.ID())) }
