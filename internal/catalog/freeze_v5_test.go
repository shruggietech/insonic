package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrozenV5AndExistingDatabaseMigration(t *testing.T) {
	if hash([]byte(strings.Join(historicalV5DDL, "\n"))) != historicalV5Digest() {
		t.Fatal("historical v5 DDL changed")
	}
	id := contracts.ID()
	path := filepath.Join(t.TempDir(), "v5.sqlite")
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range historicalV5DDL {
		if _, e = db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = db.Exec("CREATE TABLE catalog_schema(singleton BIGINT PRIMARY KEY,version BIGINT,digest TEXT); INSERT INTO catalog_schema VALUES(1,5,?)", historicalV5Digest()); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("INSERT INTO workspace VALUES(?,0,5)", id); e != nil {
		t.Fatal(e)
	}
	db.Close()
	s, e := OpenSQLite(context.Background(), path, id)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	snap, e := s.Export(context.Background())
	if e != nil || snap.CatalogSchema != 6 {
		t.Fatal(snap.CatalogSchema, e)
	}
}
func TestV5SnapshotUpgradeAndDowngradeRejection(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.CatalogSchema = 5
	snap.Digest, _ = snap.digest()
	raw, _ := json.Marshal(snap)
	up, e := ReadSnapshot(raw)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, up); e != nil {
		t.Fatal(e)
	}
}

func TestV5RejectsNewExplorationProofsWithoutRows(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	_, e := s.PutSavedQuery(ctx, contracts.ID(), 0, SavedQuery{ID: contracts.ID(), Definition: json.RawMessage(`{"title":"Current","definition":{"mode":"normalized","operation":"media-list"}}`)})
	if e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.CatalogSchema = 5
	snap.Records.SavedQueries = nil
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("schema5 accepted schema6 proofs")
	}
}
