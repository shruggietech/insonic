// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
)

func (s *Store) reconcileDiscoveryAuthority(ctx context.Context, tx *sql.Tx, result map[string]json.RawMessage) error {
	if e := s.invalidateDiscoveryForCompletedMatch(ctx, tx, result); e != nil {
		return e
	}
	relevant := false
	for _, key := range []string{"recording_id", "mapping_recording_id", "speaker_proofs", "roster_proofs", "speaker_profile_proofs", "speaker_output_proofs", "invalidated_dataset_ids", "media_id", "accepted", "speaker_association_proofs"} {
		if _, ok := result[key]; ok {
			relevant = true
		}
	}
	if !relevant {
		return nil
	}
	// Iterate compact identity pages; never retain the library's derived content.
	cursor := ""
	proofs := map[string]string{}
	if raw, ok := result["discovery_proofs"]; ok {
		if e := json.Unmarshal(raw, &proofs); e != nil {
			return e
		}
	}
	for {
		ids, e := s.identityIDs(ctx, tx, "current_speaker_discovery", "", nil, cursor, 100)
		if e != nil {
			return e
		}
		for _, id := range ids {
			var v SpeakerDiscovery
			if e = s.domainTx(ctx, tx, "Discoveries", id, &v); e != nil {
				return e
			}
			if _, e = s.discoveryCurrentTx(ctx, tx, v); e == nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Only current-authority failures remove evidence; infrastructure failures abort.
			if e != sql.ErrNoRows {
				if _, ok := e.(*contracts.Error); !ok {
					return e
				}
			}
			if _, e = s.exec(ctx, tx, "DELETE FROM current_speaker_discovery WHERE workspace_id=? AND id=?", s.workspace, id); e != nil {
				return e
			}
			proofs[id] = ""
		}
		if len(ids) < 100 {
			break
		}
		cursor = ids[len(ids)-1]
	}
	if len(proofs) > 0 {
		result["discovery_proofs"], _ = json.Marshal(proofs)
	}
	return nil
}
func (s *Store) validateDiscoveryState(ctx context.Context, tx *sql.Tx) error {
	proofs := map[string]string{}
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var value struct {
			Proofs map[string]string `json:"discovery_proofs"`
		}
		if json.Unmarshal([]byte(raw), &value) != nil {
			e = contracts.Fail("invalid_request")
			break
		}
		for id, digest := range value.Proofs {
			if !contracts.ValidID(id) || (digest != "" && !digestPattern.MatchString(digest)) {
				e = contracts.Fail("invalid_request")
				break
			}
			proofs[id] = digest
		}
	}
	re := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if re != nil {
		return re
	}
	actual := map[string]string{}
	cursor := ""
	for {
		ids, e := s.identityIDs(ctx, tx, "current_speaker_discovery", "", nil, cursor, 100)
		if e != nil {
			return e
		}
		for _, id := range ids {
			var v SpeakerDiscovery
			if e = s.domainTx(ctx, tx, "Discoveries", id, &v); e != nil {
				return e
			}
			if _, e = s.discoveryCurrentTx(ctx, tx, v); e != nil {
				return e
			}
			actual[id], _ = intent(v)
		}
		if len(ids) < 100 {
			break
		}
		cursor = ids[len(ids)-1]
	}
	for id, d := range proofs {
		if d == "" {
			delete(proofs, id)
		}
	}
	if !reflect.DeepEqual(proofs, actual) {
		return contracts.Fail("invalid_request")
	}
	return nil
}

func (s *Store) invalidateDiscoveryForCompletedMatch(ctx context.Context, tx *sql.Tx, result map[string]json.RawMessage) error {
	var state, id string
	if json.Unmarshal(result["state"], &state) != nil || state != "succeeded" || json.Unmarshal(result["work_id"], &id) != nil {
		return nil
	}
	var w Work
	if e := s.domainTx(ctx, tx, "Works", id, &w); e != nil {
		return e
	}
	if w.Kind != "recordings.match" || w.State != "succeeded" {
		return nil
	}
	var payload struct {
		Options struct {
			RecordingID string `json:"recording_id"`
		} `json:"options"`
	}
	if json.Unmarshal(w.Payload, &payload) != nil || !contracts.ValidID(payload.Options.RecordingID) {
		return nil
	}
	// The terminal journal receipt is about to be written, so a read through
	// accepted acoustic selection cannot observe this newly completed work yet.
	// Retire default selection (or this explicit selection) in that same commit.
	change, e := s.exec(ctx, tx, "DELETE FROM current_speaker_discovery WHERE workspace_id=? AND id=? AND (matching_work_id IS NULL OR matching_work_id=?)", s.workspace, payload.Options.RecordingID, w.ID)
	if e != nil {
		return e
	}
	n, e := change.RowsAffected()
	if e != nil {
		return e
	}
	if n > 0 {
		proofs := map[string]string{}
		if raw, ok := result["discovery_proofs"]; ok {
			if e = json.Unmarshal(raw, &proofs); e != nil {
				return e
			}
		}
		proofs[payload.Options.RecordingID] = ""
		result["discovery_proofs"], _ = json.Marshal(proofs)
	}
	return nil
}
