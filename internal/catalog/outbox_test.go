package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func TestSQLiteOrderedOutbox(t *testing.T) { outboxSuite(t, localStore(t, contracts.ID())) }
func outboxSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	target := contracts.ID()
	owner := contracts.ID()
	for i := 0; i < 2; i++ {
		rev, _ := s.Revision(ctx)
		if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Projections: []Projection{{Target: target, Document: json.RawMessage(`{"accepted":true}`)}}}); e != nil {
			t.Fatal(e)
		}
	}
	first, e := s.ClaimOutbox(ctx, target, owner, time.Second)
	if e != nil || first.Sequence != 1 || first.Predecessor != 0 {
		t.Fatalf("claim %+v %v", first, e)
	}
	if _, e = s.ClaimOutbox(ctx, target, contracts.ID(), time.Second); e == nil {
		t.Fatal("live claim stolen")
	}
	time.Sleep(1100 * time.Millisecond)
	second, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Second)
	if e != nil || second.Sequence != 1 || second.Generation <= first.Generation {
		t.Fatalf("takeover %+v %v", second, e)
	}
	if e = s.AcknowledgeOutbox(ctx, first); e == nil {
		t.Fatal("stale acknowledgement")
	}
	tampered := second
	tampered.Digest = "bad"
	if e = s.AcknowledgeOutbox(ctx, tampered); e == nil {
		t.Fatal("wrong event acknowledged")
	}
	if e = s.AcknowledgeOutbox(ctx, second); e != nil {
		t.Fatal(e)
	}
	if e = s.AcknowledgeOutbox(ctx, second); e != nil {
		t.Fatal("lost ack reconciliation", e)
	}
	next, e := s.ClaimOutbox(ctx, target, second.OwnerID, time.Second)
	if e != nil || next.Sequence != 2 || next.Predecessor != 1 {
		t.Fatalf("next %+v %v", next, e)
	}
}
