// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"strings"
)

// Opening migration holds a dedicated connection. SQLite's standard table
// rebuild runs with foreign keys disabled, then checks the complete graph before
// commit; ordinary connections retain foreign key enforcement.
func (s *Store) migrateRecordingState(ctx context.Context, tx *sql.Tx) error {
	var d domainTable
	for _, v := range domains {
		if v.field == "Recordings" {
			d = v
		}
	}
	if s.backend == "postgresql" {
		if _, e := tx.ExecContext(ctx, "ALTER TABLE current_recording DROP CONSTRAINT current_recording_state_check"); e != nil {
			return e
		}
		_, e := tx.ExecContext(ctx, "ALTER TABLE current_recording ADD CONSTRAINT current_recording_state_check CHECK (state IN ('ready','no-speech','no-timed-subtitles','untranscribed'))")
		return e
	}
	q := strings.Replace(d.ddl(), "current_recording (", "current_recording_new (", 1)
	if _, e := tx.ExecContext(ctx, q); e != nil {
		return e
	}
	cols := "workspace_id," + joinNames(d)
	for _, q := range []string{"INSERT INTO current_recording_new(" + cols + ") SELECT " + cols + " FROM current_recording", "DROP TABLE current_recording", "ALTER TABLE current_recording_new RENAME TO current_recording"} {
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
		return sql.ErrNoRows
	}
	return rows.Err()
}
