// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"strings"
	"testing"
	"time"
)

func TestSpeakerDiscoveryAuthority(t *testing.T) {
	speakerDiscoverySuite(t, localStore(t, contracts.ID()))
}
func speakerDiscoverySuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, r, voice := evidenceRecording(t, s)
	frozen, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil {
		t.Fatal(e)
	}
	if frozen.Context.Cues[0].Text != "Research source." {
		t.Fatal("missing current cue")
	}
	before, _ := s.SpeakerMappings(ctx, r.ID)
	payload, _ := json.Marshal(map[string]any{"recording_id": r.ID, "source_epoch": frozen.SourceEpoch, "max_cues": 100})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "speakers.discover", payload)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	afterWork, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil || afterWork.SourceEpoch != frozen.SourceEpoch {
		t.Fatal("work invalidated source", e)
	}
	raw := []byte(`{"version":1,"observations":[{"class":"mention","name":"Research","voice_id":"` + voice + `","references":[{"cue_id":"cue-000000","quote":"Research source."}],"uncertain":true}]}`)
	h, e := sd.Parse(raw, frozen.Context, 128)
	if e != nil {
		t.Fatal(e)
	}
	p, _ := sd.NormalizePolicy(sd.Policy{})
	d, _ := sd.Resolve(frozen.Context, h, p)
	a := sd.Aggregate{Version: 1, Coverage: sd.Coverage{Total: 1, Selected: 1, Calls: 1}, Hypotheses: h, Acoustic: frozen.Context.Acoustic, Policy: p, Decisions: d}
	accepted, e := s.AcceptSpeakerDiscovery(ctx, w, frozen, a)
	if e != nil || accepted.State != "succeeded" {
		t.Fatal(accepted, e)
	}
	current, e := s.SpeakerDiscovery(ctx, r.ID)
	if e != nil || current.Revision < 1 {
		t.Fatal(current, e)
	}
	if _, e = s.AcceptSpeakerDiscovery(ctx, w, frozen, a); e != nil {
		t.Fatal("lost acceptance replay", e)
	}
	if _, e = s.ResolveSpeakerDiscovery(ctx, contracts.ID(), r.ID, current.Revision, p); e != nil {
		t.Fatal(e)
	}
	after, _ := s.SpeakerMappings(ctx, r.ID)
	if sd.Digest(before) != sd.Digest(after) {
		t.Fatal("discovery mutated mappings")
	}
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Alex"}})
	if e != nil {
		t.Fatal(e)
	}
	_ = sp
	if _, e = s.SpeakerDiscovery(ctx, r.ID); e == nil {
		t.Fatal("obsolete identities remained visible")
	}
	if _, e = s.AcceptSpeakerDiscovery(ctx, w, frozen, a); e != nil {
		t.Fatal("terminal reconciliation", e)
	}
	if _, e = s.SpeakerDiscovery(ctx, r.ID); e == nil {
		t.Fatal("historical replay resurrected names")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_ = snap
}
func TestHistoricalV11Frozen(t *testing.T) {
	if historicalV11Digest() != historicalV11ExpectedDigest {
		t.Fatal("historical DDL changed")
	}
}

func TestDiscoveryPortableProofAndMigration(t *testing.T) {
	discoveryPortableSuite(t, localStore(t, contracts.ID()))
}
func discoveryPortableSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, r, _ := evidenceRecording(t, s)
	before, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.ExecContext(ctx, "DROP TABLE current_speaker_discovery"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.ExecContext(ctx, s.query("UPDATE catalog_schema SET version=11,digest=?"), historicalV11Digest()); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.ExecContext(ctx, "UPDATE workspace SET schema_version=11"); e != nil {
		t.Fatal(e)
	}
	if e = s.migrate(ctx); e != nil {
		t.Fatal(e)
	}
	after, e := s.Export(ctx)
	if e != nil || sd.Digest(before) != sd.Digest(after) {
		t.Fatal("v11 upgrade changed accepted state", e)
	}
	before.CatalogSchema = 11
	before.Digest, _ = before.digest()
	if e = localStore(t, s.workspace).Restore(ctx, before); e != nil {
		t.Fatal("historical snapshot", e)
	}
	frozen, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil {
		t.Fatal(e)
	}
	payload, _ := json.Marshal(map[string]any{"recording_id": r.ID, "source_epoch": frozen.SourceEpoch, "max_cues": 100})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "speakers.discover", payload)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	p, _ := sd.NormalizePolicy(sd.Policy{})
	d, _ := sd.Resolve(frozen.Context, nil, p)
	a := sd.Aggregate{Version: 1, Coverage: frozen.Coverage, Hypotheses: []sd.Hypothesis{}, Acoustic: frozen.Context.Acoustic, Policy: p, Decisions: d}
	if _, e = s.AcceptSpeakerDiscovery(ctx, w, frozen, a); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("valid discovery restore", e)
	}
	mutate := func(f func(*Snapshot)) {
		raw, _ := json.Marshal(snap)
		var candidate Snapshot
		json.Unmarshal(raw, &candidate)
		f(&candidate)
		candidate.Digest, _ = candidate.digest()
		if e := localStore(t, s.workspace).Restore(ctx, candidate); e == nil {
			t.Fatal("forged snapshot accepted")
		}
	}
	mutate(func(v *Snapshot) { v.CatalogSchema = 11 })
	mutate(func(v *Snapshot) { v.Records.Discoveries[0].SourceEpoch = hash([]byte("forged")) })
	mutate(func(v *Snapshot) { v.Records.Discoveries = nil })
	mutate(func(v *Snapshot) {
		var a sd.Aggregate
		json.Unmarshal(v.Records.Discoveries[0].Aggregate, &a)
		a.Decisions[0].Name = "injected"
		v.Records.Discoveries[0].Aggregate, _ = json.Marshal(a)
	})
	current, e := s.SpeakerDiscovery(ctx, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ResolveSpeakerDiscovery(ctx, contracts.ID(), r.ID, current.Revision+1, p); e == nil {
		t.Fatal("concurrent revision bypass")
	}
}

func TestDiscoveryAcceptanceFencesAndInvalidation(t *testing.T) {
	for _, mode := range []string{"foreign", "generation", "cancel", "selection", "max-cues", "revision", "source", "alias", "roster", "owner"} {
		t.Run(mode, func(t *testing.T) { discoveryMutationCase(t, localStore(t, contracts.ID()), mode) })
	}
}
func discoveryMutationCase(t *testing.T, s *Store, mode string) {
	ctx := context.Background()
	_, r, voice := evidenceRecording(t, s)
	identity, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Research"}})
	if e != nil {
		t.Fatal(e)
	}
	f, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil {
		t.Fatal(e)
	}
	payload, _ := json.Marshal(map[string]any{"recording_id": r.ID, "source_epoch": f.SourceEpoch, "max_cues": 100, "expected_revision": f.ExpectedRevision})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "speakers.discover", payload)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	p, _ := sd.NormalizePolicy(sd.Policy{})
	d, _ := sd.Resolve(f.Context, nil, p)
	a := sd.Aggregate{OwnerEvidence: f.Context.OwnerEvidence, Version: 1, Coverage: f.Coverage, Hypotheses: []sd.Hypothesis{}, Acoustic: f.Context.Acoustic, Policy: p, Decisions: d}
	switch mode {
	case "foreign":
		f.RecordingID = contracts.ID()
	case "generation":
		w.Generation++
	case "cancel":
		_, e = s.CancelWork(ctx, contracts.ID(), w.ID)
	case "selection":
		f.MatchingWorkID = contracts.ID()
	case "max-cues":
		f.MaxCues = -1
	case "revision":
		f.ExpectedRevision++
	default:
		if _, e = s.AcceptSpeakerDiscovery(ctx, w, f, a); e != nil {
			t.Fatal(e)
		}
		switch mode {
		case "source":
			replacement, err := s.EnqueueWork(ctx, contracts.ID(), "recordings.assemble", json.RawMessage("{}"))
			if err != nil {
				t.Fatal(err)
			}
			replacement, err = s.ClaimWork(ctx, replacement.ID, contracts.ID(), time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			_, e = s.CommitRecording(ctx, replacement, r.Revision, r)
		case "alias":
			identity.Aliases = []SpeakerAlias{{ID: contracts.ID(), SpeakerID: identity.Speaker.ID, Text: "Alternate"}}
			_, e = s.PutSpeaker(ctx, contracts.ID(), identity.Speaker.Revision, identity)
		case "roster":
			_, e = s.MutateRoster(ctx, contracts.ID(), r.ID, 0, "replace", []string{identity.Speaker.ID})
		case "owner":
			_, e = s.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: voice, SpeakerID: identity.Speaker.ID, DocumentDigest: r.DocumentDigest})
		}
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.SpeakerDiscovery(ctx, r.ID); e == nil {
			t.Fatal("obsolete evidence survived", mode)
		}
		if _, e = s.AcceptSpeakerDiscovery(ctx, w, f, a); e != nil {
			t.Fatal(e)
		}
		if _, e = s.SpeakerDiscovery(ctx, r.ID); e == nil {
			t.Fatal("replay resurrected evidence")
		}
		snap, err := s.Export(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = localStore(t, s.workspace).Restore(ctx, snap); err != nil {
			t.Fatal("tombstone restore", err)
		}
		return
	}
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AcceptSpeakerDiscovery(ctx, w, f, a); e == nil {
		t.Fatal("publication fence bypass", mode)
	}
	if _, e = s.SpeakerDiscovery(ctx, r.ID); e == nil {
		t.Fatal("failed work published")
	}
}

func TestDiscoveryPreservesInactiveManualOwner(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, r, voice := evidenceRecording(t, s)
	identity, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Inactive owner"}})
	if e != nil {
		t.Fatal(e)
	}
	mapping, e := s.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: voice, SpeakerID: identity.Speaker.ID, DocumentDigest: r.DocumentDigest})
	if e != nil {
		t.Fatal(e)
	}
	identity.Speaker.State = "inactive"
	if _, e = s.PutSpeaker(ctx, contracts.ID(), identity.Speaker.Revision, identity); e != nil {
		t.Fatal(e)
	}
	f, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil {
		t.Fatal(e)
	}
	sent, _ := json.Marshal(f.Context)
	if strings.Contains(string(sent), "Inactive owner") {
		t.Fatal("inactive owner joined provider candidates")
	}
	d, e := sd.Resolve(f.Context, nil, sd.Policy{})
	if e != nil || d[0].Outcome != "matched" || d[0].SpeakerID == nil || *d[0].SpeakerID != mapping.SpeakerID {
		t.Fatal(d, e)
	}
	if len(f.Context.OwnerEvidence) != 1 || f.Context.OwnerEvidence[0].MappingID != mapping.ID {
		t.Fatal("missing current owner reference")
	}
	payload, _ := json.Marshal(map[string]any{"recording_id": r.ID, "source_epoch": f.SourceEpoch, "max_cues": 100})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "speakers.discover", payload)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	p, _ := sd.NormalizePolicy(sd.Policy{})
	if _, e = s.AcceptSpeakerDiscovery(ctx, w, f, sd.Aggregate{Version: 1, Coverage: f.Coverage, OwnerEvidence: f.Context.OwnerEvidence, Hypotheses: []sd.Hypothesis{}, Acoustic: f.Context.Acoustic, Policy: p, Decisions: d}); e != nil {
		t.Fatal(e)
	}
}
