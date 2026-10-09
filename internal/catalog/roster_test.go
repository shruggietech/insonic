// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestRosterAuthority(t *testing.T) {
	rosterSuite(t, localStore(t, contracts.ID()))
}
func rosterSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, entry := entryFixture(t, s)
	if _, e := s.CommitLibrary(ctx, claim, entry); e != nil {
		t.Fatal(e)
	}
	id := contracts.ID()
	speaker, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: id, Name: "Roster Speaker", State: "active"}, Aliases: []SpeakerAlias{{ID: contracts.ID(), SpeakerID: id, Text: "Roster Alias", State: "active", Provenance: []byte(`{}`)}}})
	if e != nil {
		t.Fatal(e)
	}
	before, e := s.Roster(ctx, entry.ID)
	if e != nil || before.Declared || before.Revision != 0 {
		t.Fatalf("absent: %+v %v", before, e)
	}
	revision, _ := s.Revision(ctx)
	for _, mode := range []string{"add", "remove"} {
		if _, e = s.MutateRoster(ctx, contracts.ID(), entry.ID, 0, mode, []string{}); e == nil {
			t.Fatal("empty roster mutation accepted", mode)
		}
	}
	after, _ := s.Revision(ctx)
	absent, _ := s.Roster(ctx, entry.ID)
	if after != revision || absent.Declared || absent.Revision != 0 {
		t.Fatal("empty roster mutation changed authority")
	}
	op := contracts.ID()
	r, e := s.MutateRoster(ctx, op, entry.ID, 0, "add", []string{"Roster Alias", id})
	if e != nil || !r.Declared || len(r.Members) != 1 {
		t.Fatalf("add: %+v %v", r, e)
	}
	replay, e := s.MutateRoster(ctx, op, entry.ID, 0, "add", []string{"Roster Alias", id})
	if e != nil || replay.Revision != r.Revision {
		t.Fatalf("replay: %+v %v", replay, e)
	}
	if _, e = s.MutateRoster(ctx, contracts.ID(), entry.ID, 0, "clear", nil); e == nil {
		t.Fatal("stale roster accepted")
	}
	same, e := s.MutateRoster(ctx, contracts.ID(), entry.ID, r.Revision, "add", []string{id})
	if e != nil || same.Revision != r.Revision {
		t.Fatalf("set no-op: %+v %v", same, e)
	}
	speaker.Speaker.State = "inactive"
	if _, e = s.PutSpeaker(ctx, contracts.ID(), speaker.Speaker.Revision, speaker); e != nil {
		t.Fatal(e)
	}
	read, e := s.Roster(ctx, entry.ID)
	if e != nil || read.Members[0].State != "inactive" {
		t.Fatalf("inactive: %+v %v", read, e)
	}
	if _, e = s.MutateRoster(ctx, contracts.ID(), entry.ID, r.Revision, "replace", []string{id}); e == nil {
		t.Fatal("inactive election")
	}
	clear, e := s.MutateRoster(ctx, contracts.ID(), entry.ID, r.Revision, "remove", []string{id})
	if e != nil || !clear.Declared || len(clear.Members) != 0 || clear.Revision <= r.Revision {
		t.Fatalf("clear: %+v %v", clear, e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restored := localStore(t, snap.WorkspaceID)
	if e = restored.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	got, e := restored.Roster(ctx, entry.ID)
	if e != nil || got.Revision != clear.Revision || !got.Declared {
		t.Fatalf("portable: %+v %v", got, e)
	}
	snap.Records.Rosters[0].Revision++
	snap.Digest, _ = snap.digest()
	if e = localStore(t, snap.WorkspaceID).Restore(ctx, snap); e == nil {
		t.Fatal("forged roster revision restored")
	}
}

func TestRosterWorkspaceAndAmbiguity(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	foreign := *s
	foreign.workspace = contracts.ID()
	if e := foreign.ensureWorkspace(ctx); e != nil {
		t.Fatal(e)
	}
	sp, e := foreign.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Foreign", State: "active"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ResolveSpeakers(ctx, []string{sp.Speaker.ID}); e == nil {
		t.Fatal("foreign identity selected")
	}
	for i := 0; i < 2; i++ {
		if _, e = s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Ambiguous", State: "active"}}); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = s.ResolveSpeakers(ctx, []string{"Ambiguous"}); e == nil {
		t.Fatal("ambiguous name selected")
	}
}
