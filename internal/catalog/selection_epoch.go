// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
)

// Stream only thin selected-person mapping/current-document identities into a
// digest. Memory stays constant and unrelated work/configuration writes do not
// invalidate a continuation. No cue text, timing or assignment arrays are kept.
func (s *Store) selectionEpoch(ctx context.Context, tx *sql.Tx, selection SpeakerSelection) (string, error) {
	q := "SELECT m.id,m.revision,r.id,r.revision,r.document_digest,r.source_digest FROM speaker_mapping m JOIN current_recording r ON r.workspace_id=m.workspace_id AND r.id=m.recording_id WHERE m.workspace_id=? AND m.speaker_id=? AND r.state='ready' AND m.document_digest=r.document_digest"
	args := []any{s.workspace, selection.SpeakerID}
	if selection.RecordingID != "" {
		q += " AND r.id=?"
		args = append(args, selection.RecordingID)
	}
	q += " ORDER BY m.id"
	rows, e := tx.QueryContext(ctx, s.query(q), args...)
	if e != nil {
		return "", e
	}
	defer rows.Close()
	h := sha256.New()
	for rows.Next() {
		var id, recording, document, source string
		var mappingRevision, recordingRevision int64
		if e = rows.Scan(&id, &mappingRevision, &recording, &recordingRevision, &document, &source); e != nil {
			return "", e
		}
		raw, _ := json.Marshal([]any{id, mappingRevision, recording, recordingRevision, document, source})
		h.Write(raw)
		h.Write([]byte{'\n'})
	}
	if e = rows.Err(); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
