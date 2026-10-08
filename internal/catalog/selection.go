// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

type evidenceCursor struct {
	Epoch         string `json:"epoch"`
	SpeakerID     string `json:"speaker_id"`
	RecordingID   string `json:"recording_id"`
	LastRecording string `json:"last_recording"`
	CueOrdinal    int64  `json:"cue_ordinal"`
	LocalID       string `json:"local_id"`
}

func (s *Store) CurrentSpeakerReferences(ctx context.Context, selection SpeakerSelection) (out SpeakerSelectionPage, e error) {
	out.References = []CurrentReference{}
	if !contracts.ValidID(selection.SpeakerID) || (selection.RecordingID != "" && !contracts.ValidID(selection.RecordingID)) {
		return out, contracts.Fail("invalid_request")
	}
	if selection.Limit == 0 {
		selection.Limit = 25
	}
	if selection.Limit < 1 || selection.Limit > 100 {
		return out, contracts.Fail("invalid_request")
	}
	cursor := evidenceCursor{}
	if selection.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(selection.Cursor)
		if err != nil || len(raw) > 1024 || strict(raw, &cursor) != nil || cursor.SpeakerID != selection.SpeakerID || cursor.RecordingID != selection.RecordingID {
			return out, contracts.Fail("invalid_request")
		}
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	if e = s.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", s.workspace).Scan(&out.Revision); e != nil {
		return out, sanitize(e)
	}
	epoch, err := s.selectionEpoch(ctx, tx, selection)
	if err != nil {
		return out, sanitize(err)
	}
	out.Epoch = epoch
	if selection.Cursor != "" && cursor.Epoch != epoch {
		return out, contracts.Fail("conflict")
	}
	var exists bool
	if e = s.row(ctx, tx, "SELECT EXISTS(SELECT 1 FROM speaker WHERE workspace_id=? AND id=?)", s.workspace, selection.SpeakerID).Scan(&exists); e != nil {
		return out, sanitize(e)
	}
	if !exists {
		return out, contracts.Fail("not_found")
	}
	cueID := "json_extract(c.value,'$.id')"
	cueOrdinal := "CAST(c.key AS BIGINT)"
	expand := "JOIN json_each(r.document,'$.cues') c ON 1=1"
	match := "EXISTS(SELECT 1 FROM json_each(c.value,'$.speaker_attributions') a WHERE json_extract(a.value,'$.speaker_id')=m.local_speaker_id)"
	if s.backend == "postgresql" {
		cueID = "c.value->>'id'"
		expand = "JOIN LATERAL jsonb_array_elements(r.document::jsonb->'cues') WITH ORDINALITY c(value,ordinality) ON true"
		cueOrdinal = "(c.ordinality-1)"
		match = "EXISTS(SELECT 1 FROM jsonb_array_elements(c.value->'speaker_attributions') a(value) WHERE a.value->>'speaker_id'=m.local_speaker_id)"
	}
	q := "SELECT r.id,r.revision,r.document_digest," + cueID + "," + cueOrdinal + ",m.local_speaker_id,m.revision,r.source_digest,r.source_map FROM speaker_mapping m JOIN current_recording r ON r.workspace_id=m.workspace_id AND r.id=m.recording_id " + expand + " WHERE m.workspace_id=? AND m.speaker_id=? AND r.state='ready' AND m.document_digest=r.document_digest AND " + match
	args := []any{s.workspace, selection.SpeakerID}
	if selection.RecordingID != "" {
		q += " AND r.id=?"
		args = append(args, selection.RecordingID)
	}
	if selection.Cursor != "" {
		q += " AND (r.id>? OR (r.id=? AND " + cueOrdinal + ">?) OR (r.id=? AND " + cueOrdinal + "=? AND m.local_speaker_id>?))"
		args = append(args, cursor.LastRecording, cursor.LastRecording, cursor.CueOrdinal, cursor.LastRecording, cursor.CueOrdinal, cursor.LocalID)
	}
	q += " ORDER BY r.id," + cueOrdinal + ",m.local_speaker_id LIMIT ?"
	args = append(args, selection.Limit+1)
	lastOrdinal := int64(0)
	rows, e := tx.QueryContext(ctx, s.query(q), args...)
	if e != nil {
		return out, sanitize(e)
	}
	for rows.Next() {
		ref := CurrentReference{SpeakerID: selection.SpeakerID}
		var raw string
		var ordinal int64
		if e = rows.Scan(&ref.RecordingID, &ref.RecordingRevision, &ref.DocumentDigest, &ref.CueID, &ordinal, &ref.LocalSpeakerID, &ref.MappingRevision, &ref.SourceDigest, &raw); e != nil {
			break
		}
		canonicalMap, err := canonical([]byte(raw))
		if err != nil {
			e = err
			break
		}
		ref.SourceMapDigest = hash(canonicalMap)
		out.References = append(out.References, ref)
		if len(out.References) <= selection.Limit {
			lastOrdinal = ordinal
		}
	}
	err = rows.Err()
	rows.Close()
	if e != nil {
		return out, sanitize(e)
	}
	if err != nil {
		return out, sanitize(err)
	}
	if len(out.References) > selection.Limit {
		out.References = out.References[:selection.Limit]
		last := out.References[len(out.References)-1]
		raw, _ := json.Marshal(evidenceCursor{epoch, selection.SpeakerID, selection.RecordingID, last.RecordingID, lastOrdinal, last.LocalSpeakerID})
		out.Next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return out, sanitize(tx.Commit())
}
func (s *Store) ResolveEvidence(ctx context.Context, ref CurrentReference) (out ResolvedEvidence, e error) {
	if !contracts.ValidID(ref.RecordingID) || !contracts.ValidID(ref.LocalSpeakerID) || !contracts.ValidID(ref.SpeakerID) || ref.RecordingRevision < 1 || ref.MappingRevision < 1 || ref.CueID == "" || !digestPattern.MatchString(ref.DocumentDigest) || !digestPattern.MatchString(ref.SourceDigest) || !digestPattern.MatchString(ref.SourceMapDigest) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	var r Recording
	if e = s.domainTx(ctx, tx, "Recordings", ref.RecordingID, &r); e != nil {
		return out, sanitize(e)
	}
	raw, e := canonical(r.SourceMap)
	if e != nil {
		return out, e
	}
	if r.Revision != ref.RecordingRevision || r.DocumentDigest != ref.DocumentDigest || r.SourceDigest != ref.SourceDigest || hash(raw) != ref.SourceMapDigest {
		return out, contracts.Fail("conflict")
	}
	id := operationID("speaker-mapping", ref.RecordingID, ref.LocalSpeakerID)
	if e = s.domainTx(ctx, tx, "SpeakerMappings", id, &out.Mapping); e != nil {
		if e == sql.ErrNoRows {
			return out, contracts.Fail("conflict")
		}
		return out, sanitize(e)
	}
	if out.Mapping.Revision != ref.MappingRevision || out.Mapping.SpeakerID != ref.SpeakerID || out.Mapping.DocumentDigest != ref.DocumentDigest {
		return out, contracts.Fail("conflict")
	}
	out.Cue, e = s.segmentCue(ctx, tx, Segment{RecordingID: ref.RecordingID, DocumentDigest: ref.DocumentDigest, CueID: ref.CueID, LocalSpeakerID: ref.LocalSpeakerID})
	if e != nil {
		return out, e
	}
	out.Reference = ref
	out.SourceMap = r.SourceMap
	return out, sanitize(tx.Commit())
}
