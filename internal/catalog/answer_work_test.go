// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func answerWorkSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	epoch, err := s.AnswerEpoch(ctx, nil, 100000, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	profile := Profile{ID: contracts.ID(), Revision: 1, Role: "storage", Adapter: "filesystem", Version: contracts.Version, Configuration: []byte("{\"root\":\"artifacts\"}")}
	rev, _ := s.Revision(ctx)
	if _, err = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Profiles: []Profile{profile}}}); err != nil {
		t.Fatal(err)
	}
	makeCandidate := func() (Work, AnswerCandidate, json.RawMessage) {
		id := contracts.ID()
		candidate := AnswerCandidate{PublicationID: contracts.ID(), PackageID: contracts.ID(), Digest: strings.Repeat("a", 64), Size: 3, SourceEpoch: epoch.Digest}
		payload, _ := json.Marshal(map[string]any{"publication_id": candidate.PublicationID, "request": answers.Request{Version: contracts.Version, Question: "q"}})
		if _, e := s.EnqueueWork(ctx, id, "answers.request", payload); e != nil {
			t.Fatal(e)
		}
		claim, e := s.ClaimWork(ctx, id, contracts.ID(), time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		p := Publication{ID: candidate.PublicationID, ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: profile.ID, ProfileRevision: 1, Digest: candidate.Digest, Size: candidate.Size, Kind: answers.GroundedKind, Key: "objects/" + contracts.ID(), Owner: contracts.ID()}
		p, e = s.BeginPublication(ctx, p, time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		p.Verification = "sha256-readback"
		if _, e = s.AdmitPublication(ctx, p); e != nil {
			t.Fatal(e)
		}
		if e = s.ArtifactReference(ctx, p.ID, id, false); e != nil {
			t.Fatal(e)
		}
		result, _ := json.Marshal(map[string]any{"candidate": candidate, "accepted": true, "answer_id": candidate.PublicationID})
		return claim, candidate, result
	}
	claim, candidate, result := makeCandidate()
	forged := claim
	forged.Generation++
	if _, err = s.AcceptAnswer(ctx, forged, candidate, result); err == nil {
		t.Fatal("expired generation accepted")
	}
	forgedCandidate := candidate
	forgedCandidate.Digest = strings.Repeat("b", 64)
	if _, err = s.AcceptAnswer(ctx, claim, forgedCandidate, result); err == nil {
		t.Fatal("mismatched bytes accepted")
	}
	accepted, err := s.AcceptAnswer(ctx, claim, candidate, result)
	if err != nil || accepted.State != "succeeded" {
		t.Fatal(accepted, err)
	}
	if _, err = s.CancelWork(ctx, contracts.ID(), claim.ID); err == nil {
		t.Fatal("cancel succeeded after atomic acceptance")
	}
	replay, err := s.AcceptAnswer(ctx, claim, candidate, result)
	if err != nil || replay.JournalReceiptID != accepted.JournalReceiptID {
		t.Fatal("lost response not replayed", err)
	}
	snapshot, err := s.Export(ctx)
	if err != nil {
		t.Fatal("receipt integrity", err)
	}
	restored := localStore(t, snapshot.WorkspaceID)
	if err = restored.Restore(ctx, snapshot); err != nil {
		t.Fatal("answer receipt restore", err)
	}
	retained, err := restored.Work(ctx, claim.ID)
	if err != nil || retained.State != "succeeded" || string(retained.Result) != string(accepted.Result) {
		t.Fatal("restored answer acceptance changed", retained, err)
	}
	cancelled, cc, cr := makeCandidate()
	if _, err = s.CancelWork(ctx, contracts.ID(), cancelled.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcceptAnswer(ctx, cancelled, cc, cr); err == nil {
		t.Fatal("cancelled work accepted")
	}
	stale, sc, sr := makeCandidate()
	if _, err = s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "changed source identity"}}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcceptAnswer(ctx, stale, sc, sr); err == nil {
		t.Fatal("changed source epoch accepted")
	}
}
func TestSQLiteAnswerWorkFence(t *testing.T) { answerWorkSuite(t, localStore(t, contracts.ID())) }
