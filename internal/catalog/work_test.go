package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func workSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	op := contracts.ID()
	w, e := s.EnqueueWork(ctx, op, "media.import", json.RawMessage(`{"source":"fixture.wav"}`))
	if e != nil {
		t.Fatal(e)
	}
	same, e := s.EnqueueWork(ctx, op, "media.import", w.Payload)
	if e != nil || same.ID != w.ID {
		t.Fatal("work replay", e)
	}
	if _, e = s.EnqueueWork(ctx, op, "media.import", json.RawMessage(`{"source":"changed.wav"}`)); e == nil {
		t.Fatal("changed intent accepted")
	}
	claim, e := s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CancelWork(ctx, contracts.ID(), w.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CheckpointWork(ctx, claim, "complete", "succeeded", json.RawMessage(`{"accepted":true}`), time.Minute); e == nil {
		t.Fatal("cancelled claim accepted")
	}
	if _, e = s.CommitLibrary(ctx, claim, LibraryEntry{}); e == nil {
		t.Fatal("stale library admission")
	}
	if _, e = s.RetryWork(ctx, contracts.ID(), w.ID); e != nil {
		t.Fatal(e)
	}
	next, e := s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil || next.Generation <= claim.Generation {
		t.Fatal("retry generation", e)
	}
	if _, e = s.CheckpointWork(ctx, next, "complete", "succeeded", json.RawMessage(`{"accepted":true}`), time.Minute); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	got, e := dest.Work(ctx, w.ID)
	if e != nil || got.State != "succeeded" || got.LeaseUntil != 0 {
		t.Fatal("portable work", got, e)
	}
}
func TestSQLiteWorkAuthority(t *testing.T) { workSuite(t, localStore(t, contracts.ID())) }
func TestHistoricalDDLIdentity(t *testing.T) {
	if len(historicalV1DDL) != 27 {
		t.Fatal("historical DDL altered")
	}
	if historicalV1DDL[12] != "CREATE TABLE IF NOT EXISTS media_asset (workspace_id TEXT NOT NULL,id TEXT NOT NULL,artifact_id TEXT NOT NULL,role TEXT NOT NULL,duration_us BIGINT NOT NULL,PRIMARY KEY (workspace_id,id),FOREIGN KEY (workspace_id) REFERENCES workspace(id),FOREIGN KEY (workspace_id,artifact_id) REFERENCES artifact(workspace_id,id),CHECK (duration_us>=0))" {
		t.Fatal("historical clock altered")
	}
}

func TestShutdownInterruptsRealWork(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	w, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"source":"fixture"}`))
	if e != nil {
		t.Fatal(e)
	}
	owner := contracts.ID()
	claim, e := s.ClaimWork(ctx, w.ID, owner, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.InterruptOwner(ctx, owner); e != nil {
		t.Fatal(e)
	}
	interrupted, e := s.Work(ctx, w.ID)
	if e != nil || interrupted.State != "interrupted" || interrupted.LeaseUntil != 0 || interrupted.Owner != "" {
		t.Fatal("real shutdown retained authority", interrupted, e)
	}
	next, e := s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil || next.Generation != claim.Generation+1 {
		t.Fatal("shutdown recovery delayed", e)
	}
	if _, e = s.CheckpointWork(ctx, claim, "old-completion", "succeeded", json.RawMessage(`null`), time.Minute); e == nil {
		t.Fatal("interrupted old worker completed")
	}
}

func TestLargeBatchReferenceResultsRemainPortable(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	work, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if e != nil {
		t.Fatal(e)
	}
	claim, e := s.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	results := make([]map[string]any, 10000)
	for i := range results {
		results[i] = map[string]any{"index": i, "entry_id": contracts.ID(), "asset_id": contracts.ID(), "report_publication_id": contracts.ID(), "state": "accepted"}
	}
	raw, e := json.Marshal(map[string]any{"items": results})
	if e != nil {
		t.Fatal(e)
	}
	if len(raw) <= 16384 || len(raw) >= 4<<20 {
		t.Fatal("fixture does not exercise batch journal bounds", len(raw))
	}
	if _, e = s.CheckpointWork(ctx, claim, "complete", "succeeded", raw, time.Minute); e != nil {
		t.Fatal("valid 10000-item result rejected", e)
	}
	snapshot, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, snapshot); e != nil {
		t.Fatal("large result snapshot rejected", e)
	}
	got, e := dest.Work(ctx, work.ID)
	if e != nil || string(got.Result) != string(raw) {
		t.Fatal("large result lost through restore", e)
	}
	results[50]["metadata"] = map[string]any{"title": "superseded capture"}
	raw, _ = json.Marshal(map[string]any{"items": results})
	if referenceResult(raw) {
		t.Fatal("large journal admitted archived metadata")
	}
}

func TestWorkMutationsReturnCurrentJournalProof(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	w, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	assertCurrent := func(w Work) {
		t.Helper()
		current, e := s.Work(ctx, w.ID)
		if e != nil || current.JournalReceiptID != w.JournalReceiptID {
			t.Fatal("returned work carries superseded journal proof", e)
		}
	}
	assertCurrent(w)
	w, e = s.RenewWork(ctx, w, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	assertCurrent(w)
	w, e = s.CheckpointWork(ctx, w, "complete", "succeeded", json.RawMessage(`{"accepted":true}`), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	assertCurrent(w)
}
