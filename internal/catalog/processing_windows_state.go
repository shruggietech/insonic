// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/shruggietech/insonic/internal/contracts"
)

// Retirement belongs to the same receipt/transaction as accepted source or
// transcript replacement. No obsolete window bytes become an alternate store.
func (s *Store) reconcileProcessingAuthority(ctx context.Context, tx *sql.Tx, op string, revision int64, result map[string]json.RawMessage) error {
	var mediaID string
	for _, key := range []string{"recording_id", "media_id"} {
		if raw, ok := result[key]; ok {
			if strict(raw, &mediaID) != nil || !contracts.ValidID(mediaID) {
				return contracts.Fail("invalid_request")
			}
			break
		}
	}
	if mediaID == "" {
		return nil
	}
	ids, err := s.identityIDs(ctx, tx, "processing_plan", "media_id=? AND state='active'", []any{mediaID}, "", 100001)
	if err != nil {
		return err
	}
	if len(ids) > 100000 {
		return contracts.Fail("output_limit")
	}
	planProofs, checkpointProofs := map[string]string{}, map[string]string{}
	var cleanupIDs []string
	if raw, ok := result["cleanup_ids"]; ok && strict(raw, &cleanupIDs) != nil {
		return contracts.Fail("invalid_request")
	}
	for _, id := range ids {
		var p ProcessingPlan
		if err = s.domainTx(ctx, tx, "ProcessingPlans", id, &p); err != nil {
			return err
		}
		state := "invalidated"
		if op == operationID("recording-current", p.WorkID, mediaID) {
			state = "completed"
		} else if err = s.processingSourceAuthority(ctx, tx, p); err == nil {
			continue
		} else if typed, ok := err.(*contracts.Error); !ok || typed.Code != "conflict" {
			return err
		}
		p.State = state
		if err = s.putDomain(ctx, tx, "ProcessingPlans", p); err != nil {
			return err
		}
		planProofs[p.ID] = processingRecordProof(p)
		checkpointIDs, e := s.identityIDs(ctx, tx, "processing_window_checkpoint", "plan_id=? AND state='ready'", []any{p.ID}, "", 300001)
		if e != nil {
			return e
		}
		if len(checkpointIDs) > 300000 {
			return contracts.Fail("output_limit")
		}
		for _, checkpointID := range checkpointIDs {
			var c ProcessingWindowCheckpoint
			if err = s.domainTx(ctx, tx, "ProcessingCheckpoints", checkpointID, &c); err != nil {
				return err
			}
			c.State = "retired"
			if err = s.putDomain(ctx, tx, "ProcessingCheckpoints", c); err != nil {
				return err
			}
			checkpointProofs[c.ID] = processingRecordProof(c)
			if err = s.queueRemoved(ctx, tx, mediaID, publicationList(&c.PublicationID), json.RawMessage(`[]`), revision); err != nil {
				return err
			}
			cleanupIDs = append(cleanupIDs, c.PublicationID)
		}
	}
	if len(planProofs) == 0 {
		return nil
	}
	result["processing_plan_proofs"], _ = json.Marshal(planProofs)
	result["processing_checkpoint_proofs"], _ = json.Marshal(checkpointProofs)
	if len(cleanupIDs) > 0 {
		sort.Strings(cleanupIDs)
		unique := cleanupIDs[:0]
		for _, id := range cleanupIDs {
			if len(unique) == 0 || unique[len(unique)-1] != id {
				unique = append(unique, id)
			}
		}
		result["cleanup_ids"], _ = json.Marshal(unique)
		result["cleanup_entry_id"], _ = json.Marshal(mediaID)
	}
	return nil
}

func (s *Store) validateProcessingState(ctx context.Context, tx *sql.Tx, records Records) error {
	proofs := map[string]map[string]string{"processing_plan_proofs": {}, "processing_checkpoint_proofs": {}}
	rows, err := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if err != nil {
		return err
	}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			break
		}
		var result map[string]json.RawMessage
		if err = strict([]byte(raw), &result); err != nil {
			break
		}
		for key, values := range proofs {
			if value, exists := result[key]; exists {
				var changes map[string]string
				if err = strict(value, &changes); err != nil {
					break
				}
				for id, digest := range changes {
					if !contracts.ValidID(id) || !digestPattern.MatchString(digest) {
						err = contracts.Fail("invalid_request")
						break
					}
					values[id] = digest
				}
			}
		}
		if err != nil {
			break
		}
	}
	rowErr := rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if rowErr != nil {
		return rowErr
	}
	verify := func(key, id string, value any) error {
		if proofs[key][id] != processingRecordProof(value) {
			return contracts.Fail("invalid_request")
		}
		delete(proofs[key], id)
		return nil
	}
	plans := make(map[string]ProcessingPlan, len(records.ProcessingPlans))
	windows := make(map[string]map[string]bool, len(records.ProcessingPlans))
	generations := make(map[string]int64, len(records.ProcessingPlans))
	for _, p := range records.ProcessingPlans {
		if !validProcessingPlan(p) {
			return contracts.Fail("invalid_request")
		}
		if err = verify("processing_plan_proofs", p.ID, p); err != nil {
			return err
		}
		var w Work
		if err = s.domainTx(ctx, tx, "Works", p.WorkID, &w); err != nil {
			return err
		}
		if !validProcessingWork(p, w) {
			return contracts.Fail("invalid_request")
		}
		plans[p.ID] = p
		generations[p.WorkID] = w.Generation
		doc, err := processingPlanDocument(p)
		if err != nil {
			return err
		}
		members := make(map[string]bool, len(doc.Windows))
		for _, window := range doc.Windows {
			members[window.ID] = true
		}
		windows[p.ID] = members
		if p.State == "active" {
			if err = s.processingSourceAuthority(ctx, tx, p); err != nil {
				return err
			}
		}
	}
	for _, c := range records.ProcessingCheckpoints {
		if !validProcessingCheckpoint(c) {
			return contracts.Fail("invalid_request")
		}
		if err = verify("processing_checkpoint_proofs", c.ID, c); err != nil {
			return err
		}
		p, exists := plans[c.PlanID]
		if !exists || p.WorkID != c.WorkID || !windows[c.PlanID][c.WindowID] || c.Attempt > generations[c.WorkID] || c.Attempt < p.Attempt || (c.State == "ready") != (p.State == "active") {
			return contracts.Fail("invalid_request")
		}
		if err = s.processingCheckpointPublication(ctx, tx, c); err != nil {
			return err
		}
	}
	if len(proofs["processing_plan_proofs"])+len(proofs["processing_checkpoint_proofs"]) != 0 {
		return contracts.Fail("invalid_request")
	}
	return nil
}
