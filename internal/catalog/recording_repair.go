// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
)

type RecordingRepairFence struct {
	ExpectedRevision int64  `json:"expected_revision"`
	DocumentDigest   string `json:"document_digest"`
	SourceRevision   int64  `json:"source_revision"`
	SourceDigest     string `json:"source_digest"`
	MappingDigest    string `json:"mapping_digest"`
}

func SpeakerMappingsDigest(m []SpeakerMapping) string { d, _ := intent(m); return d }
func (s *Store) recordingRepairAuthority(ctx context.Context, tx *sql.Tx, claim Work, id string, f RecordingRepairFence) error {
	if claim.Kind != "recordings.repair" || f.ExpectedRevision < 1 || f.SourceRevision < 1 || !digestPattern.MatchString(f.SourceDigest) || !digestPattern.MatchString(f.MappingDigest) {
		return contracts.Fail("invalid_request")
	}
	var input struct {
		MediaID        string `json:"media_id"`
		Expected       int64  `json:"expected_revision"`
		SourceRevision int64  `json:"source_revision"`
	}
	if json.Unmarshal(claim.Payload, &input) != nil || input.MediaID != id || input.Expected != f.ExpectedRevision || input.SourceRevision != f.SourceRevision {
		return contracts.Fail("invalid_request")
	}
	var r Recording
	if e := s.domainTx(ctx, tx, "Recordings", id, &r); e != nil {
		return e
	}
	var entry LibraryEntry
	if e := s.domainTx(ctx, tx, "Library", id, &entry); e != nil {
		return e
	}
	if r.Revision != f.ExpectedRevision || r.DocumentDigest != f.DocumentDigest || r.SourceRevision != f.SourceRevision || r.SourceDigest != f.SourceDigest || entry.Revision != f.SourceRevision || entry.Digest != f.SourceDigest {
		return contracts.Condition("conflict", "revision_stale")
	}
	mappings, e := s.recordingMappings(ctx, tx, id)
	if e != nil {
		return e
	}
	if SpeakerMappingsDigest(mappings) != f.MappingDigest {
		return contracts.Condition("conflict", "revision_stale")
	}
	return nil
}
func currentRepairDecision(raw json.RawMessage, r Recording) json.RawMessage {
	var value map[string]json.RawMessage
	json.Unmarshal(raw, &value)
	value["recording_id"], _ = json.Marshal(r.ID)
	value["revision"], _ = json.Marshal(r.Revision)
	value["document_digest"], _ = json.Marshal(r.DocumentDigest)
	out, _ := json.Marshal(value)
	return out
}
func (s *Store) CommitRecordingRepair(ctx context.Context, claim Work, f RecordingRepairFence, candidate *Recording, decision json.RawMessage) (json.RawMessage, error) {
	var header struct {
		Kind    string `json:"kind"`
		Version string `json:"schema_version"`
		Status  string `json:"status"`
	}
	if len(decision) == 0 || len(decision) > 512<<10 || !referenceMetadata(decision) || json.Unmarshal(decision, &header) != nil || header.Kind != "recording-repair" || !contracts.SupportedDocumentVersion(header.Version) || candidate != nil && header.Status != "accepted" || candidate == nil && header.Status != "unchanged" {
		return nil, contracts.Fail("invalid_request")
	}
	var payload struct {
		MediaID string `json:"media_id"`
	}
	if json.Unmarshal(claim.Payload, &payload) != nil || !contracts.ValidID(payload.MediaID) {
		return nil, contracts.Fail("invalid_request")
	}
	if candidate != nil {
		if candidate.ID != payload.MediaID || candidate.SourceRevision != f.SourceRevision || candidate.SourceDigest != f.SourceDigest {
			return nil, contracts.Fail("invalid_request")
		}
		r, e := s.commitRecording(ctx, claim, f.ExpectedRevision, *candidate, &f, decision)
		if e != nil {
			return nil, e
		}
		_ = r
		raw, found, err := s.RecordingRepairDecision(ctx, claim.ID, payload.MediaID)
		if err == nil && !found {
			err = contracts.Fail("unavailable")
		}
		return raw, err
	}
	digest, e := intent([]any{"recording-repair-unchanged", f, decision})
	if e != nil {
		return nil, e
	}
	op := operationID("recording-repair", claim.ID, payload.MediaID)
	var out json.RawMessage
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, err := s.workAuthority(ctx, tx, claim); err != nil {
			return err
		}
		receipt, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			var v struct {
				Decision json.RawMessage `json:"repair_decision"`
			}
			if json.Unmarshal(receipt.Result, &v) != nil {
				return contracts.Fail("invalid_request")
			}
			out = v.Decision
			return nil
		}
		if err = s.recordingRepairAuthority(ctx, tx, claim, payload.MediaID, f); err != nil {
			return err
		}
		var r Recording
		if err = s.domainTx(ctx, tx, "Recordings", payload.MediaID, &r); err != nil {
			return err
		}
		out = currentRepairDecision(decision, r)
		_, err = s.accept(ctx, tx, op, digest, rev, map[string]any{"repair_recording_id": payload.MediaID, "repair_decision": out})
		return err
	})
	return out, e
}

// Recovery reads only accepted decision bytes. It does not infer acceptance from
// the terminal worker result, which may not have been written before interruption.
func (s *Store) RecordingRepairDecision(ctx context.Context, workID, recordingID string) (json.RawMessage, bool, error) {
	if !contracts.ValidID(workID) || !contracts.ValidID(recordingID) {
		return nil, false, contracts.Fail("invalid_request")
	}
	var raw string
	e := s.db.QueryRowContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? AND id IN (?,?) ORDER BY revision DESC LIMIT 1"), s.workspace, operationID("recording-current", workID, recordingID), operationID("recording-repair", workID, recordingID)).Scan(&raw)
	if e == sql.ErrNoRows {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, sanitize(e)
	}
	var value struct {
		Decision json.RawMessage `json:"repair_decision"`
	}
	if json.Unmarshal([]byte(raw), &value) != nil || len(value.Decision) == 0 {
		return nil, false, contracts.Fail("invalid_request")
	}
	return value.Decision, true, nil
}

// ConsumeRepairWindowAttempt records the finite budget before worker execution.
// An interrupted unverified attempt remains consumed; verified checkpoints resume
// without another call. The frozen work payload, not the caller, owns the budget.
func (s *Store) ConsumeRepairWindowAttempt(ctx context.Context, claim Work, planID, windowID string) (int, error) {
	if !contracts.ValidID(planID) || !contracts.ValidID(windowID) {
		return 0, contracts.Fail("invalid_request")
	}
	number := 0
	err := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		work, e := s.workAuthority(ctx, tx, claim)
		if e != nil {
			return e
		}
		var input struct {
			MediaID string `json:"media_id"`
			Repair  *struct {
				Fence  RecordingRepairFence `json:"fence"`
				Policy struct {
					MaxAttempts int `json:"max_attempts"`
				} `json:"policy"`
				Selection struct {
					Windows []struct {
						ID string `json:"window_id"`
					} `json:"windows"`
				} `json:"selection"`
			} `json:"repair"`
		}
		if json.Unmarshal(work.Payload, &input) != nil || input.Repair == nil || input.Repair.Policy.MaxAttempts < 1 || input.Repair.Policy.MaxAttempts > 3 {
			return contracts.Fail("invalid_request")
		}
		if e = s.recordingRepairAuthority(ctx, tx, work, input.MediaID, input.Repair.Fence); e != nil {
			return e
		}
		selected := false
		for _, w := range input.Repair.Selection.Windows {
			selected = selected || w.ID == windowID
		}
		if !selected {
			return contracts.Fail("invalid_request")
		}
		var plan ProcessingPlan
		if e = s.domainTx(ctx, tx, "ProcessingPlans", planID, &plan); e != nil {
			return e
		}
		if plan.State != "active" || !validProcessingWork(plan, work) || !processingWindowMember(plan, ProcessingWindowCheckpoint{PlanID: planID, WorkID: claim.ID, WindowID: windowID}) {
			return contracts.Fail("conflict")
		}
		for n := 1; n <= input.Repair.Policy.MaxAttempts; n++ {
			digest, _ := intent([]any{"repair-window-attempt", planID, windowID, n})
			op := operationID("repair-window-attempt", claim.ID, windowID, strconv.Itoa(n))
			_, exists, e := s.replay(ctx, tx, op, digest)
			if e != nil {
				return e
			}
			if exists {
				continue
			}
			_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"repair_attempt_work_id": claim.ID, "repair_attempt_window_id": windowID, "repair_attempt_number": n})
			number = n
			return e
		}
		return contracts.Condition("output_limit", "repair_attempts_exhausted")
	})
	return number, err
}
func (s *Store) RepairWindowAttempts(ctx context.Context, workID string) (map[string]int, error) {
	if !contracts.ValidID(workID) {
		return nil, contracts.Fail("invalid_request")
	}
	work, window, number := "json_extract(result,'$.repair_attempt_work_id')", "json_extract(result,'$.repair_attempt_window_id')", "CAST(json_extract(result,'$.repair_attempt_number') AS INTEGER)"
	if s.backend == "postgresql" {
		work, window, number = "result::jsonb->>'repair_attempt_work_id'", "result::jsonb->>'repair_attempt_window_id'", "CAST(result::jsonb->>'repair_attempt_number' AS INTEGER)"
	}
	rows, e := s.db.QueryContext(ctx, s.query("SELECT "+window+",MAX("+number+") FROM operation_receipt WHERE workspace_id=? AND "+work+"=? GROUP BY "+window), s.workspace, workID)
	if e != nil {
		return nil, sanitize(e)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if e = rows.Scan(&id, &n); e != nil {
			return nil, sanitize(e)
		}
		if !contracts.ValidID(id) || n < 1 || n > 3 || len(out) >= 16384 {
			return nil, contracts.Fail("invalid_request")
		}
		out[id] = n
	}
	return out, sanitize(rows.Err())
}
