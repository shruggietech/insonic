// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestPublishedV8FreezeAndPopulatedMigration(t *testing.T) {
	if historicalV8Digest() != "4f4420d0e9944f00cde24302dc142c07d5cd203ea7826bb555863a3885bc0dbb" {
		t.Fatal("published schema8 DDL changed")
	}
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, rec, voice := evidenceRecording(t, s)
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Historical owner correction"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), rec.Revision, SpeakerMapping{RecordingID: rec.ID, LocalSpeakerID: voice, SpeakerID: sp.Speaker.ID, DocumentDigest: rec.DocumentDigest}); e != nil {
		t.Fatal(e)
	}
	maps, e := s.SpeakerMappings(ctx, rec.ID)
	if e != nil {
		t.Fatal(e)
	}
	legacyDigest := legacyMappingDigest(maps)
	conn, e := s.db.Conn(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	if _, e = conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); e != nil {
		t.Fatal(e)
	}
	for _, q := range historicalV8DDL {
		if strings.HasPrefix(q, "CREATE TABLE IF NOT EXISTS speaker_mapping (") {
			q = strings.Replace(q, "speaker_mapping (", "speaker_mapping_v8 (", 1)
			if _, e = conn.ExecContext(ctx, q); e != nil {
				t.Fatal(e)
			}
		}
	}
	for _, q := range []string{"INSERT INTO speaker_mapping_v8(workspace_id,id,recording_id,local_speaker_id,speaker_id,document_digest,revision) SELECT workspace_id,id,recording_id,local_speaker_id,speaker_id,document_digest,revision FROM speaker_mapping", "DROP TABLE speaker_mapping", "ALTER TABLE speaker_mapping_v8 RENAME TO speaker_mapping", "DROP TABLE speaker_output", "DROP TABLE speaker_profile", "DROP TABLE speaker_checkpoint"} {
		if _, e = conn.ExecContext(ctx, q); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := conn.QueryContext(ctx, "SELECT id,result FROM operation_receipt WHERE workspace_id=?", s.workspace)
	if e != nil {
		t.Fatal(e)
	}
	updates := map[string]string{}
	for rows.Next() {
		var id, raw string
		if e = rows.Scan(&id, &raw); e != nil {
			t.Fatal(e)
		}
		var result map[string]json.RawMessage
		json.Unmarshal([]byte(raw), &result)
		var mapped string
		json.Unmarshal(result["mapping_recording_id"], &mapped)
		if mapped == rec.ID {
			result["mapping_digest"], _ = json.Marshal(legacyDigest)
			data, _ := json.Marshal(result)
			updates[id] = string(data)
		}
	}
	rows.Close()
	for id, raw := range updates {
		if _, e = conn.ExecContext(ctx, "UPDATE operation_receipt SET result=? WHERE workspace_id=? AND id=?", raw, s.workspace, id); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = conn.ExecContext(ctx, "UPDATE catalog_schema SET version=8,digest=?", historicalV8Digest()); e != nil {
		t.Fatal(e)
	}
	if _, e = conn.ExecContext(ctx, "UPDATE workspace SET schema_version=8"); e != nil {
		t.Fatal(e)
	}
	if _, e = conn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); e != nil {
		t.Fatal(e)
	}
	conn.Close()
	if e = s.migrate(ctx); e != nil {
		t.Fatal(e)
	}
	maps, e = s.SpeakerMappings(ctx, rec.ID)
	if e != nil || len(maps) != 1 || maps[0].Origin != "manual" {
		t.Fatal("historical mapping did not retain independent authority", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("migrated historical proofs", e)
	}
}
