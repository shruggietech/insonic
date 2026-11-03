// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"testing"
)

func contextualDiscovery(t *testing.T, s *Store, r Recording, voice string, names ...string) SpeakerDiscovery {
	t.Helper()
	ctx := context.Background()
	f, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
	if e != nil {
		t.Fatal(e)
	}
	observations := []any{}
	for index, name := range names {
		cue := f.Context.Cues[min(index, len(f.Context.Cues)-1)]
		observations = append(observations, map[string]any{"class": "self-identification", "name": name, "voice_id": voice, "references": []any{map[string]any{"cue_id": cue.ID, "quote": cue.Text}}, "uncertain": false})
	}
	raw, _ := json.Marshal(map[string]any{"version": 1, "observations": observations})
	h, e := sd.Parse(raw, f.Context, 128)
	if e != nil {
		t.Fatal(e)
	}
	p, _ := sd.NormalizePolicy(sd.Policy{})
	d, e := sd.Resolve(f.Context, h, p)
	if e != nil {
		t.Fatal(e)
	}
	payload, _ := json.Marshal(map[string]any{"recording_id": r.ID, "source_epoch": f.SourceEpoch, "max_cues": 100, "expected_revision": f.ExpectedRevision})
	w := speakerWorkPayload(t, s, "speakers.discover", payload)
	_, e = s.AcceptSpeakerDiscovery(ctx, w, f, sd.Aggregate{Version: 1, OwnerEvidence: f.Context.OwnerEvidence, Coverage: f.Coverage, Hypotheses: h, Acoustic: f.Context.Acoustic, Policy: p, Decisions: d})
	if e != nil {
		t.Fatal(e)
	}
	out, e := s.SpeakerDiscovery(ctx, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func TestContextualApplicationReplayCorrectionAndRetraction(t *testing.T) {
	contextualApplicationSuite(t, localStore(t, contracts.ID()))
}
func contextualApplicationSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, r, voice := evidenceRecording(t, s)
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Research"}})
	if e != nil {
		t.Fatal(e)
	}
	d := contextualDiscovery(t, s, r, voice, "Research")
	op := contracts.ID()
	out, e := s.ApplySpeakerDiscovery(ctx, op, r.ID, d.Revision, nil)
	if e != nil || out.Applied != 1 {
		t.Fatal(out, e)
	}
	ms, e := s.SpeakerMappings(ctx, r.ID)
	if e != nil || len(ms) != 1 || ms[0].Origin != "contextual" {
		t.Fatal(ms, e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: sp.Speaker.ID, ConfirmedOnly: true})
	if e != nil || len(page.References) != 1 {
		t.Fatal(page, e)
	}
	if _, e = s.ApplySpeakerDiscovery(ctx, op, r.ID, d.Revision, nil); e != nil {
		t.Fatal("lost response replay", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("contextual restore", e)
	}
	m := ms[0]
	m.SpeakerID = sp.Speaker.ID
	m, e = s.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ApplySpeakerDiscovery(ctx, op, r.ID, d.Revision, nil); e != nil {
		t.Fatal(e)
	}
	if _, e = s.RetractSpeakerDiscovery(ctx, contracts.ID(), r.ID, r.Revision, []MappingRevision{{ID: ms[0].ID, Revision: ms[0].Revision}}); e == nil {
		t.Fatal("retraction overwrote correction")
	}
	ms, _ = s.SpeakerMappings(ctx, r.ID)
	if len(ms) != 1 || ms[0].Origin != "manual" || ms[0].Revision != m.Revision {
		t.Fatal(ms)
	}
}

func TestContextualInvalidationAndPortableTamper(t *testing.T) {
	contextualInvalidationSuite(t, func() *Store { return localStore(t, contracts.ID()) })
}
func contextualInvalidationSuite(t *testing.T, factory func() *Store) {
	for _, mode := range []string{"identity", "source", "recording-revision", "contradiction", "abstention", "tamper"} {
		t.Run(mode, func(t *testing.T) {
			s := factory()
			ctx := context.Background()
			_, r, voice := evidenceRecording(t, s)
			if mode == "contradiction" {
				var doc map[string]any
				decoder := json.NewDecoder(bytes.NewReader(r.Document))
				decoder.UseNumber()
				if err := decoder.Decode(&doc); err != nil {
					t.Fatal(err)
				}
				cues := doc["cues"].([]any)
				raw, _ := json.Marshal(cues[0])
				var second map[string]any
				json.Unmarshal(raw, &second)
				second["id"], second["ordinal"], second["source_order"] = "cue-000001", 1, 1
				second["payload"] = map[string]any{"raw_text": "Other.", "plain_text": "Other.", "lines": []string{"Other."}}
				second["speaker_attributions"] = []any{map[string]any{"speaker_id": voice, "start_milliseconds": 200, "end_milliseconds": 300}}
				doc["cues"] = append(cues, second)
				doc["document"].(map[string]any)["cue_count"] = 2
				doc["stats"].(map[string]any)["cue_count"] = 2
				r.Document, _ = json.Marshal(doc)
				r.SourceMap = json.RawMessage(`{"start_numerator":"0","start_denominator":"1","stream_index":0,"policy":"supplied-document-source-clock;no-retiming"}`)
				r.DocumentDigest = hash(r.Document)
				w := speakerWorkPayload(t, s, "recordings.process", json.RawMessage("{}"))
				var err error
				r, err = s.CommitRecording(ctx, w, r.Revision, r)
				if err != nil {
					t.Fatal(err)
				}
			}
			sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Research"}})
			if e != nil {
				t.Fatal(e)
			}
			d := contextualDiscovery(t, s, r, voice, "Research")
			op := contracts.ID()
			if _, e = s.ApplySpeakerDiscovery(ctx, op, r.ID, d.Revision, nil); e != nil {
				t.Fatal(e)
			}
			ms, _ := s.SpeakerMappings(ctx, r.ID)
			switch mode {
			case "identity":
				sp.Speaker.Name = "Corrected identity"
				if _, e = s.PutSpeaker(ctx, contracts.ID(), sp.Speaker.Revision, sp); e != nil {
					t.Fatal(e)
				}
			case "source", "recording-revision":
				if mode == "source" {
					r.Document = bytes.ReplaceAll(r.Document, []byte("Research source."), []byte("Replacement source."))
					r.DocumentDigest = hash(r.Document)
				}
				w := speakerWorkPayload(t, s, "recordings.process", json.RawMessage("{}"))
				if _, e = s.CommitRecording(ctx, w, r.Revision, r); e != nil {
					t.Fatal(e)
				}
			case "contradiction":
				if _, e = s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Other"}}); e != nil {
					t.Fatal(e)
				}
				conflicting := contextualDiscovery(t, s, r, voice, "Research", "Other")
				out, err := s.ApplySpeakerDiscovery(ctx, contracts.ID(), r.ID, conflicting.Revision, nil)
				if err != nil || out.Retracted != 1 || out.Conflicting != 1 {
					t.Fatal(out, err)
				}
			case "abstention":
				f, e := s.FreezeSpeakerDiscovery(ctx, r.ID, 100, "")
				if e != nil {
					t.Fatal(e)
				}
				p, _ := sd.NormalizePolicy(sd.Policy{})
				decisions, _ := sd.Resolve(f.Context, nil, p)
				raw, _ := json.Marshal(map[string]any{"recording_id": r.ID, "source_epoch": f.SourceEpoch, "max_cues": 100, "expected_revision": f.ExpectedRevision})
				w := speakerWorkPayload(t, s, "speakers.discover", raw)
				if _, e = s.AcceptSpeakerDiscovery(ctx, w, f, sd.Aggregate{Version: 1, OwnerEvidence: f.Context.OwnerEvidence, Coverage: f.Coverage, Hypotheses: []sd.Hypothesis{}, Acoustic: f.Context.Acoustic, Policy: p, Decisions: decisions}); e != nil {
					t.Fatal(e)
				}
				current, e := s.SpeakerDiscovery(ctx, r.ID)
				if e != nil {
					t.Fatal(e)
				}
				out, e := s.ApplySpeakerDiscovery(ctx, contracts.ID(), r.ID, current.Revision, nil)
				if e != nil || out.Retracted != 0 || out.Preserved != 1 {
					t.Fatal(out, e)
				}
			case "tamper":
				snap, e := s.Export(ctx)
				if e != nil {
					t.Fatal(e)
				}
				var proof ContextualLineage
				json.Unmarshal(snap.Records.SpeakerMappings[0].Provenance, &proof)
				proof.Evidence[0].Roots = []string{hash([]byte("forged independent root"))}
				snap.Records.SpeakerMappings[0].Provenance, _ = json.Marshal(proof)
				snap.Digest, _ = snap.digest()
				if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
					t.Fatal("forged contextual proof accepted")
				}
				return
			}
			if mode == "abstention" {
				current, e := s.SpeakerMappings(ctx, r.ID)
				if e != nil || len(current) != 1 || current[0].Revision != ms[0].Revision {
					t.Fatal(current, e)
				}
				return
			}
			current, e := s.SpeakerMappings(ctx, r.ID)
			if e != nil || len(current) != 0 {
				t.Fatal(mode, current, e)
			}
			if _, e = s.ApplySpeakerDiscovery(ctx, op, r.ID, d.Revision, nil); e != nil {
				t.Fatal("old receipt replay", e)
			}
			current, _ = s.SpeakerMappings(ctx, r.ID)
			if len(current) != 0 {
				t.Fatal("replay resurrected stale mapping", ms, current)
			}
			snap, e := s.Export(ctx)
			if e != nil {
				t.Fatal("export", e)
			}
			if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
				t.Fatal("restore", e)
			}
		})
	}
}
