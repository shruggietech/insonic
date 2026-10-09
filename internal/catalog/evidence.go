// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

func validEvidenceState(state string, artifact *string) bool {
	return (state == "current" && artifact != nil) || (state == "invalidated" && artifact == nil)
}

func referenceOnlyOptions(raw json.RawMessage) bool {
	if !referenceMetadata(raw) {
		return false
	}
	var value any
	if strict(raw, &value) != nil {
		return false
	}
	var visit func(any) bool
	visit = func(value any) bool {
		switch v := value.(type) {
		case map[string]any:
			for key, child := range v {
				switch strings.ToLower(key) {
				case "attribution", "speaker_attributions", "assignments", "transcript", "plain_text", "start_us", "end_us", "start_milliseconds", "end_milliseconds":
					return false
				}
				if !visit(child) {
					return false
				}
			}
		case []any:
			for _, child := range v {
				if !visit(child) {
					return false
				}
			}
		}
		return true
	}
	return visit(value)
}

type ResolvedSegment struct {
	Reference    Segment         `json:"reference"`
	Cue          json.RawMessage `json:"cue"`
	KnownSpeaker *SpeakerMapping `json:"known_speaker"`
}

func (s *Store) segmentCue(ctx context.Context, tx *sql.Tx, segment Segment) (json.RawMessage, error) {
	var r Recording
	if e := s.domainTx(ctx, tx, "Recordings", segment.RecordingID, &r); e != nil {
		return nil, e
	}
	if r.State != "ready" || r.DocumentDigest != segment.DocumentDigest {
		return nil, contracts.Fail("conflict")
	}
	var document struct {
		Cues []json.RawMessage `json:"cues"`
	}
	if json.Unmarshal(r.Document, &document) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	for _, raw := range document.Cues {
		var cue struct {
			ID           string `json:"id"`
			Attributions []struct {
				SpeakerID string `json:"speaker_id"`
			} `json:"speaker_attributions"`
		}
		if json.Unmarshal(raw, &cue) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if cue.ID != segment.CueID {
			continue
		}
		for _, a := range cue.Attributions {
			if a.SpeakerID == segment.LocalSpeakerID {
				return raw, nil
			}
		}
	}
	return nil, contracts.Fail("conflict")
}
func (s *Store) validateEvidence(ctx context.Context, tx *sql.Tx, segments []Segment) error {
	for _, segment := range segments {
		if _, e := s.segmentCue(ctx, tx, segment); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) ResolveSegment(ctx context.Context, id string, revision int64) (ResolvedSegment, error) {
	var out ResolvedSegment
	if !contracts.ValidID(id) || revision < 1 {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	r := &out.Reference
	e = s.row(ctx, tx, "SELECT id,revision,recording_id,document_digest,cue_id,local_speaker_id,clip_artifact_id FROM speaker_segment WHERE workspace_id=? AND id=? AND revision=?", s.workspace, id, revision).Scan(&r.ID, &r.Revision, &r.RecordingID, &r.DocumentDigest, &r.CueID, &r.LocalSpeakerID, &r.ClipArtifactID)
	if e != nil {
		return out, sanitize(e)
	}
	out.Cue, e = s.segmentCue(ctx, tx, *r)
	if e != nil {
		return out, sanitize(e)
	}
	mappings, e := s.recordingMappings(ctx, tx, r.RecordingID)
	if e != nil {
		return out, sanitize(e)
	}
	for _, m := range mappings {
		if m.LocalSpeakerID == r.LocalSpeakerID && m.DocumentDigest == r.DocumentDigest {
			copy := m
			out.KnownSpeaker = &copy
			break
		}
	}
	return out, sanitize(tx.Commit())
}

// Retirement candidates are derived manifests/preparations/clips only. Source
// media and retained model binaries remain protected even if an old row reused
// the same artifact identity for more than one role.
func (s *Store) queueEvidenceArtifacts(ctx context.Context, tx *sql.Tx, entry string, artifactIDs []string, revision int64) error {
	seen := map[string]bool{}
	for _, artifact := range artifactIDs {
		if artifact == "" || seen[artifact] {
			continue
		}
		seen[artifact] = true
		var protected bool
		if e := s.row(ctx, tx, "SELECT EXISTS(SELECT 1 FROM media_asset WHERE workspace_id=? AND artifact_id=? AND role IN ('original','subtitle-source')) OR EXISTS(SELECT 1 FROM model_artifact WHERE workspace_id=? AND artifact_id=? AND role IN ('weights','checkpoint','model'))", s.workspace, artifact, s.workspace, artifact).Scan(&protected); e != nil {
			return e
		}
		if protected {
			continue
		}
		rows, e := tx.QueryContext(ctx, s.query("SELECT id FROM artifact_publication WHERE workspace_id=? AND artifact_id=? ORDER BY id"), s.workspace, artifact)
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
			var existing Cleanup
			if e = s.domainTx(ctx, tx, "Cleanups", id, &existing); e == nil {
				continue
			} else if e != sql.ErrNoRows {
				return e
			}
			if e = s.putDomain(ctx, tx, "Cleanups", Cleanup{ID: id, EntryID: entry, State: "pending", Revision: revision}); e != nil {
				return e
			}
		}
		// A location from a historical catalog has not been independently admitted.
		// Keep its identity as a pending obligation until selected-store readback.
		locationExpr := "json_extract(p.data,'$.location_id')"
		if s.backend == "postgresql" {
			locationExpr = "(p.data::jsonb->>'location_id')"
		}
		locations, e := s.evidenceArtifacts(ctx, tx, "SELECT l.id FROM artifact_location l WHERE l.workspace_id=? AND l.artifact_id=? AND l.state='available' AND NOT EXISTS(SELECT 1 FROM artifact_publication p WHERE p.workspace_id=l.workspace_id AND "+locationExpr+"=l.id) ORDER BY l.id", s.workspace, artifact)
		if e != nil {
			return e
		}
		for _, id := range locations {
			var existing Cleanup
			if e = s.domainTx(ctx, tx, "Cleanups", id, &existing); e == nil {
				continue
			} else if e != sql.ErrNoRows {
				return e
			}
			location := id
			if e = s.putDomain(ctx, tx, "Cleanups", Cleanup{ID: id, EntryID: entry, State: "pending", Revision: revision, LegacyLocationID: &location}); e != nil {
				return e
			}
		}
		if _, e = s.exec(ctx, tx, "DELETE FROM model_artifact WHERE workspace_id=? AND artifact_id=? AND role NOT IN ('weights','checkpoint','model')", s.workspace, artifact); e != nil {
			return e
		}
	}
	return nil
}

func (s *Store) evidenceArtifacts(ctx context.Context, tx *sql.Tx, q string, args ...any) ([]string, error) {
	rows, e := tx.QueryContext(ctx, s.query(q), args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id sql.NullString
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		if id.Valid {
			out = append(out, id.String)
		}
	}
	return out, rows.Err()
}

func (s *Store) reconcileRecordingEvidence(ctx context.Context, tx *sql.Tx, old, new Recording, revision int64) error {
	if old.ID == "" {
		return nil
	}
	// Called on document replacement, or conservatively on known-speaker mapping
	// correction with the same document. The latter preserves current segments.
	if old.DocumentDigest == new.DocumentDigest && old.State == new.State && old.SourceDigest == new.SourceDigest && old.SourceRevision == new.SourceRevision && bytes.Equal(old.SourceMap, new.SourceMap) {
		return s.invalidateRecordingDatasets(ctx, tx, old.ID, revision)
	}
	artifacts, e := s.evidenceArtifacts(ctx, tx, "SELECT clip_artifact_id FROM speaker_segment WHERE workspace_id=? AND recording_id=?", s.workspace, old.ID)
	if e != nil {
		return e
	}
	if e = s.invalidateRecordingDatasets(ctx, tx, old.ID, revision); e != nil {
		return e
	}
	if _, e = s.exec(ctx, tx, "DELETE FROM speaker_segment WHERE workspace_id=? AND recording_id=?", s.workspace, old.ID); e != nil {
		return e
	}
	return s.queueEvidenceArtifacts(ctx, tx, old.ID, artifacts, revision)
}

func (s *Store) invalidateRecordingDatasets(ctx context.Context, tx *sql.Tx, recording string, revision int64) error {
	rows, e := tx.QueryContext(ctx, s.query("SELECT DISTINCT m.dataset_id FROM dataset_member m JOIN speaker_segment s ON s.workspace_id=m.workspace_id AND s.id=m.segment_id AND s.revision=m.segment_revision WHERE m.workspace_id=? AND s.recording_id=? ORDER BY m.dataset_id"), s.workspace, recording)
	if e != nil {
		return e
	}
	datasets := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		datasets = append(datasets, id)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	for _, id := range datasets {
		artifacts := []string{}
		for _, query := range []string{"SELECT manifest_artifact_id FROM training_dataset WHERE workspace_id=? AND id=?", "SELECT preparation_artifact_id FROM training_run WHERE workspace_id=? AND dataset_id=?", "SELECT manifest_artifact_id FROM model_version WHERE workspace_id=? AND dataset_id=?"} {
			v, e := s.evidenceArtifacts(ctx, tx, query, s.workspace, id)
			if e != nil {
				return e
			}
			artifacts = append(artifacts, v...)
		}
		if _, e = s.exec(ctx, tx, "UPDATE training_dataset SET manifest_artifact_id=NULL,options='{}',state='invalidated',invalidated_by=?,invalidated_revision=? WHERE workspace_id=? AND id=?", recording, revision, s.workspace, id); e != nil {
			return e
		}
		for _, query := range []string{"DELETE FROM dataset_member WHERE workspace_id=? AND dataset_id=?", "UPDATE training_run SET preparation_artifact_id=NULL,options='{}',state='invalidated' WHERE workspace_id=? AND dataset_id=?", "UPDATE model_version SET manifest_artifact_id=NULL,state='invalidated' WHERE workspace_id=? AND dataset_id=?"} {
			if _, e = s.exec(ctx, tx, query, s.workspace, id); e != nil {
				return e
			}
		}
		if e = s.queueEvidenceArtifacts(ctx, tx, recording, artifacts, revision); e != nil {
			return e
		}
	}
	return nil
}

func (s *Store) validateCorpusState(ctx context.Context, tx *sql.Tx, requireAll ...bool) error {
	for _, q := range []string{
		"SELECT count(*) FROM dataset_member m JOIN training_dataset d ON d.workspace_id=m.workspace_id AND d.id=m.dataset_id WHERE m.workspace_id=? AND d.state<>'current'",
		"SELECT count(*) FROM training_run r JOIN training_dataset d ON d.workspace_id=r.workspace_id AND d.id=r.dataset_id WHERE r.workspace_id=? AND r.state='current' AND d.state<>'current'",
		"SELECT count(*) FROM model_version v JOIN training_run r ON r.workspace_id=v.workspace_id AND r.id=v.run_id JOIN training_dataset d ON d.workspace_id=v.workspace_id AND d.id=v.dataset_id WHERE v.workspace_id=? AND v.state='current' AND (r.state<>'current' OR d.state<>'current')",
	} {
		var count int
		if e := s.row(ctx, tx, q, s.workspace).Scan(&count); e != nil {
			return e
		}
		if count != 0 {
			return contracts.Fail("invalid_request")
		}
	}
	rows, e := tx.QueryContext(ctx, s.query("SELECT revision,result FROM operation_receipt WHERE workspace_id=? AND result LIKE '%\"invalidated_dataset_ids\"%' ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	tombstones := map[string]int64{}
	for rows.Next() {
		var revision int64
		var result string
		if e = rows.Scan(&revision, &result); e != nil {
			break
		}
		var proof struct {
			IDs []string `json:"invalidated_dataset_ids"`
		}
		if json.Unmarshal([]byte(result), &proof) != nil {
			e = contracts.Fail("invalid_request")
			break
		}
		for _, id := range proof.IDs {
			if !contracts.ValidID(id) {
				e = contracts.Fail("invalid_request")
				break
			}
			tombstones[id] = revision
		}
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	complete := len(requireAll) > 0 && requireAll[0]
	if len(tombstones) > 0 || complete {
		rows, e = tx.QueryContext(ctx, s.query("SELECT id,state,invalidated_revision,manifest_artifact_id FROM training_dataset WHERE workspace_id=?"), s.workspace)
		if e != nil {
			return e
		}
		for rows.Next() {
			var id, state string
			var invalidated int64
			var manifest *string
			if e = rows.Scan(&id, &state, &invalidated, &manifest); e != nil {
				break
			}
			if revision, ok := tombstones[id]; ok {
				if state != "invalidated" || manifest != nil || invalidated != revision {
					e = contracts.Fail("invalid_request")
					break
				}
				delete(tombstones, id)
			} else if complete && state == "invalidated" {
				e = contracts.Fail("invalid_request")
				break
			}
		}
		err = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if err != nil {
			return err
		}
		if len(tombstones) > 0 {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

func (s *Store) invalidatedDatasetIDs(ctx context.Context, tx *sql.Tx, owner string, revision int64) ([]string, error) {
	return s.evidenceArtifacts(ctx, tx, "SELECT id FROM training_dataset WHERE workspace_id=? AND invalidated_by=? AND invalidated_revision=? AND state='invalidated' ORDER BY id", s.workspace, owner, revision)
}
