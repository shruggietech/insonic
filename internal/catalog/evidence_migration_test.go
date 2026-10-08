// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestPopulatedSchema3EvidenceMigration(t *testing.T) {
	historicalEvidenceMigrationSuite(t, localStore(t, contracts.ID()))
}
func historicalEvidenceMigrationSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	source, manifest, preparation, clip, weights := availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s)
	job, e := s.StartJob(ctx, contracts.ID(), contracts.ID(), 10, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"ALTER TABLE speaker DROP COLUMN revision", "ALTER TABLE speaker DROP COLUMN state"} {
		if _, e = s.db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	for _, table := range []string{"graph_layout", "saved_query", "current_extraction", "model_artifact", "model_version", "training_run", "dataset_member", "training_dataset", "speaker_segment", "speaker_mapping", "current_recording", "library_cleanup"} {
		if _, e = s.db.Exec("DROP TABLE " + table); e != nil {
			t.Fatal(e)
		}
	}
	for _, ddl := range historicalV3DDL {
		for _, table := range []string{"speaker_segment", "training_dataset", "dataset_member", "training_run", "model_version", "model_artifact", "library_cleanup"} {
			if strings.HasPrefix(ddl, "CREATE TABLE IF NOT EXISTS "+table+" ") {
				if _, e = s.db.Exec(ddl); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
	asset, speaker, segment, dataset, run, model, version := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	for _, row := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO media_asset VALUES(?,?,?,?,?)", []any{s.workspace, asset, source.ArtifactID, "original", int64(1000000)}},
		{"INSERT INTO speaker VALUES(?,?,?)", []any{s.workspace, speaker, "Known"}},
		{"INSERT INTO speaker_segment VALUES(?,?,?,?,?,?,?,?,?,?)", []any{s.workspace, segment, int64(1), asset, speaker, int64(0), int64(1000000), int64(0), `{"turns":["legacy-assignment-marker"]}`, clip.ArtifactID}},
		{"INSERT INTO training_dataset VALUES(?,?,?,?,?)", []any{s.workspace, dataset, speaker, manifest.ArtifactID, `{"speaker_attributions":["legacy-assignment-marker"]}`}},
		{"INSERT INTO dataset_member VALUES(?,?,?,?,?,?)", []any{s.workspace, contracts.ID(), dataset, int64(0), segment, int64(1)}},
		{"INSERT INTO training_run VALUES(?,?,?,?,?,?,?,?)", []any{s.workspace, run, dataset, speaker, job.JobID, preparation.ArtifactID, "fixture", `{"turns":["legacy-assignment-marker"]}`}},
		{"INSERT INTO speaker_model VALUES(?,?,?,?)", []any{s.workspace, model, speaker, "Family"}},
		{"INSERT INTO model_version VALUES(?,?,?,?,?,?,?)", []any{s.workspace, version, model, run, dataset, manifest.ArtifactID, "trained"}},
		{"INSERT INTO model_artifact VALUES(?,?,?,?,?,?)", []any{s.workspace, contracts.ID(), version, weights.ArtifactID, "weights", "fixture"}},
	} {
		if _, e = s.db.Exec(s.query(row.sql), row.args...); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = s.db.Exec(s.query("UPDATE catalog_schema SET version=3,digest=?"), historicalV3ExpectedDigest); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE workspace SET schema_version=3"); e != nil {
		t.Fatal(e)
	}
	if e = s.migrate(ctx); e != nil {
		t.Fatal("actual historical schema3 upgrade", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(snap)
	if strings.Contains(string(raw), "legacy-assignment-marker") || len(snap.Records.Segments) != 0 || len(snap.Records.Members) != 0 {
		t.Fatal("legacy assignment copy survived upgrade")
	}
	if len(snap.Records.Datasets) != 1 || snap.Records.Datasets[0].State != "invalidated" || snap.Records.Datasets[0].ManifestArtifactID != nil || snap.Records.Runs[0].PreparationArtifactID != nil || snap.Records.Versions[0].ManifestArtifactID != nil {
		t.Fatal("lineage not explicitly invalidated")
	}
	if len(snap.Records.Assets) != 1 || snap.Records.Assets[0].ArtifactID != source.ArtifactID || len(snap.Records.ModelArtifacts) != 1 || snap.Records.ModelArtifacts[0].ArtifactID != weights.ArtifactID {
		t.Fatal("source or weights lost")
	}
	if len(snap.Records.Cleanups) != 3 {
		t.Fatal("derived cleanup obligations missing", snap.Records.Cleanups)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("upgraded state not portable", e)
	}
}
