// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func selectionSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, recording, local := evidenceRecording(t, s)
	person, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Same name"}})
	if e != nil {
		t.Fatal(e)
	}
	other, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Same name"}})
	if e != nil {
		t.Fatal(e)
	}
	mapping, e := s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, SpeakerMapping{RecordingID: recording.ID, LocalSpeakerID: local, SpeakerID: person.Speaker.ID, DocumentDigest: recording.DocumentDigest})
	if e != nil {
		t.Fatal(e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: person.Speaker.ID, Limit: 1})
	if e != nil || len(page.References) != 1 {
		t.Fatal("direct current cues", page, e)
	}
	empty, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: other.Speaker.ID})
	if e != nil || len(empty.References) != 0 {
		t.Fatal("equal text merged identities", empty, e)
	}
	current, e := s.ResolveEvidence(ctx, page.References[0])
	if e != nil || len(current.Cue) == 0 {
		t.Fatal(current, e)
	}
	mapping.SpeakerID = other.Speaker.ID
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, mapping); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ResolveEvidence(ctx, page.References[0]); e == nil {
		t.Fatal("stale mapping evidence accepted")
	}
	snapshot, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snapshot); e != nil {
		t.Fatal("selection current snapshot", e)
	}
}
func TestSQLiteCurrentSpeakerSelection(t *testing.T) {
	selectionSuite(t, localStore(t, contracts.ID()))
}
