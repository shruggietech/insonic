package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
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
		first, e := s.ClaimLease(ctx, "worker", contracts.ID(), 100*time.Millisecond)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.ClaimLease(ctx, "worker", contracts.ID(), time.Second); e == nil {
			t.Fatal("live owner stolen")
		}
		time.Sleep(150 * time.Millisecond)
		next, e := s.ClaimLease(ctx, "worker", contracts.ID(), time.Second)
		if e != nil || next.Generation != first.Generation+1 {
			t.Fatalf("takeover %+v %v", next, e)
		}
		if e = s.RenewLease(ctx, first, time.Second); e == nil {
			t.Fatal("obsolete lease renewed")
		}
		if e = s.RenewLease(ctx, next, time.Second); e != nil {
			t.Fatal(e)
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
