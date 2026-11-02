// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"testing"
	"time"
)

func TestDiscoveryAcousticRequiresAcceptedProof(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, r, _ := evidenceRecording(t, s)
	absent, e := s.SpeakerDiscoveryAcoustic(ctx, r.ID, "")
	if e != nil || absent.Availability != "unavailable" {
		t.Fatal(absent, e)
	}
	f, e := s.FreezeSpeakerMatching(ctx, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(map[string]any{"options": map[string]any{"recording_id": r.ID}, "snapshot": f})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.match", b)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	result := json.RawMessage(`{"recording_id":"` + r.ID + `","state":"accepted","decisions":[]}`)
	if _, e = s.CheckpointWork(ctx, w, "complete", "succeeded", result, time.Minute); e != nil {
		t.Fatal(e)
	}
	value, e := s.SpeakerDiscoveryAcoustic(ctx, r.ID, w.ID)
	if e != nil || value.Availability != "stale" || len(value.Decisions) != 0 {
		t.Fatal("unproven predictions trusted", value, e)
	}
	foreign := localStore(t, contracts.ID())
	if _, e = foreign.SpeakerDiscoveryAcoustic(ctx, r.ID, w.ID); e == nil {
		t.Fatal("foreign recording accepted")
	}
}

func TestDiscoveryAcousticAuthenticatesMatchAndIndependentRoots(t *testing.T) {
	discoveryAcousticSuite(t, func(t *testing.T) *Store { return localStore(t, contracts.ID()) })
}
func discoveryAcousticSuite(t *testing.T, open func(*testing.T) *Store) {
	for _, sameSource := range []bool{false, true} {
		t.Run(map[bool]string{false: "independent", true: "overlap"}[sameSource], func(t *testing.T) {
			s := open(t)
			ctx := context.Background()
			ds, sp, source, _ := speakerDatasetFixture(t, s)
			output := speakerOutputFixture(t, s, ds)
			if _, e := s.SetSpeakerProfile(ctx, contracts.ID(), sp.ID, 0, output.ID); e != nil {
				t.Fatal(e)
			}
			claim, entry := entryFixture(t, s)
			if !sameSource {
				entry.Digest = hash([]byte("independent-target"))
			}
			entry, e := s.CommitLibrary(ctx, claim, entry)
			if e != nil {
				t.Fatal(e)
			}
			target := source
			target.ID = entry.ID
			target.Revision = 0
			target.SourceRevision = entry.Revision
			target.SourceDigest = entry.Digest
			target, e = s.CommitRecording(ctx, claim, 0, target)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.MutateRoster(ctx, contracts.ID(), target.ID, 0, "replace", []string{sp.ID}); e != nil {
				t.Fatal(e)
			}
			frozen, e := s.FreezeSpeakerMatching(ctx, target.ID)
			if e != nil {
				t.Fatal(e)
			}
			payload, _ := json.Marshal(map[string]any{"options": map[string]any{"recording_id": target.ID}, "snapshot": frozen})
			w, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.match", payload)
			if e != nil {
				t.Fatal(e)
			}
			w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			voices := documentSpeakers(target.Document)
			voice := ""
			for id := range voices {
				voice = id
			}
			score, runner := 0.95, -0.1
			decisions := []SpeakerMatchDecision{{LocalSpeakerID: voice, SpeakerID: sp.ID, State: "accepted", Score: &score, RunnerUpScore: &runner, CueIDs: []string{"cue-000000"}, Diagnostics: json.RawMessage("{}")}}
			if _, e = s.CommitSpeakerMatching(ctx, w, frozen, decisions, json.RawMessage("{}")); e != nil {
				t.Fatal(e)
			}
			result, _ := json.Marshal(map[string]any{"recording_id": target.ID, "state": "accepted", "decisions": []any{map[string]any{"local_speaker_id": voice, "speaker_id": sp.ID, "version_id": output.ID, "state": "accepted", "score": score, "runner_up_score": runner}}})
			if _, e = s.CheckpointWork(ctx, w, "complete", "succeeded", result, time.Minute); e != nil {
				t.Fatal(e)
			}
			acoustic, e := s.SpeakerDiscoveryAcoustic(ctx, target.ID, "")
			if e != nil || acoustic.Availability != "current" || len(acoustic.Decisions) != 1 || acoustic.Decisions[0].Independent == sameSource {
				t.Fatalf("incorrect lineage: %+v %v", acoustic, e)
			}
			f, e := s.FreezeSpeakerDiscovery(ctx, target.ID, 100, "")
			if e != nil {
				t.Fatal(e)
			}
			p, _ := sd.NormalizePolicy(sd.Policy{})
			resolved, e := sd.Resolve(f.Context, nil, p)
			if e != nil {
				t.Fatal(e)
			}
			expected := "matched"
			if sameSource {
				expected = "unknown"
			}
			if resolved[0].Outcome != expected {
				t.Fatal(resolved)
			}
			if !sameSource {
				replacement := speakerOutputFixture(t, s, ds)
				if _, e = s.SetSpeakerProfile(ctx, contracts.ID(), sp.ID, frozen.Profiles[0].Revision, replacement.ID); e != nil {
					t.Fatal(e)
				}
				stale, e := s.SpeakerDiscoveryAcoustic(ctx, target.ID, w.ID)
				if e != nil || stale.Availability != "stale" || len(stale.Decisions) != 0 {
					t.Fatal("replaced profile retained prediction", stale, e)
				}
			}
			if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), target.Revision, SpeakerMapping{RecordingID: target.ID, LocalSpeakerID: voice, SpeakerID: sp.ID, DocumentDigest: target.DocumentDigest, Revision: func() int64 { m, _ := s.SpeakerMappings(ctx, target.ID); return m[0].Revision }()}); e != nil {
				t.Fatal(e)
			}
			stale, e := s.SpeakerDiscoveryAcoustic(ctx, target.ID, w.ID)
			if e != nil || stale.Availability != "stale" || len(stale.Decisions) != 0 {
				t.Fatal("owner correction retained prediction", stale, e)
			}
		})
	}
}
