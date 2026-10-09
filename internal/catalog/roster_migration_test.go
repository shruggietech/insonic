// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestPublishedV6DDLAndMigration(t *testing.T) {
	current := []string{}
	for _, q := range migrationStatements() {
		if strings.Contains(q, "CREATE TABLE IF NOT EXISTS recording_roster") || strings.Contains(q, "CREATE TABLE IF NOT EXISTS roster_member") {
			continue
		}
		current = append(current, strings.ReplaceAll(q, "'no-timed-subtitles','untranscribed'", "'no-timed-subtitles'"))
	}
	if strings.Join(current, "\n") != strings.Join(historicalV6DDL, "\n") {
		t.Fatal("published schema 6 DDL differs")
	}
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, recording, local := evidenceRecording(t, s)
	speaker, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Retained"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, SpeakerMapping{RecordingID: recording.ID, LocalSpeakerID: local, SpeakerID: speaker.Speaker.ID, DocumentDigest: recording.DocumentDigest}); e != nil {
		t.Fatal(e)
	}
	rev, _ := s.Revision(ctx)
	segment := Segment{ID: contracts.ID(), Revision: 1, RecordingID: recording.ID, DocumentDigest: recording.DocumentDigest, CueID: "cue-000000", LocalSpeakerID: local}
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Segments: []Segment{segment}}}); e != nil {
		t.Fatal(e)
	}
	// Construct an actual populated schema-6 table with its published constraint,
	// rather than changing only the version label on a schema-7 table.
	conn, e := s.db.Conn(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); e != nil {
		t.Fatal(e)
	}
	var ddl string
	for _, q := range historicalV6DDL {
		if strings.HasPrefix(q, "CREATE TABLE IF NOT EXISTS current_recording (") {
			ddl = strings.Replace(q, "current_recording (", "recording_v6 (", 1)
		}
	}
	for _, q := range []string{ddl, "INSERT INTO recording_v6 SELECT * FROM current_recording", "DROP TABLE current_recording", "ALTER TABLE recording_v6 RENAME TO current_recording", "DROP TABLE roster_member", "DROP TABLE recording_roster"} {
		if _, e = conn.ExecContext(ctx, q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = conn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); e != nil {
		t.Fatal(e)
	}
	conn.Close()
	if _, e := s.db.ExecContext(ctx, "UPDATE catalog_schema SET version=6,digest=?", historicalV6Digest()); e != nil {
		t.Fatal(e)
	}
	if e := s.migrate(ctx); e != nil {
		t.Fatal(e)
	}
	snapshot, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(snapshot.Records.Segments) != 1 || len(snapshot.Records.SpeakerMappings) != 1 || !json.Valid(snapshot.Records.Recordings[0].Document) {
		t.Fatal("populated references lost on migration")
	}
}
