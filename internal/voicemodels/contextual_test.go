// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"path/filepath"
	"strings"
	"testing"
)

func contextualDatasetFixture(t *testing.T, s *Service, db *catalog.Store, count int) (catalog.SpeakerDataset, catalog.Speaker) {
	t.Helper()
	ctx := context.Background()
	r, voice := recording(t, db, count)
	sp, e := db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Research"}})
	if e != nil {
		t.Fatalf("%v", e)
	}
	f, e := db.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil {
		t.Fatalf("%v", e)
	}
	raw, _ := json.Marshal(map[string]any{"version": 1, "observations": []any{map[string]any{"class": "self-identification", "name": "Research", "voice_id": voice, "references": []any{map[string]any{"cue_id": f.Context.Cues[0].ID, "quote": f.Context.Cues[0].Text}}, "uncertain": false}}})
	h, e := sd.Parse(raw, f.Context, 128)
	if e != nil {
		t.Fatalf("%v", e)
	}
	policy, _ := sd.NormalizePolicy(sd.Policy{})
	decisions, e := sd.Resolve(f.Context, h, policy)
	if e != nil {
		t.Fatalf("%v", e)
	}
	w := work(t, db, "speakers.discover", map[string]any{"recording_id": r.ID, "source_epoch": f.SourceEpoch, "max_cues": 100, "expected_revision": f.ExpectedRevision})
	_, e = db.AcceptSpeakerDiscovery(ctx, w, f, sd.Aggregate{Version: 1, Coverage: f.Coverage, Hypotheses: h, Acoustic: f.Context.Acoustic, Policy: policy, Decisions: decisions})
	if e != nil {
		t.Fatalf("%v", e)
	}
	d, e := db.SpeakerDiscovery(ctx, r.ID)
	if e != nil {
		t.Fatalf("%v", e)
	}
	if _, e = db.ApplySpeakerDiscovery(ctx, contracts.ID(), r.ID, d.Revision, nil); e != nil {
		t.Fatalf("%v", e)
	}
	dataset, e := s.CreateDataset(ctx, contracts.ID(), DatasetOptions{SpeakerID: sp.Speaker.ID})
	if e != nil {
		t.Fatalf("%v", e)
	}
	if len(dataset.References) != count || dataset.References[0].EvidenceClass != "independent-contextual" || dataset.References[0].EvidenceDigest == "" {
		t.Fatal(dataset)
	}
	return dataset, sp.Speaker
}
func TestContextualEnrollmentRetractionRetainsHistory(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	dataset, sp := contextualDatasetFixture(t, s, db, 3)
	claim := work(t, db, "models.train", TrainPayload{Options: TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Research", Kind: "voice-embedding", Adapter: builtin(), BaseModelID: contracts.ID(), BaseDigest: strings.Repeat("b", 64)}})
	result, e := s.ExecuteTrain(ctx, claim)
	if e != nil {
		t.Fatalf("%v", e)
	}
	version := result.(map[string]any)["version_id"].(string)
	if _, e = db.SetSpeakerProfile(ctx, contracts.ID(), sp.ID, 0, version); e != nil {
		t.Fatalf("%v", e)
	}
	r, e := db.Recording(ctx, dataset.References[0].RecordingID)
	if e != nil {
		t.Fatalf("%v", e)
	}
	ms, _ := db.SpeakerMappings(ctx, r.ID)
	if _, e = db.MutateRoster(ctx, contracts.ID(), r.ID, 0, "replace", []string{sp.ID}); e != nil {
		t.Fatalf("%v", e)
	}
	if _, e = db.RetractSpeakerDiscovery(ctx, contracts.ID(), r.ID, r.Revision, []catalog.MappingRevision{{ID: ms[0].ID, Revision: ms[0].Revision}}); e != nil {
		t.Fatalf("%v", e)
	}
	stale, e := db.SpeakerDataset(ctx, dataset.Dataset.ID)
	if e != nil || stale.Dataset.State != "invalidated" {
		t.Fatal(stale, e)
	}
	frozen, e := db.FreezeSpeakerMatching(ctx, r.ID)
	if e != nil || len(frozen.Profiles) != 0 {
		t.Fatal(frozen, e)
	}
	if _, e = s.Fetch(ctx, version, filepath.Join(t.TempDir(), "historical")); e != nil {
		t.Fatalf("%v", e)
	}
	if _, e = db.Export(ctx); e != nil {
		t.Fatalf("%v", e)
	}
}
