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

func recordingFixture(t *testing.T, s *Store) (Work, LibraryEntry, Recording) {
	t.Helper()
	ctx := context.Background()
	claim, entry := entryFixture(t, s)
	entry, e := s.CommitLibrary(ctx, claim, entry)
	if e != nil {
		t.Fatal(e)
	}
	work, e := s.EnqueueWork(ctx, contracts.ID(), "recording.process", json.RawMessage(`{"media_id":"fixture"}`))
	if e != nil {
		t.Fatal(e)
	}
	work, e = s.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	r := Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: entry.Revision, State: "no-speech", Document: json.RawMessage("null"), SourceMap: json.RawMessage(`{}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	return work, entry, r
}
func TestCurrentRecordingValidation(t *testing.T) {
	_, _, r := recordingFixture(t, localStore(t, contracts.ID()))
	if !validRecording(r) {
		t.Fatal("explicit no-speech rejected")
	}
	r.Document = json.RawMessage(`{"speaker_attributions":[]}`)
	if validRecording(r) {
		t.Fatal("non-null no-speech accepted")
	}
	r.Document = json.RawMessage("null")
	r.State = "ready"
	if validRecording(r) {
		t.Fatal("ready null accepted")
	}
	r.State = "no-speech"
	r.SourceMap = json.RawMessage(`{"turns":[{"start_us":0,"end_us":1}]}`)
	if validRecording(r) {
		t.Fatal("second assignment store accepted")
	}
}
func TestSQLiteCurrentRecordingProofs(t *testing.T) { recordingSuite(t, localStore(t, contracts.ID())) }
func recordingSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, entry, r := recordingFixture(t, s)
	p := availableArtifact(t, s)
	r.MappedAudioPublicationID = &p.ID
	got, e := s.CommitRecording(ctx, claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	if got.Revision < 1 {
		t.Fatal("revision missing")
	}
	if _, e = s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Minute); e == nil {
		t.Fatal("current mapped audio retired")
	}
	replay, e := s.CommitRecording(ctx, claim, 0, r)
	if e != nil || replay.Revision != got.Revision {
		t.Fatal("replay", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(snap.Records.Recordings) != 1 {
		t.Fatal("master document not embedded")
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	altered := snap
	altered.Records.Recordings = append([]Recording(nil), snap.Records.Recordings...)
	altered.Records.Recordings[0].State = "no-timed-subtitles"
	altered.Digest, _ = altered.digest()
	if e = localStore(t, s.workspace).Restore(ctx, altered); e == nil {
		t.Fatal("forged latest recording proof accepted")
	}
	claim2, _, _ := recordingFixture(t, s)
	// Reuse separate live work with the same immutable source identity.
	next := got
	next.MappedAudioPublicationID = nil
	next.State = "no-timed-subtitles"
	next.SourceRevision = entry.Revision
	if _, e = s.CommitRecording(ctx, claim2, 0, next); e == nil {
		t.Fatal("stale CAS accepted")
	}
	current, e := s.CommitRecording(ctx, claim2, got.Revision, next)
	if e != nil {
		t.Fatal(e)
	}
	pending, e := s.Cleanups(ctx)
	if e != nil || len(pending) != 1 || pending[0].ID != p.ID {
		t.Fatal("cleanup missing", e)
	}
	newer, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(newer)
	if strings.Contains(string(raw), `"recording":{"document"`) {
		t.Fatal("document copied into receipt")
	}
	rollback := newer
	rollback.Records.Recordings = []Recording{got}
	rollback.Digest, _ = rollback.digest()
	if e = localStore(t, s.workspace).Restore(ctx, rollback); e == nil {
		t.Fatal("old accepted result revived")
	}
	if current.Revision == got.Revision {
		t.Fatal("replacement revision unchanged")
	}
	cancelled, e := s.CancelWork(ctx, contracts.ID(), claim2.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecording(ctx, cancelled, current.Revision, current); e == nil {
		t.Fatal("cancelled authority accepted")
	}
}

func TestCurrentSpeakerMapping(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	claim, _, r := recordingFixture(t, s)
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc map[string]json.RawMessage
	if e = json.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}
	delete(doc, "media_timing")
	var cues []map[string]json.RawMessage
	if e = json.Unmarshal(doc["cues"], &cues); e != nil || len(cues) == 0 {
		t.Fatal("cue fixture", e)
	}
	local := "Imported voice: \u00e9 \U0001f399"
	cues[0]["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": local, "start_milliseconds": 0, "end_milliseconds": 100}})
	doc["cues"], _ = json.Marshal(cues)
	r.Document, _ = json.Marshal(doc)
	r.DocumentDigest = hash(r.Document)
	r.State = "ready"
	current, e := s.CommitRecording(ctx, claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	a, b := contracts.ID(), contracts.ID()
	rev, _ := s.Revision(ctx)
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Speakers: []Speaker{{ID: a, Name: "A"}, {ID: b, Name: "B"}}}}); e != nil {
		t.Fatal(e)
	}
	m := SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: local, SpeakerID: a, DocumentDigest: r.DocumentDigest}
	first, e := s.SetSpeakerMapping(ctx, contracts.ID(), current.Revision, m)
	if e != nil {
		t.Fatal(e)
	}
	m.SpeakerID = b
	if _, err := s.SetSpeakerMapping(ctx, contracts.ID(), current.Revision, m); err == nil {
		t.Fatal("stale mapping correction accepted")
	}
	m.Revision = first.Revision
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), current.Revision, m); e != nil {
		t.Fatal(e)
	}
	unchanged, e := s.Recording(ctx, r.ID)
	if e != nil || string(unchanged.Document) != string(current.Document) {
		t.Fatal("mapping rewrote document", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("mapping restore", e)
	}
	forged := snap
	forged.Records.SpeakerMappings = []SpeakerMapping{first}
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Fatal("old known identity revived")
	}
	replacement := current
	replacement.State = "no-speech"
	replacement.Document = json.RawMessage("null")
	replacement.DocumentDigest = ""
	nextClaim, _, _ := recordingFixture(t, s)
	if _, e = s.CommitRecording(ctx, nextClaim, current.Revision, replacement); e != nil {
		t.Fatal(e)
	}
	mappings, e := s.SpeakerMappings(ctx, r.ID)
	if e != nil || len(mappings) != 0 {
		t.Fatal("obsolete local mappings retained", e)
	}
}
