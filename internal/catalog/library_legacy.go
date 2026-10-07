// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"strings"
)

// removeLegacyMetadata consumes the old append-only representation when a
// current library record is published. Owner literals survive without an obsolete
// extractor association. Retirement work contains only managed publication IDs.
func (s *Store) removeLegacyMetadata(ctx context.Context, tx *sql.Tx, entry LibraryEntry, revision int64) error {
	rows, e := tx.QueryContext(ctx, s.query("SELECT p.id FROM metadata_snapshot m JOIN artifact_publication p ON p.workspace_id=m.workspace_id AND p.artifact_id=m.report_artifact_id WHERE m.workspace_id=? AND m.asset_id=?"), s.workspace, entry.AssetID)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	for _, id := range ids {
		if e = s.putDomain(ctx, tx, "Cleanups", Cleanup{ID: id, EntryID: entry.ID, State: "pending", Revision: revision}); e != nil {
			return e
		}
	}
	obs := "SELECT o.id FROM metadata_observation o JOIN metadata_snapshot m ON m.workspace_id=o.workspace_id AND m.id=o.snapshot_id WHERE m.workspace_id=? AND m.asset_id=?"
	for _, q := range []string{
		"UPDATE date_observation SET observation_id=NULL WHERE workspace_id=? AND basis='owner' AND observation_id IN (" + obs + ")",
		"DELETE FROM date_selection WHERE workspace_id=? AND date_id IN (SELECT id FROM date_observation WHERE workspace_id=? AND observation_id IN (" + obs + "))",
		"DELETE FROM date_observation WHERE workspace_id=? AND observation_id IN (" + obs + ")",
	} {
		args := []any{s.workspace}
		if strings.HasPrefix(q, "DELETE FROM date_selection ") {
			args = append(args, s.workspace)
		}
		args = append(args, s.workspace, entry.AssetID)
		if _, e = s.exec(ctx, tx, q, args...); e != nil {
			return e
		}
	}
	if _, e = s.exec(ctx, tx, "DELETE FROM metadata_observation WHERE workspace_id=? AND snapshot_id IN (SELECT id FROM metadata_snapshot WHERE workspace_id=? AND asset_id=?)", s.workspace, s.workspace, entry.AssetID); e != nil {
		return e
	}
	_, e = s.exec(ctx, tx, "DELETE FROM metadata_snapshot WHERE workspace_id=? AND asset_id=?", s.workspace, entry.AssetID)
	return e
}
