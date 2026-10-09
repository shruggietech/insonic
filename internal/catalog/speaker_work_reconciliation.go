// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

// Replace obsolete execution snapshots with noncontent proof markers in the same
// transaction that replaces current evidence. Old leases immediately lose authority.
func (s *Store) reconcileSpeakerWorkAuthority(ctx context.Context, tx *sql.Tx, op string, result map[string]json.RawMessage) error {
	var recordingID string
	for _, key := range []string{"recording_id", "mapping_recording_id"} {
		if raw, ok := result[key]; ok {
			if e := strict(raw, &recordingID); e != nil {
				return e
			}
			break
		}
	}
	if recordingID == "" {
		if _, ok := result["speaker_output_proofs"]; !ok {
			return nil
		}
	}
	var recording Recording
	if recordingID != "" {
		if e := s.domainTx(ctx, tx, "Recordings", recordingID, &recording); e != nil {
			return e
		}
	}
	payload := "payload"
	if s.backend == "postgresql" {
		payload = "payload::jsonb"
	}
	snapshotID := s.evidenceJSONText(s.evidenceJSONText(payload, "snapshot"), "recording")
	// Nested JSON objects use -> on PostgreSQL, rather than ->> scalar extraction.
	if s.backend == "postgresql" {
		snapshotID = "(" + payload + "->'snapshot'->'recording')"
	} else {
		snapshotID = "json_extract(payload,'$.snapshot.recording')"
	}
	datasetID := s.evidenceJSONText(s.evidenceJSONText(payload, "options"), "dataset_id")
	if s.backend == "postgresql" {
		datasetID = "(" + payload + "->'options'->>'dataset_id')"
	}
	staleOutput := "EXISTS (SELECT 1 FROM json_each(work_operation.payload,'$.snapshot.outputs') frozen JOIN speaker_output current_output ON current_output.workspace_id=work_operation.workspace_id AND current_output.id=json_extract(frozen.value,'$.id') WHERE json_extract(frozen.value,'$.metadata.state')='current' AND json_extract(current_output.metadata,'$.state')='invalidated')"
	if s.backend == "postgresql" {
		staleOutput = "EXISTS (SELECT 1 FROM jsonb_array_elements(COALESCE(work_operation.payload::jsonb->'snapshot'->'outputs','[]'::jsonb)) frozen(value) JOIN speaker_output current_output ON current_output.workspace_id=work_operation.workspace_id AND current_output.id=frozen.value->>'id' WHERE frozen.value->'metadata'->>'state'='current' AND current_output.metadata::jsonb->>'state'='invalidated')"
	}
	where := "(kind='recordings.match' AND (" + s.evidenceJSONText(snapshotID, "id") + "=? OR " + staleOutput + ")) OR (kind='models.train' AND state IN ('pending','running','interrupted') AND NOT EXISTS(SELECT 1 FROM speaker_output o WHERE o.workspace_id=work_operation.workspace_id AND o.work_id=work_operation.id) AND " + datasetID + " IN (SELECT id FROM training_dataset WHERE workspace_id=? AND state='invalidated'))"
	ids, e := s.identityIDs(ctx, tx, "work_operation", where, []any{recordingID, s.workspace}, "", 10001)
	if e != nil {
		return e
	}
	if len(ids) > 10000 {
		return contracts.Fail("output_limit")
	}
	proofs := map[string]string{}
	inputs := map[string]map[string]string{}
	for _, id := range ids {
		var w Work
		if e = s.domainTx(ctx, tx, "Works", id, &w); e != nil {
			return e
		}
		var request struct {
			RequestDigest string `json:"request_digest"`
			Snapshot      struct {
				Recording Recording       `json:"recording"`
				Outputs   []SpeakerOutput `json:"outputs"`
				Digest    string          `json:"digest"`
			} `json:"snapshot"`
		}
		if e = json.Unmarshal(w.Payload, &request); e != nil {
			return contracts.Fail("invalid_request")
		}
		scrub := false
		if w.Kind == "recordings.match" {
			r := request.Snapshot.Recording
			if r.ID == "" {
				continue
			}
			scrub = r.ID == recording.ID && (r.Revision != recording.Revision || r.DocumentDigest != recording.DocumentDigest || r.SourceDigest != recording.SourceDigest || r.SourceRevision != recording.SourceRevision)
			for _, old := range request.Snapshot.Outputs {
				var current SpeakerOutput
				if e = s.domainTx(ctx, tx, "SpeakerOutputs", old.ID, &current); e != nil {
					return e
				}
				if !bytes.Equal(old.Metadata, current.Metadata) {
					scrub = true
				}
			}
			if !scrub {
				continue
			}
		}
		if w.State == "pending" || w.State == "running" || w.State == "interrupted" {
			w.State = "failed"
			w.Error = "operation_failed"
		}
		w.LeaseUntil = 0
		w.Phase = "evidence-invalidated"
		w.Result, _ = json.Marshal(map[string]any{"recording_id": recordingID, "stale": true, "snapshot_digest": request.Snapshot.Digest})
		if scrub {
			var initial string
			if e = s.row(ctx, tx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.ID).Scan(&initial); e != nil {
				return e
			}
			marker := map[string]string{"invalidated_recording_id": request.Snapshot.Recording.ID, "input_digest": initial, "snapshot_digest": request.Snapshot.Digest}
			if request.RequestDigest != "" {
				if !digestPattern.MatchString(request.RequestDigest) {
					return contracts.Fail("invalid_request")
				}
				marker["request_digest"] = request.RequestDigest
			}
			w.Payload, _ = json.Marshal(marker)
			pd, _ := intent(w.Payload)
			inputs[id] = map[string]string{"initial_digest": initial, "payload_digest": pd}
		}
		w.JournalReceiptID = op
		if !validWork(w) {
			return contracts.Fail("invalid_request")
		}
		if e = s.putDomain(ctx, tx, "Works", w); e != nil {
			return e
		}
		proofs[id] = workJournalDigest(w)
	}
	if len(proofs) > 0 {
		result["reconciled_work_proofs"], _ = json.Marshal(proofs)
	}
	if len(inputs) > 0 {
		result["speaker_work_input_proofs"], _ = json.Marshal(inputs)
	}
	return nil
}
func (s *Store) validateScrubbedSpeakerWorkInput(ctx context.Context, tx *sql.Tx, w Work, initial string) error {
	var marker struct {
		RequestDigest  string `json:"request_digest,omitempty"`
		RecordingID    string `json:"invalidated_recording_id"`
		InputDigest    string `json:"input_digest"`
		SnapshotDigest string `json:"snapshot_digest"`
	}
	if strict(w.Payload, &marker) != nil || !contracts.ValidID(marker.RecordingID) || marker.InputDigest != initial || !digestPattern.MatchString(marker.SnapshotDigest) || marker.RequestDigest != "" && !digestPattern.MatchString(marker.RequestDigest) {
		return contracts.Fail("invalid_request")
	}
	pd, _ := intent(w.Payload)
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return e
		}
		var proof struct {
			Inputs map[string]map[string]string `json:"speaker_work_input_proofs"`
		}
		if json.Unmarshal([]byte(raw), &proof) != nil {
			return contracts.Fail("invalid_request")
		}
		if p, ok := proof.Inputs[w.ID]; ok && p["initial_digest"] == initial && p["payload_digest"] == pd {
			return nil
		}
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return contracts.Fail("invalid_request")
}
