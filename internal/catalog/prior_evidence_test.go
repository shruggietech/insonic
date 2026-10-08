// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"testing"
	"time"
)

func priorEvidenceFixture(t *testing.T, s *Store, knownStream bool) (Recording, string, string) {
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
	first, second := contracts.ID(), contracts.ID()
	if second < first {
		first, second = second, first
	}
	cues[0]["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": first, "start_milliseconds": 0, "end_milliseconds": 100}, {"speaker_id": first, "start_milliseconds": 0, "end_milliseconds": 100}, {"speaker_id": second, "start_milliseconds": 50, "end_milliseconds": 150}, {"speaker_id": second}})
	other := map[string]json.RawMessage{}
	for k, v := range cues[0] {
		other[k] = v
	}
	other["id"] = json.RawMessage(`"cue-000001"`)
	other["ordinal"] = json.RawMessage(`1`)
	other["source_order"] = json.RawMessage(`1`)
	other["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": first, "start_milliseconds": 0, "end_milliseconds": 100}})
	cues = append(cues, other)
	doc["cues"], _ = json.Marshal(cues)
	for _, key := range []string{"document", "stats"} {
		var metadata map[string]json.RawMessage
		json.Unmarshal(doc[key], &metadata)
		metadata["cue_count"] = json.RawMessage(`2`)
		doc[key], _ = json.Marshal(metadata)
	}
	r.Document, _ = json.Marshal(doc)
	r.DocumentDigest = hash(r.Document)
	r.State = "ready"
	if knownStream {
		r.SourceMap = json.RawMessage(`{"stream_index":0,"channel":null}`)
	}
	r, e = s.CommitRecording(ctx, claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	return r, first, second
}
func priorEvidenceSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	recording, first, second := priorEvidenceFixture(t, s, true)
	person, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Known"}})
	if e != nil {
		t.Fatal(e)
	}
	var mapping SpeakerMapping
	for _, voice := range []string{first, second} {
		m, e := s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, SpeakerMapping{RecordingID: recording.ID, LocalSpeakerID: voice, SpeakerID: person.Speaker.ID, DocumentDigest: recording.DocumentDigest})
		if e != nil {
			t.Fatal(e)
		}
		if voice == second {
			mapping = m
		}
	}
	scope := SpeakerSelection{SpeakerID: person.Speaker.ID, Limit: 1}
	page, e := s.CurrentSpeakerReferences(ctx, scope)
	if e != nil || len(page.References) != 1 || page.Next == "" {
		t.Fatal(page, e)
	}
	flags, e := s.ComparePriorSpeakerEvidence(ctx, scope, page.References[0], page.Epoch)
	if e != nil || len(flags) != 2 || flags[0].Duplicate || !flags[1].Duplicate || flags[0].LocalVoices != 2 || !flags[0].FirstReference {
		t.Fatal("same cue duplicates", flags, e)
	}
	scope.Cursor = page.Next
	next, e := s.CurrentSpeakerReferences(ctx, scope)
	if e != nil {
		t.Fatal(e)
	}
	flags, e = s.ComparePriorSpeakerEvidence(ctx, scope, next.References[0], next.Epoch)
	if e != nil || len(flags) != 2 || flags[0].Duplicate || !flags[0].Overlap || flags[1].Duplicate || flags[1].Overlap || flags[0].FirstReference {
		t.Fatal("cross-page overlap/untimed", flags, e)
	}
	scope.Cursor = next.Next
	third, e := s.CurrentSpeakerReferences(ctx, scope)
	if e != nil {
		t.Fatal(e)
	}
	flags, e = s.ComparePriorSpeakerEvidence(ctx, scope, third.References[0], third.Epoch)
	if e != nil || len(flags) != 1 || !flags[0].Duplicate || flags[0].FirstReference {
		t.Fatal("cross-page duplicate", flags, e)
	}
	if _, e = s.ComparePriorSpeakerEvidence(ctx, scope, third.References[0], ""); e == nil {
		t.Fatal("empty epoch accepted")
	}
	other, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Other"}})
	if e != nil {
		t.Fatal(e)
	}
	mapping.SpeakerID = other.Speaker.ID
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, mapping); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ComparePriorSpeakerEvidence(ctx, scope, third.References[0], third.Epoch); e == nil {
		t.Fatal("changed prior evidence accepted stale epoch")
	}
	// A unrelated live work heartbeat leaves the target evidence epoch stable.
	scope.Cursor = ""
	current, e := s.CurrentSpeakerReferences(ctx, scope)
	if e != nil {
		t.Fatal(e)
	}
	work, e := s.EnqueueWork(ctx, contracts.ID(), "fixture", json.RawMessage(`{}`))
	if e != nil {
		t.Fatal(e)
	}
	work, e = s.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.RenewWork(ctx, work, time.Minute); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ComparePriorSpeakerEvidence(ctx, scope, current.References[0], current.Epoch); e != nil {
		t.Fatal("unrelated heartbeat", e)
	}
	// Cross-recording equivalence requires the same proven stream/channel.
	another, voice, _ := priorEvidenceFixture(t, s, true)
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), another.Revision, SpeakerMapping{RecordingID: another.ID, LocalSpeakerID: voice, SpeakerID: person.Speaker.ID, DocumentDigest: another.DocumentDigest}); e != nil {
		t.Fatal(e)
	}
	scope.Limit = 100
	page, e = s.CurrentSpeakerReferences(ctx, scope)
	if e != nil {
		t.Fatal(e)
	}
	laterID := recording.ID
	if another.ID > laterID {
		laterID = another.ID
	}
	var later CurrentReference
	for _, ref := range page.References {
		if ref.RecordingID == laterID && ref.CueID == "cue-000000" {
			later = ref
			break
		}
	}
	flags, e = s.ComparePriorSpeakerEvidence(ctx, scope, later, page.Epoch)
	if e != nil || len(flags) == 0 || !flags[0].Duplicate {
		t.Fatal("cross-recording duplicate", flags, e)
	}
	scoped := SpeakerSelection{SpeakerID: person.Speaker.ID, RecordingID: laterID, Limit: 100}
	localPage, e := s.CurrentSpeakerReferences(ctx, scoped)
	if e != nil {
		t.Fatal(e)
	}
	flags, e = s.ComparePriorSpeakerEvidence(ctx, scoped, localPage.References[0], localPage.Epoch)
	if e != nil || len(flags) == 0 || flags[0].Duplicate {
		t.Fatal("recording scope included external prior evidence", flags, e)
	}
	unknownOne, unknownVoiceOne, _ := priorEvidenceFixture(t, s, false)
	unknownTwo, unknownVoiceTwo, _ := priorEvidenceFixture(t, s, false)
	for _, input := range []struct {
		r     Recording
		voice string
	}{{unknownOne, unknownVoiceOne}, {unknownTwo, unknownVoiceTwo}} {
		if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), input.r.Revision, SpeakerMapping{RecordingID: input.r.ID, LocalSpeakerID: input.voice, SpeakerID: person.Speaker.ID, DocumentDigest: input.r.DocumentDigest}); e != nil {
			t.Fatal(e)
		}
	}
	page, e = s.CurrentSpeakerReferences(ctx, scope)
	if e != nil {
		t.Fatal(e)
	}
	laterID = unknownOne.ID
	if unknownTwo.ID > laterID {
		laterID = unknownTwo.ID
	}
	for _, ref := range page.References {
		if ref.RecordingID == laterID && ref.CueID == "cue-000000" {
			later = ref
			break
		}
	}
	flags, e = s.ComparePriorSpeakerEvidence(ctx, scope, later, page.Epoch)
	if e != nil || len(flags) == 0 || flags[0].Duplicate {
		t.Fatal("unknown stream fabricated cross-recording equivalence", flags, e)
	}
}
func TestSQLitePriorSpeakerEvidence(t *testing.T) {
	priorEvidenceSuite(t, localStore(t, contracts.ID()))
}
