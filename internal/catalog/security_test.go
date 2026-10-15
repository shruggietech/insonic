package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSQLiteSecurityAndConcurrency(t *testing.T) { securitySuite(t, localStore(t, contracts.ID())) }
func securitySuite(t *testing.T, s *Store) {
	ctx := context.Background()
	t.Run("one transactional CAS winner", func(t *testing.T) {
		rev, _ := s.Revision(ctx)
		var winners atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Settings: []Setting{{Name: "choice", Value: json.RawMessage(`null`)}}})
				if e == nil {
					winners.Add(1)
				}
			}()
		}
		wg.Wait()
		if winners.Load() != 1 {
			t.Fatalf("winners %d", winners.Load())
		}
	})
	t.Run("cross-workspace FK and exact timestamp", func(t *testing.T) {
		other := &Store{db: s.db, workspace: contracts.ID(), backend: s.backend, schema: s.schema}
		if e := other.ensureWorkspace(ctx); e != nil {
			t.Fatal(e)
		}
		artifact := Artifact{ID: contracts.ID(), Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 1, Kind: "source"}
		if _, e := other.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Records: Records{Artifacts: []Artifact{artifact}}}); e != nil {
			t.Fatal(e)
		}
		rev, _ := s.Revision(ctx)
		if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Assets: []Asset{{ID: contracts.ID(), ArtifactID: artifact.ID, Role: "original", DurationUS: 10}}}}); e == nil {
			t.Fatal("workspace escaped")
		}
		if validateRecord(MetadataSnapshot{ID: contracts.ID(), AssetID: contracts.ID(), Extractor: "fixture", Version: "1", State: "failed", Captured: Instant{ISO: "2026-10-05T00:00:00Z", UnixNS: 1}}) == nil {
			t.Fatal("mismatched timestamp")
		}
		if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Settings: []Setting{{Name: "bad", Value: json.RawMessage(`{"password":"fixture-must-not-persist"}`)}}}); e == nil {
			t.Fatal("resolved credential persisted")
		}
		after, _ := s.Revision(ctx)
		if after != rev {
			t.Fatal("rejected admission changed authority")
		}
	})
	t.Run("lease takeover fences old owner", func(t *testing.T) {
		readLease := func(role string) Lease {
			t.Helper()
			got := Lease{Role: role}
			if e := s.db.QueryRowContext(ctx, s.query("SELECT owner_id,generation,lease_until FROM workspace_lease WHERE workspace_id=? AND role=?"), s.workspace, role).Scan(&got.OwnerID, &got.Generation, &got.Until); e != nil {
				t.Fatal(e)
			}
			return got
		}
		setDeadline := func(claim Lease, until int64) Lease {
			t.Helper()
			// Control only this fixture's expiry in the real durable backend.
			// Production claims and renewals retain their actual clocks/TTL limits.
			r, e := s.db.ExecContext(ctx, s.query("UPDATE workspace_lease SET lease_until=? WHERE workspace_id=? AND role=? AND owner_id=? AND generation=? AND lease_until=?"), until, s.workspace, claim.Role, claim.OwnerID, claim.Generation, claim.Until)
			if e != nil {
				t.Fatal(e)
			}
			if changed, e := r.RowsAffected(); e != nil || changed != 1 {
				t.Fatalf("fixture lease update changed %d rows: %v", changed, e)
			}
			claim.Until = until
			if got := readLease(claim.Role); got != claim {
				t.Fatalf("fixture lease differs: got %+v, want %+v", got, claim)
			}
			return claim
		}
		requireConflict := func(e error) {
			t.Helper()
			if typed, ok := e.(*contracts.Error); !ok || typed.Code != "conflict" {
				t.Fatalf("want lease conflict, got %v", e)
			}
		}
		first, e := s.ClaimLease(ctx, "worker", contracts.ID(), time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		// A 100ms real-time fixture could expire during durable I/O on busy CI.
		// Pin live authority explicitly, then expire it deliberately below.
		first = setDeadline(first, math.MaxInt64)
		_, e = s.ClaimLease(ctx, first.Role, contracts.ID(), time.Minute)
		requireConflict(e)
		if got := readLease(first.Role); got != first {
			t.Fatalf("rejected claim changed live authority: %+v", got)
		}
		first = setDeadline(first, 0)
		requireConflict(s.RenewLease(ctx, first, time.Minute))
		next, e := s.ClaimLease(ctx, first.Role, contracts.ID(), time.Minute)
		if e != nil || next.Generation != first.Generation+1 || next.OwnerID == first.OwnerID {
			t.Fatalf("takeover %+v %v", next, e)
		}
		next = setDeadline(next, math.MaxInt64)
		requireConflict(s.RenewLease(ctx, first, time.Minute))
		// Match the new owner so this rejection proves generation fencing alone.
		staleGeneration := next
		staleGeneration.Generation = first.Generation
		requireConflict(s.RenewLease(ctx, staleGeneration, time.Minute))
		if got := readLease(next.Role); got != next {
			t.Fatalf("stale renewal changed current authority: %+v", got)
		}
		if e = s.RenewLease(ctx, next, time.Minute); e != nil {
			t.Fatal(e)
		}
		renewed := readLease(next.Role)
		if renewed.OwnerID != next.OwnerID || renewed.Generation != next.Generation || renewed.Until <= 0 || renewed.Until == next.Until {
			t.Fatalf("valid renewal did not update current lease: %+v", renewed)
		}
	})
	t.Run("restore validation rolls back", func(t *testing.T) {
		snap, e := s.Export(ctx)
		if e != nil {
			t.Fatal(e)
		}
		empty := &Store{db: s.db, workspace: contracts.ID(), backend: s.backend, schema: s.schema}
		if e = empty.ensureWorkspace(ctx); e != nil {
			t.Fatal(e)
		}
		snap.WorkspaceID = empty.workspace
		snap.Records.Assets = append(snap.Records.Assets, Asset{ID: contracts.ID(), ArtifactID: contracts.ID(), Role: "original", DurationUS: 1})
		snap.Digest, _ = snap.digest()
		if e = empty.Restore(ctx, snap); e == nil {
			t.Fatal("invalid restore accepted")
		}
		rev, _ := empty.Revision(ctx)
		if rev != 0 {
			t.Fatal("partial restore published")
		}
		var count int
		if e = empty.db.QueryRowContext(ctx, empty.query("SELECT count(*) FROM operation_receipt WHERE workspace_id=?"), empty.workspace).Scan(&count); e != nil || count != 0 {
			t.Fatal("restore left partial receipts", e)
		}
	})
}
func TestSQLiteConnectionPragmas(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		conn, e := s.db.Conn(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer conn.Close()
		for _, p := range []struct {
			query string
			want  int
		}{{"PRAGMA foreign_keys", 1}, {"PRAGMA synchronous", 2}} {
			var n int
			if e = conn.QueryRowContext(ctx, p.query).Scan(&n); e != nil || n != p.want {
				t.Fatalf("%s %d %v", p.query, n, e)
			}
		}
	}
}
func TestLosslessJSONRejectsAmbiguityAndDamage(t *testing.T) {
	for _, data := range [][]byte{[]byte(`{"revision":1,"revision":2}`), []byte(`{"options":{"same":1,"same":2}}`), {'"', 0xff, '"'}, []byte(`{} {}`)} {
		if ValidateJSON(data) == nil {
			t.Fatal("ambiguous or damaged JSON admitted")
		}
	}
	data := []byte(`{"exact":1791138600123456789}`)
	normalized, e := canonical(data)
	if e != nil || string(normalized) != string(data) {
		t.Fatal("integer precision lost", e)
	}
}
