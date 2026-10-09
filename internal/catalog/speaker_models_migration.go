// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

func (s *Store) hasColumn(ctx context.Context, tx *sql.Tx, table, name string) (bool, error) {
	if s.backend == "postgresql" {
		var count int
		e := s.row(ctx, tx, "SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=? AND column_name=?", table, name).Scan(&count)
		return count != 0, e
	}
	rows, e := tx.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if e != nil {
		return false, e
	}
	defer rows.Close()
	for rows.Next() {
		var seq, notnull, pk int
		var col, kind string
		var def *string
		if e = rows.Scan(&seq, &col, &kind, &notnull, &def, &pk); e != nil {
			return false, e
		}
		if col == name {
			return true, nil
		}
	}
	return false, rows.Err()
}
func (s *Store) migrateSpeakerAuthority(ctx context.Context, tx *sql.Tx) error {
	for _, col := range []struct{ name, ddl string }{{"origin", "TEXT NOT NULL DEFAULT 'manual'"}, {"provenance", "TEXT NOT NULL DEFAULT '{}'"}} {
		has, e := s.hasColumn(ctx, tx, "speaker_mapping", col.name)
		if e != nil {
			return e
		}
		if !has {
			if _, e = tx.ExecContext(ctx, "ALTER TABLE speaker_mapping ADD COLUMN "+col.name+" "+col.ddl); e != nil {
				return e
			}
		}
	}
	// Legacy job lineage remains readable; new training binds durable work directly.
	if s.backend == "postgresql" {
		_, e := tx.ExecContext(ctx, "ALTER TABLE training_run DROP CONSTRAINT IF EXISTS training_run_workspace_id_job_id_fkey")
		return e
	}
	d := domainNamed("Runs")
	q := strings.Replace(d.ddl(), "training_run (", "training_run_new (", 1)
	if _, e := tx.ExecContext(ctx, q); e != nil {
		return e
	}
	cols := "workspace_id," + joinNames(d)
	for _, q := range []string{"INSERT INTO training_run_new(" + cols + ") SELECT " + cols + " FROM training_run", "DROP TABLE training_run", "ALTER TABLE training_run_new RENAME TO training_run"} {
		if _, e := tx.ExecContext(ctx, q); e != nil {
			return e
		}
	}
	rows, e := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if e != nil {
		return e
	}
	defer rows.Close()
	if rows.Next() {
		return contracts.Fail("invalid_request")
	}
	return rows.Err()
}
func normalizeHistoricalMappings(records *Records) error {
	for i := range records.SpeakerMappings {
		m := &records.SpeakerMappings[i]
		if m.Origin != "" && m.Origin != "manual" || len(m.Provenance) > 0 && string(m.Provenance) != "{}" {
			return contracts.Fail("invalid_request")
		}
		m.Origin = "manual"
		m.Provenance = json.RawMessage(`{}`)
	}
	return nil
}
