// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"testing"
)

func TestLegacyArtifactUpgradeAndSnapshot(t *testing.T) {
	ctx := context.Background()
	id := contracts.ID()
	path := filepath.Join(t.TempDir(), "old.sqlite")
	s, e := OpenSQLite(ctx, path, id)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Settings: []Setting{{Name: "preserved", Value: json.RawMessage(`true`)}}})
	if e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.CatalogSchema = 1
	snap.State = snap.State[:len(snap.State)-1]
	snap.Digest, _ = snap.digest()
	if _, e = s.db.Exec("DROP TABLE artifact_publication"); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"ALTER TABLE speaker DROP COLUMN revision", "ALTER TABLE speaker DROP COLUMN state"} {
		if _, e = s.db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = s.db.Exec("UPDATE catalog_schema SET version=1,digest=?", legacyMigrationDigest()); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = OpenSQLite(ctx, path, id)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	settings, e := s.Settings(ctx)
	if e != nil || len(settings) != 1 {
		t.Fatalf("upgrade lost settings %v", e)
	}
	dst, e := OpenSQLite(ctx, filepath.Join(t.TempDir(), "new.sqlite"), id)
	if e != nil {
		t.Fatal(e)
	}
	defer dst.Close()
	if e = dst.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
}
