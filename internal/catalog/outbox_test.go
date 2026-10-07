package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func TestSQLiteOrderedOutbox(t *testing.T) { outboxSuite(t, localStore(t, contracts.ID())) }

// Advance the stored lease to an expired state without depending on runner
// scheduling between separate durable transactions. The production clock and
// acknowledgement fences remain in use.
func expireOutboxLease(t *testing.T, s *Store, claim OutboxClaim) {
	t.Helper()
	err := s.write(context.Background(), func(tx *sql.Tx, _ int64) error {
		result, err := s.exec(context.Background(), tx, "UPDATE graph_target SET lease_until=0 WHERE workspace_id=? AND id=? AND owner_id=? AND generation=?", claim.WorkspaceID, claim.Target, claim.OwnerID, claim.Generation)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err == nil && rows != 1 {
			t.Fatalf("lease transition affected %d targets", rows)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
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
	first, e := s.ClaimOutbox(ctx, target, owner, time.Minute)
	if e != nil || first.Sequence != 1 || first.Predecessor != 0 {
		t.Fatalf("claim %+v %v", first, e)
	}
	if _, e = s.ClaimOutbox(ctx, target, contracts.ID(), time.Minute); e == nil {
		t.Fatal("live claim stolen")
	}
	expireOutboxLease(t, s, first)
	second, e := s.ClaimOutbox(ctx, target, contracts.ID(), time.Minute)
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
	next, e := s.ClaimOutbox(ctx, target, second.OwnerID, time.Minute)
	if e != nil || next.Sequence != 2 || next.Predecessor != 1 {
		t.Fatalf("next %+v %v", next, e)
	}
}
