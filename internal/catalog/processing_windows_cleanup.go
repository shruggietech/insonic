// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"

	"github.com/shruggietech/insonic/internal/contracts"
)

// RecoverProcessingWindowCleanup closes the publication-before-checkpoint crash
// window. Only stale generations are considered, and ready checkpoint references
// retain their normal structural retirement fence.
func (s *Store) RecoverProcessingWindowCleanup(ctx context.Context, workID string) error {
	if !contracts.ValidID(workID) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, revision int64) error {
		var work Work
		if err := s.domainTx(ctx, tx, "Works", workID, &work); err != nil {
			return err
		}
		if work.Kind != "recordings.process" {
			return contracts.Fail("invalid_request")
		}
		ids, err := s.identityIDs(ctx, tx, "processing_plan", "work_id=?", []any{workID}, "", 2)
		if err != nil || len(ids) == 0 {
			return err
		}
		if len(ids) != 1 {
			return contracts.Fail("invalid_request")
		}
		var plan ProcessingPlan
		if err = s.domainTx(ctx, tx, "ProcessingPlans", ids[0], &plan); err != nil {
			return err
		}
		document, err := processingPlanDocument(plan)
		if err != nil {
			return err
		}
		now, err := s.now(ctx, tx)
		if err != nil {
			return err
		}
		// Enumerate actual claim generations, not every integer up to an imported
		// generation counter. Heartbeats collapse through DISTINCT.
		generation, receiptWork := "CAST(json_extract(result,'$.generation') AS BIGINT)", "json_extract(result,'$.work_id')"
		if s.backend == "postgresql" {
			generation, receiptWork = "CAST(result::jsonb->>'generation' AS BIGINT)", "result::jsonb->>'work_id'"
		}
		rows, err := tx.QueryContext(ctx, s.query("SELECT DISTINCT "+generation+" FROM operation_receipt WHERE workspace_id=? AND "+receiptWork+"=? AND "+generation+">=1 AND "+generation+"<=? ORDER BY 1"), s.workspace, workID, work.Generation)
		if err != nil {
			return err
		}
		var generations []int64
		for rows.Next() {
			var value int64
			if err = rows.Scan(&value); err != nil {
				break
			}
			if value != work.Generation || work.State != "running" || work.LeaseUntil <= now {
				generations = append(generations, value)
			}
		}
		rowErr := rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if rowErr != nil || len(generations) == 0 {
			return rowErr
		}
		kind, state := "json_extract(p.data,'$.kind')", "json_extract(p.data,'$.state')"
		if s.backend == "postgresql" {
			kind, state = "p.data::jsonb->>'kind'", "p.data::jsonb->>'state'"
		}
		rows, err = tx.QueryContext(ctx, s.query("SELECT p.id FROM artifact_publication p WHERE p.workspace_id=? AND "+kind+"='processing-window' AND "+state+"='available' AND NOT EXISTS(SELECT 1 FROM library_cleanup c WHERE c.workspace_id=p.workspace_id AND c.id=p.id) AND NOT EXISTS(SELECT 1 FROM processing_window_checkpoint c WHERE c.workspace_id=p.workspace_id AND c.publication_id=p.id AND c.state='ready')"), s.workspace)
		if err != nil {
			return err
		}
		available := map[string]bool{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				break
			}
			available[id] = true
		}
		rowErr = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if rowErr != nil || len(available) == 0 {
			return rowErr
		}
		for _, generation := range generations {
			for _, window := range document.Windows {
				if err = ctx.Err(); err != nil {
					return err
				}
				id := ProcessingWindowPublicationID(workID, plan.PlanDigest, window.ID, generation)
				if !available[id] {
					continue
				}
				publication, err := s.publicationTx(ctx, tx, id)
				if err != nil {
					return err
				}
				referenced, err := s.libraryArtifactReference(ctx, tx, publication.ArtifactID)
				if err != nil {
					return err
				}
				if referenced {
					continue
				}
				if err = s.putDomain(ctx, tx, "Cleanups", Cleanup{ID: id, EntryID: plan.MediaID, Revision: revision + 1, State: "pending"}); err != nil {
					return err
				}
				digest, _ := intent([]any{"processing-window-orphan", workID, id, generation})
				if _, err = s.accept(ctx, tx, operationID("processing-window-orphan", id), digest, revision, map[string]any{"cleanup_entry_id": plan.MediaID, "cleanup_ids": []string{id}}); err != nil {
					return err
				}
				revision++
			}
		}
		return nil
	})
}
