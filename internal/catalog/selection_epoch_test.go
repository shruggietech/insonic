package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func selectionEpochSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, one, voiceOne := evidenceRecording(t, s)
	_, two, voiceTwo := evidenceRecording(t, s)
	person, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Person"}})
	if e != nil {
		t.Fatal(e)
	}
	other, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Other"}})
	if e != nil {
		t.Fatal(e)
	}
	for _, input := range []struct {
		recording Recording
		voice     string
	}{{one, voiceOne}, {two, voiceTwo}} {
		if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), input.recording.Revision, SpeakerMapping{RecordingID: input.recording.ID, LocalSpeakerID: input.voice, SpeakerID: person.Speaker.ID, DocumentDigest: input.recording.DocumentDigest}); e != nil {
			t.Fatal(e)
		}
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: person.Speaker.ID, Limit: 1})
	if e != nil || len(page.References) != 1 || page.Next == "" {
		t.Fatal("first page", page, e)
	}
	if _, e = s.RenewWork(ctx, claim, time.Minute); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutTerm(ctx, contracts.ID(), 0, Term{ID: contracts.ID(), Canonical: "unrelated", Variants: json.RawMessage(`[]`)}); e != nil {
		t.Fatal(e)
	}
	continued, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: person.Speaker.ID, Limit: 1, Cursor: page.Next})
	if e != nil || len(continued.References) != 1 || continued.References[0].RecordingID == page.References[0].RecordingID {
		t.Fatal("unrelated writes invalidated continuation", continued, e)
	}
	ref := page.References[0]
	mappings, e := s.SpeakerMappings(ctx, ref.RecordingID)
	if e != nil {
		t.Fatal(e)
	}
	m := mappings[0]
	m.SpeakerID = other.Speaker.ID
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), ref.RecordingRevision, m); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: person.Speaker.ID, Limit: 1, Cursor: page.Next}); e == nil {
		t.Fatal("relevant mapping edit accepted continuation")
	}
}
func TestSQLiteSpeakerSelectionEpoch(t *testing.T) {
	selectionEpochSuite(t, localStore(t, contracts.ID()))
}
