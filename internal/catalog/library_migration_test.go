package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"testing"
)

func TestActualHistoricalCatalogUpgrades(t *testing.T) {
	for _, version := range []int{1, 2, 3, 4} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "historical.sqlite")
			db, e := sql.Open("sqlite3", path+"?_foreign_keys=on")
			if e != nil {
				t.Fatal(e)
			}
			ddl := legacyMigrationStatements()
			digest := legacyMigrationDigest()
			if version == 2 {
				ddl = historicalV2Statements()
				digest = historicalV2Digest()
			}
			if version == 3 {
				ddl = historicalV3DDL
				digest = historicalV3Digest()
			}
			if version == 4 {
				ddl = historicalV4DDL
				digest = historicalV4Digest()
			}
			for _, q := range ddl {
				if _, e = db.Exec(q); e != nil {
					t.Fatal(e)
				}
			}
			_, e = db.Exec("CREATE TABLE catalog_schema(singleton BIGINT PRIMARY KEY,version BIGINT NOT NULL,digest TEXT NOT NULL)")
			if e != nil {
				t.Fatal(e)
			}
			_, e = db.Exec("INSERT INTO catalog_schema VALUES(1,?,?)", version, digest)
			if e != nil {
				t.Fatal(e)
			}
			wid, aid, sid := contracts.ID(), contracts.ID(), contracts.ID()
			_, e = db.Exec("INSERT INTO workspace VALUES(?,0,?)", wid, version)
			if e != nil {
				t.Fatal(e)
			}
			_, e = db.Exec("INSERT INTO artifact VALUES(?,?,?,?,?)", wid, aid, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 12, "source")
			if e != nil {
				t.Fatal(e)
			}
			_, e = db.Exec("INSERT INTO media_asset VALUES(?,?,?,?,?)", wid, sid, aid, "original", int64(0))
			if e != nil {
				t.Fatal(e)
			}
			speaker := contracts.ID()
			if _, e = db.Exec("INSERT INTO speaker VALUES(?,?,?)", wid, speaker, "Historical name"); e != nil {
				t.Fatal(e)
			}
			db.Close()
			s, e := OpenSQLite(ctx, path, wid)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			snap, e := s.Export(ctx)
			if e != nil {
				t.Fatal(e)
			}
			if len(snap.Records.Assets) != 1 || snap.Records.Assets[0].DurationUS != 0 || snap.CatalogSchema != SchemaVersion {
				t.Fatal("upgrade lost source timing")
			}
			identity, e := s.Speaker(ctx, speaker)
			if e != nil || identity.Speaker.State != "active" || identity.Speaker.Revision < 1 {
				t.Fatal("identity migration", e)
			}
			if e = localStore(t, wid).Restore(ctx, snap); e != nil {
				t.Fatal("upgraded snapshot proof", e)
			}
			var actual string
			if e = s.db.QueryRow("SELECT digest FROM catalog_schema").Scan(&actual); e != nil || actual != migrationDigest() {
				t.Fatal("wrong upgrade identity", e)
			}
		})
	}
}
func TestHistoricalSnapshotSerialization(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.CatalogSchema = 2
	snap.Digest, _ = snap.digest()
	raw, e := json.Marshal(snap)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := ReadSnapshot(raw)
	if e != nil {
		t.Fatal(e)
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, decoded); e != nil {
		t.Fatal("historical snapshot lost integrity", e)
	}
}
