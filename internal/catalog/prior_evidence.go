// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

func (s *Store) evidenceJSONText(value, key string) string {
	if s.backend == "postgresql" {
		return "(" + value + "->>'" + key + "')"
	}
	return "json_extract(" + value + ",'$." + key + "')"
}
func (s *Store) evidenceJSONCues(document, alias string) string {
	if s.backend == "postgresql" {
		return "JOIN LATERAL jsonb_array_elements(" + document + "::jsonb->'cues') WITH ORDINALITY " + alias + "(value,ordinality) ON true"
	}
	return "JOIN json_each(" + document + ",'$.cues') " + alias + " ON 1=1"
}
func (s *Store) evidenceJSONAssignments(cue, alias string) string {
	if s.backend == "postgresql" {
		return "JOIN LATERAL jsonb_array_elements(" + cue + "->'speaker_attributions') WITH ORDINALITY " + alias + "(value,ordinality) ON true"
	}
	return "JOIN json_each(" + cue + ",'$.speaker_attributions') " + alias + " ON 1=1"
}
func (s *Store) evidenceOrdinal(alias string) string {
	if s.backend == "postgresql" {
		return "(" + alias + ".ordinality-1)"
	}
	return "CAST(" + alias + ".key AS BIGINT)"
}
func (s *Store) evidenceMS(value, key string) string {
	return "CAST(" + s.evidenceJSONText(value, key) + " AS BIGINT)"
}

// Compare only sole-current embedded evidence, in canonical reference and
// assignment order. Each reference uses one scalar-result query for all its
// assignments; no prior assignment arrays leave the database or enter cursors.
func (s *Store) ComparePriorSpeakerEvidence(ctx context.Context, selection SpeakerSelection, ref CurrentReference, expectedEpoch string) (out []PriorEvidenceComparison, e error) {
	out = []PriorEvidenceComparison{}
	if !digestPattern.MatchString(expectedEpoch) || !contracts.ValidID(selection.SpeakerID) || selection.SpeakerID != ref.SpeakerID || (selection.RecordingID != "" && selection.RecordingID != ref.RecordingID) || !contracts.ValidID(ref.RecordingID) || !contracts.ValidLocalSpeakerID(ref.LocalSpeakerID) || ref.RecordingRevision < 1 || ref.MappingRevision < 1 || ref.CueID == "" || !digestPattern.MatchString(ref.DocumentDigest) || !digestPattern.MatchString(ref.SourceDigest) || !digestPattern.MatchString(ref.SourceMapDigest) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	epoch, e := s.selectionEpoch(ctx, tx, selection)
	if e != nil {
		return out, sanitize(e)
	}
	if epoch != expectedEpoch {
		return out, contracts.Fail("conflict")
	}
	var recordingRevision, mappingRevision int64
	var document, source, sourceMap, person, mappingDocument string
	e = s.row(ctx, tx, "SELECT r.revision,r.document_digest,r.source_digest,r.source_map,m.revision,m.speaker_id,m.document_digest FROM current_recording r JOIN speaker_mapping m ON m.workspace_id=r.workspace_id AND m.recording_id=r.id WHERE r.workspace_id=? AND r.id=? AND r.state='ready' AND m.local_speaker_id=?", s.workspace, ref.RecordingID, ref.LocalSpeakerID).Scan(&recordingRevision, &document, &source, &sourceMap, &mappingRevision, &person, &mappingDocument)
	if e == sql.ErrNoRows {
		return out, contracts.Fail("conflict")
	}
	if e != nil {
		return out, sanitize(e)
	}
	canonicalMap, e := canonical([]byte(sourceMap))
	if e != nil {
		return out, e
	}
	if recordingRevision != ref.RecordingRevision || mappingRevision != ref.MappingRevision || document != ref.DocumentDigest || mappingDocument != document || source != ref.SourceDigest || person != ref.SpeakerID || hash(canonicalMap) != ref.SourceMapDigest {
		return out, contracts.Fail("conflict")
	}
	var routing struct {
		Stream  *int64 `json:"stream_index"`
		Channel *int64 `json:"channel"`
	}
	if json.Unmarshal(canonicalMap, &routing) != nil {
		return out, contracts.Fail("invalid_request")
	}
	// Filter source identity before expanding any prior cues. An unknown stream
	// cannot establish equivalence across separate library entries.
	prior := "SELECT r.id,r.document,m.local_speaker_id FROM speaker_mapping m JOIN current_recording r ON r.workspace_id=m.workspace_id AND r.id=m.recording_id WHERE m.workspace_id=? AND m.speaker_id=? AND r.state='ready' AND m.document_digest=r.document_digest AND r.source_digest=? AND r.id<=?"
	args := []any{s.workspace, selection.SpeakerID, ref.SourceDigest, ref.RecordingID}
	if selection.ConfirmedOnly {
		prior += " AND m.origin='manual'"
	}
	if selection.RecordingID != "" {
		prior += " AND r.id=?"
		args = append(args, selection.RecordingID)
	}
	if routing.Stream == nil {
		prior += " AND r.id=?"
		args = append(args, ref.RecordingID)
	} else {
		mapValue := "r.source_map"
		if s.backend == "postgresql" {
			mapValue += "::jsonb"
		}
		prior += " AND " + s.evidenceMS(mapValue, "stream_index") + "=?"
		args = append(args, *routing.Stream)
		channelJSON := s.evidenceJSONText(mapValue, "channel")
		if routing.Channel != nil {
			prior += " AND CAST(" + channelJSON + " AS BIGINT)=?"
			args = append(args, *routing.Channel)
		} else {
			prior += " AND " + channelJSON + " IS NULL"
		}
	}
	cueID := s.evidenceJSONText("c.value", "id")
	cueOrdinal := s.evidenceOrdinal("c")
	assignmentOrdinal := s.evidenceOrdinal("a")
	assignmentSpeaker := s.evidenceJSONText("a.value", "speaker_id")
	start := s.evidenceMS("a.value", "start_milliseconds")
	end := s.evidenceMS("a.value", "end_milliseconds")
	q := "WITH prior_recordings AS (" + prior + "), current_cue AS (SELECT r.id,r.local_speaker_id," + cueOrdinal + " AS cue_ordinal,c.value AS cue FROM prior_recordings r " + s.evidenceJSONCues("r.document", "c") + " WHERE r.id=? AND r.local_speaker_id=? AND " + cueID + "=?), targets AS (SELECT t.id,t.local_speaker_id,t.cue_ordinal," + assignmentOrdinal + " AS assignment_ordinal," + start + " AS start_ms," + end + " AS end_ms FROM current_cue t " + s.evidenceJSONAssignments("t.cue", "a") + " WHERE " + assignmentSpeaker + "=t.local_speaker_id) "
	args = append(args, ref.RecordingID, ref.LocalSpeakerID, ref.CueID)
	earlierRef := "(r.id<t.id OR (r.id=t.id AND " + cueOrdinal + "<t.cue_ordinal) OR (r.id=t.id AND " + cueOrdinal + "=t.cue_ordinal AND r.local_speaker_id<t.local_speaker_id))"
	earlierAssignment := "(" + earlierRef + " OR (r.id=t.id AND " + cueOrdinal + "=t.cue_ordinal AND r.local_speaker_id=t.local_speaker_id AND " + assignmentOrdinal + "<t.assignment_ordinal))"
	priorAssignments := "SELECT 1 FROM prior_recordings r " + s.evidenceJSONCues("r.document", "c") + " " + s.evidenceJSONAssignments("c.value", "a") + " WHERE " + assignmentSpeaker + "=r.local_speaker_id AND " + earlierAssignment
	duplicate := "EXISTS(" + priorAssignments + " AND " + start + "=t.start_ms AND " + end + "=t.end_ms)"
	overlap := "EXISTS(" + priorAssignments + " AND " + start + "<t.end_ms AND " + end + ">t.start_ms)"
	firstRef := "NOT EXISTS(SELECT 1 FROM prior_recordings r " + s.evidenceJSONCues("r.document", "c") + " WHERE r.id=t.id AND " + earlierRef + " AND EXISTS(SELECT 1 FROM " + assignmentArrayFrom(s, "c.value", "a") + " WHERE " + assignmentSpeaker + "=r.local_speaker_id))"
	q += "SELECT t.assignment_ordinal," + duplicate + "," + overlap + ",(SELECT count(DISTINCT r.local_speaker_id) FROM prior_recordings r WHERE r.id=t.id)," + firstRef + " FROM targets t ORDER BY t.assignment_ordinal LIMIT 65537"
	rows, e := tx.QueryContext(ctx, s.query(q), args...)
	if e != nil {
		return out, sanitize(e)
	}
	for rows.Next() {
		var value PriorEvidenceComparison
		if e = rows.Scan(&value.AssignmentOrdinal, &value.Duplicate, &value.Overlap, &value.LocalVoices, &value.FirstReference); e != nil {
			break
		}
		out = append(out, value)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return out, sanitize(e)
	}
	if err != nil {
		return out, sanitize(err)
	}
	if len(out) == 0 {
		return out, contracts.Fail("conflict")
	}
	if len(out) > 65536 {
		return out, contracts.Fail("output_limit")
	}
	return out, sanitize(tx.Commit())
}
func assignmentArrayFrom(s *Store, cue, alias string) string {
	if s.backend == "postgresql" {
		return "jsonb_array_elements(" + cue + "->'speaker_attributions') " + alias + "(value)"
	}
	return "json_each(" + cue + ",'$.speaker_attributions') " + alias
}
