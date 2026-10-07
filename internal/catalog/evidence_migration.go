// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"strings"
)

type historicalEvidenceRows struct {
	table, columns string
	rows           [][]any
}
type historicalEvidence struct {
	lineage   []historicalEvidenceRows
	artifacts map[string][]string
}

// Read only noncontent lineage before replacing the old FK graph. Old intervals,
// attribution, memberships and assignment-bearing options are deliberately not
// copied into either the new tables or an operational receipt.
func (s *Store) captureHistoricalEvidence(ctx context.Context, tx *sql.Tx) (historicalEvidence, error) {
	out := historicalEvidence{artifacts: map[string][]string{}}
	// Every backend write takes this workspace barrier. Lock all historical
	// workspaces before capturing and replacing the shared evidence tables.
	if s.backend == "postgresql" {
		rows, e := tx.QueryContext(ctx, "SELECT id FROM workspace ORDER BY id FOR UPDATE")
		if e != nil {
			return out, e
		}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				break
			}
		}
		err := rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
		if err != nil {
			return out, err
		}
	}
	for _, table := range []historicalEvidenceRows{
		{table: "training_dataset", columns: "workspace_id,id,speaker_id", rows: [][]any{}},
		{table: "training_run", columns: "workspace_id,id,dataset_id,speaker_id,job_id,adapter", rows: [][]any{}},
		{table: "model_version", columns: "workspace_id,id,model_id,run_id,dataset_id,kind", rows: [][]any{}},
		{table: "model_artifact", columns: "workspace_id,id,version_id,artifact_id,role,format", rows: [][]any{}},
	} {
		rows, e := tx.QueryContext(ctx, "SELECT "+table.columns+" FROM "+table.table+" ORDER BY workspace_id,id")
		if e != nil {
			return out, e
		}
		for rows.Next() {
			values := make([]any, len(strings.Split(table.columns, ",")))
			ptrs := make([]any, len(values))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if e = rows.Scan(ptrs...); e != nil {
				break
			}
			for i, value := range values {
				if b, ok := value.([]byte); ok {
					values[i] = string(b)
				}
			}
			table.rows = append(table.rows, values)
		}
		err := rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
		if err != nil {
			return out, err
		}
		out.lineage = append(out.lineage, table)
	}
	query := "SELECT workspace_id,clip_artifact_id AS artifact_id FROM speaker_segment WHERE clip_artifact_id IS NOT NULL UNION SELECT workspace_id,manifest_artifact_id FROM training_dataset WHERE manifest_artifact_id IS NOT NULL UNION SELECT workspace_id,preparation_artifact_id FROM training_run WHERE preparation_artifact_id IS NOT NULL UNION SELECT workspace_id,manifest_artifact_id FROM model_version WHERE manifest_artifact_id IS NOT NULL ORDER BY workspace_id,artifact_id"
	rows, e := tx.QueryContext(ctx, query)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var workspace, artifact string
		if e = rows.Scan(&workspace, &artifact); e != nil {
			break
		}
		out.artifacts[workspace] = append(out.artifacts[workspace], artifact)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	if err != nil {
		return out, err
	}
	for _, table := range []string{"model_artifact", "model_version", "training_run", "dataset_member", "training_dataset", "speaker_segment"} {
		if _, e = tx.ExecContext(ctx, "DROP TABLE "+table); e != nil {
			return out, e
		}
	}
	return out, nil
}

func (s *Store) restoreHistoricalEvidence(ctx context.Context, tx *sql.Tx, old historicalEvidence) error {
	for _, table := range old.lineage {
		columns := table.columns
		for _, values := range table.rows {
			args := append([]any{}, values...)
			switch table.table {
			case "training_dataset":
				var revision int64
				workspace := values[0].(string)
				if e := s.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", workspace).Scan(&revision); e != nil {
					return e
				}
				columns = table.columns + ",manifest_artifact_id,options,state,invalidated_revision,invalidated_by"
				args = append(args, nil, "{}", "invalidated", revision+1, workspace)
			case "training_run":
				columns = table.columns + ",preparation_artifact_id,options,state"
				args = append(args, nil, "{}", "invalidated")
			case "model_version":
				columns = table.columns + ",manifest_artifact_id,state"
				args = append(args, nil, "invalidated")
			}
			marks := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
			if _, e := s.exec(ctx, tx, "INSERT INTO "+table.table+"("+columns+") VALUES("+marks+")", args...); e != nil {
				return e
			}
		}
	}
	for workspace, artifacts := range old.artifacts {
		scope := *s
		scope.workspace = workspace
		var revision int64
		if e := scope.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", workspace).Scan(&revision); e != nil {
			return e
		}
		if e := scope.queueEvidenceArtifacts(ctx, tx, workspace, artifacts, revision+1); e != nil {
			return e
		}
		ids, e := scope.evidenceArtifacts(ctx, tx, "SELECT id FROM library_cleanup WHERE workspace_id=? AND entry_id=? AND revision=? ORDER BY id", workspace, workspace, revision+1)
		if e != nil {
			return e
		}
		invalidated, e := scope.invalidatedDatasetIDs(ctx, tx, workspace, revision+1)
		if e != nil {
			return e
		}
		if len(ids)+len(invalidated) == 0 {
			continue
		}
		digest, e := intent([]any{"schema4-reference-evidence", workspace, artifacts})
		if e != nil {
			return e
		}
		if _, e = scope.accept(ctx, tx, operationID("evidence-migration", workspace), digest, revision, map[string]any{"cleanup_entry_id": workspace, "cleanup_ids": ids, "invalidated_dataset_ids": invalidated, "legacy_evidence": "invalidated"}); e != nil {
			return e
		}
	}
	return nil
}
