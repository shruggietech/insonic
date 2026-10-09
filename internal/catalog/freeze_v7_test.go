// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestFreezeV7(t *testing.T) {
	current := []string{}
	for _, q := range migrationStatements() {
		if strings.Contains(q, "CREATE TABLE IF NOT EXISTS model_alias ") || strings.Contains(q, "CREATE TABLE IF NOT EXISTS model_source ") {
			continue
		}
		current = append(current, q)
	}
	if strings.Join(current, "\n") != strings.Join(historicalV7DDL, "\n") {
		t.Fatal("published schema7 DDL drift")
	}
}
func TestPopulatedV7ModelMigrationAndSnapshot(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, rec, _ := evidenceRecording(t, s)
	snapshot, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snapshot.CatalogSchema = 7
	snapshot.Digest, _ = snapshot.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snapshot); e != nil {
		t.Fatal("schema7 snapshot", e)
	}
	if _, e = s.db.ExecContext(ctx, "DROP TABLE model_alias"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.ExecContext(ctx, "DROP TABLE model_source"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.ExecContext(ctx, "UPDATE catalog_schema SET version=7,digest=?", historicalV7Digest()); e != nil {
		t.Fatal(e)
	}
	if e = s.migrate(ctx); e != nil {
		t.Fatal(e)
	}
	got, e := s.Recording(ctx, rec.ID)
	if e != nil || got.DocumentDigest != rec.DocumentDigest {
		t.Fatal("lost current authority", e)
	}
	snapshot, e = s.Export(ctx)
	if e != nil || snapshot.CatalogSchema != 8 {
		t.Fatal("schema8", e)
	}
}
