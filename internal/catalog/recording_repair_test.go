// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSQLiteRecordingRepairFencesAndReplay(t *testing.T) {
	recordingRepairSuite(t, localStore(t, contracts.ID()))
}
func recordingRepairSuite(t *testing.T, s *Store) {
	recordingRepairConcurrentFenceSuite(t, s)
	t.Helper()
	ctx := context.Background()
	claim, _, r := recordingFixture(t, s)
	current, e := s.CommitRecording(ctx, claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(map[string]any{"media_id": r.ID, "expected_revision": current.Revision, "source_revision": current.SourceRevision})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.repair", raw)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	mappings, _ := s.SpeakerMappings(ctx, r.ID)
	md, _ := intent(mappings)
	f := RecordingRepairFence{ExpectedRevision: current.Revision, DocumentDigest: current.DocumentDigest, SourceRevision: current.SourceRevision, SourceDigest: current.SourceDigest, MappingDigest: strings.Repeat("f", 64)}
	decision := json.RawMessage(`{"kind":"recording-repair","schema_version":"1.1.0","status":"unchanged","reason":"not_improved"}`)
	if _, e = s.CommitRecordingRepair(ctx, w, f, nil, decision); e == nil {
		t.Fatal("stale mapping fence accepted")
	}
	f.MappingDigest = md
	out, e := s.CommitRecordingRepair(ctx, w, f, nil, decision)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.CommitRecordingRepair(ctx, w, f, nil, decision)
	if e != nil || string(out) != string(again) {
		t.Fatalf("replay: %s %s %v", out, again, e)
	}
	same, e := s.Recording(ctx, r.ID)
	if e != nil || same.Revision != current.Revision {
		t.Fatal("unchanged decision replaced recording")
	}
	accepted, ok, e := s.RecordingRepairDecision(ctx, w.ID, r.ID)
	if e != nil || !ok || string(accepted) != string(out) {
		t.Fatal("decision recovery", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("unchanged receipt restore", e)
	}
	w2, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.repair", raw)
	if e != nil {
		t.Fatal(e)
	}
	w2, e = s.ClaimWork(ctx, w2.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	next := current
	next.State = "no-timed-subtitles"
	acceptedDecision := json.RawMessage(`{"kind":"recording-repair","schema_version":"1.1.0","status":"accepted","reason":"improved"}`)
	if _, e = s.CommitRecordingRepair(ctx, w2, f, &next, acceptedDecision); e != nil {
		t.Fatal(e)
	}
	changed, _ := s.Recording(ctx, r.ID)
	if changed.Revision == current.Revision {
		t.Fatal("repair did not commit")
	}
	if _, e = s.CommitRecordingRepair(ctx, w2, f, &next, acceptedDecision); e != nil {
		t.Fatal("accepted replay", e)
	}
}

func recordingRepairConcurrentFenceSuite(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	claim, _, r := recordingFixture(t, s)
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	delete(doc, "media_timing")
	var cues []map[string]json.RawMessage
	json.Unmarshal(doc["cues"], &cues)
	cues[0]["speaker_attributions"] = json.RawMessage(`[{"speaker_id":"repair-voice","start_milliseconds":0,"end_milliseconds":100}]`)
	doc["cues"], _ = json.Marshal(cues)
	r.Document, _ = json.Marshal(doc)
	r.DocumentDigest = hash(r.Document)
	r.State = "ready"
	current, e := s.CommitRecording(ctx, claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	payload, _ := json.Marshal(map[string]any{"media_id": r.ID, "expected_revision": current.Revision, "source_revision": current.SourceRevision})
	work, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.repair", payload)
	if e != nil {
		t.Fatal(e)
	}
	work, e = s.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	mappings, _ := s.SpeakerMappings(ctx, r.ID)
	f := RecordingRepairFence{ExpectedRevision: current.Revision, DocumentDigest: current.DocumentDigest, SourceRevision: current.SourceRevision, SourceDigest: current.SourceDigest, MappingDigest: SpeakerMappingsDigest(mappings)}
	decision := json.RawMessage(`{"kind":"recording-repair","schema_version":"1.1.0","status":"unchanged","reason":"not_improved"}`)
	for _, change := range []func(*RecordingRepairFence){func(f *RecordingRepairFence) { f.DocumentDigest = strings.Repeat("f", 64) }, func(f *RecordingRepairFence) { f.SourceDigest = strings.Repeat("f", 64) }, func(f *RecordingRepairFence) { f.SourceRevision++ }, func(f *RecordingRepairFence) { f.ExpectedRevision++ }} {
		altered := f
		change(&altered)
		if _, e = s.CommitRecordingRepair(ctx, work, altered, nil, decision); e == nil {
			t.Fatal("stale fence accepted")
		}
	}
	owner := work
	owner.Generation++
	if _, e = s.CommitRecordingRepair(ctx, owner, f, nil, decision); e == nil {
		t.Fatal("obsolete owner accepted")
	}
	speaker := contracts.ID()
	rev, _ := s.Revision(ctx)
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Speakers: []Speaker{{ID: speaker, Name: "Current correction"}}}}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), current.Revision, SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: "repair-voice", SpeakerID: speaker, DocumentDigest: r.DocumentDigest}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecordingRepair(ctx, work, f, nil, decision); e == nil {
		t.Fatal("mapping-only correction overwritten")
	}
	same, _ := s.Recording(ctx, r.ID)
	if same.Revision != current.Revision || string(same.Document) != string(current.Document) {
		t.Fatal("mapping-only correction changed document")
	}
	accepted := json.RawMessage(`{"kind":"recording-repair","schema_version":"1.1.0","status":"accepted","reason":"improved"}`)
	if _, e = s.CommitRecordingRepair(ctx, work, f, &current, accepted); e == nil {
		t.Fatal("accepted repair bypassed mapping fence")
	}
	mappings, _ = s.SpeakerMappings(ctx, r.ID)
	f.MappingDigest = SpeakerMappingsDigest(mappings)
	if _, e = s.CancelWork(ctx, contracts.ID(), work.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecordingRepair(ctx, work, f, nil, decision); e == nil {
		t.Fatal("cancelled owner accepted")
	}
}
