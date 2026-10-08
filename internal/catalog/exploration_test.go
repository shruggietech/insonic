package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestSavedQueryImmutableCASRestartAndProof(t *testing.T) {
	savedQuerySuite(t, localStore)
}
func savedQuerySuite(t *testing.T, create func(*testing.T, string) *Store) {
	ctx := context.Background()
	s := create(t, contracts.ID())
	id := contracts.ID()
	q, e := s.PutSavedQuery(ctx, contracts.ID(), 0, SavedQuery{ID: id, Definition: json.RawMessage(`{"title":"First","definition":{"mode":"normalized","operation":"text-search","filters":{"text":"source"}}}`)})
	if e != nil {
		t.Fatal(e)
	}
	old := q
	replayID := contracts.ID()
	replayed, e := s.PutSavedQuery(ctx, replayID, 1, SavedQuery{ID: id, Definition: q.Definition})
	if e != nil {
		t.Fatal(e)
	}
	replayed.Validations = json.RawMessage(`[{"target_profile":{"profile_id":"00000000-0000-4000-8000-000000000001","profile_revision":1},"adapter_id":"ladybugdb","catalog_schema_version":"6","projection_schema_version":"1","status":"pending"}]`)
	replay, e := s.PutSavedQuery(ctx, replayID, 1, replayed)
	if e != nil || string(replay.Validations) != "[]" {
		t.Fatal("compatibility changed idempotent result", replay, e)
	}
	q = replay
	q.Definition = json.RawMessage(`{"title":"Second","definition":{"mode":"normalized","operation":"media-list"}}`)
	q, e = s.PutSavedQuery(ctx, contracts.ID(), q.Revision, q)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutSavedQuery(ctx, contracts.ID(), old.Revision, old); e == nil {
		t.Fatal("stale update accepted")
	}
	history, e := s.SavedQueries(ctx, id)
	if e != nil || len(history) != 3 {
		t.Fatal(history, e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	layoutID := contracts.ID()
	layout, e := s.PutGraphLayout(ctx, contracts.ID(), 0, GraphLayout{ID: layoutID, QueryID: id, Layout: json.RawMessage(`{"positions":{"node":{"x":10,"y":20}},"paused":true,"filter":"","limit":200}`)})
	if e != nil {
		t.Fatal(e)
	}
	snap, e = s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restored := create(t, s.workspace)
	if e = restored.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	history, e = restored.SavedQueries(ctx, id)
	if e != nil || len(history) != 3 {
		t.Fatal(history, e)
	}
	persisted, e := restored.GraphLayout(ctx, layoutID)
	if e != nil || persisted.Revision != layout.Revision {
		t.Fatal(persisted, e)
	}
	if _, e = restored.PutGraphLayout(ctx, contracts.ID(), 0, layout); e == nil {
		t.Fatal("stale layout accepted")
	}
	snap.Records.SavedQueries[0].Definition = json.RawMessage(`{"title":"Forged","definition":{"mode":"normalized","operation":"media-list"}}`)
	snap.Digest, _ = snap.digest()
	if e = create(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("forged saved definition restored")
	}
}
func TestSavedQueryRejectsProcessingCopies(t *testing.T) {
	s := localStore(t, contracts.ID())
	for _, raw := range []string{`{"title":"x","definition":{"mode":"normalized","operation":"media-list"},"document":{}}`, `{"title":"x","definition":{"mode":"normalized","operation":"media-list"},"pinned_result":"old"}`} {
		if _, e := s.PutSavedQuery(context.Background(), contracts.ID(), 0, SavedQuery{ID: contracts.ID(), Definition: json.RawMessage(raw)}); e == nil {
			t.Fatal("processing snapshot saved")
		}
	}
}

func TestRebuildRequestReplayDoesNotAppendNewEvent(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	op := contracts.ID()
	first, e := s.EnqueueGraphRebuild(ctx, op)
	if e != nil {
		t.Fatal(e)
	}
	second, e := s.EnqueueGraphRebuild(ctx, op)
	if e != nil || second.Revision != first.Revision {
		t.Fatal(second, e)
	}
	status, e := s.GraphStatus(ctx)
	if e != nil || status["pending_events"] != 1 {
		t.Fatal(status, e)
	}
}
