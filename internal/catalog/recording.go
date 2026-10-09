// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
)

// Derived metadata contains references and diagnostics, never another turn list.
func referenceMetadata(raw json.RawMessage) bool {
	if !nonsecret(raw) {
		return false
	}
	var value any
	if strict(raw, &value) != nil {
		return false
	}
	var visit func(any) bool
	visit = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				switch k {
				case "speaker_attributions", "assignments", "turns", "cues", "document":
					return false
				}
				if !visit(v) {
					return false
				}
			}
		case []any:
			for _, v := range x {
				if !visit(v) {
					return false
				}
			}
		}
		return true
	}
	return visit(value)
}
func validRecording(r Recording) bool {
	if !contracts.ValidID(r.ID) || r.SourceRevision < 1 || !digestPattern.MatchString(r.SourceDigest) || !referenceMetadata(r.SourceMap) || !referenceMetadata(r.Provenance) || !referenceMetadata(r.Diagnostics) || r.Revision < 0 {
		return false
	}
	if r.MappedAudioPublicationID != nil && !contracts.ValidID(*r.MappedAudioPublicationID) {
		return false
	}
	null := bytes.Equal(bytes.TrimSpace(r.Document), []byte("null"))
	if r.State == "untranscribed" || r.State == "no-speech" || r.State == "no-timed-subtitles" {
		return null && r.DocumentDigest == ""
	}
	if r.State != "ready" || null || !validJSON(r.Document) || !digestPattern.MatchString(r.DocumentDigest) || hash(r.Document) != r.DocumentDigest {
		return false
	}
	return subtitles.ValidateDocument(r.Document) == nil
}
func validSpeakerMapping(m SpeakerMapping) bool {
	return contracts.ValidID(m.ID) && contracts.ValidID(m.RecordingID) && contracts.ValidLocalSpeakerID(m.LocalSpeakerID) && contracts.ValidID(m.SpeakerID) && digestPattern.MatchString(m.DocumentDigest) && m.Revision > 0
}
func recordingDigest(r Recording) string { r.Revision = 0; d, _ := intent(r); return d }
func (s *Store) Recording(ctx context.Context, id string) (Recording, error) {
	var r Recording
	e := s.readOne(ctx, "Recordings", id, &r)
	return r, e
}
func (s *Store) SpeakerMappings(ctx context.Context, id string) ([]SpeakerMapping, error) {
	if !contracts.ValidID(id) {
		return nil, contracts.Fail("invalid_request")
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, sanitize(e)
	}
	defer tx.Rollback()
	out, e := s.recordingMappings(ctx, tx, id)
	return out, sanitize(e)
}
func (s *Store) recordingMappings(ctx context.Context, tx *sql.Tx, id string) ([]SpeakerMapping, error) {
	d := domainTable{}
	for _, candidate := range domains {
		if candidate.field == "SpeakerMappings" {
			d = candidate
		}
	}
	rows, e := tx.QueryContext(ctx, s.query("SELECT "+joinNames(d)+" FROM speaker_mapping WHERE workspace_id=? AND recording_id=? ORDER BY id"), s.workspace, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []SpeakerMapping{}
	for rows.Next() {
		var m SpeakerMapping
		if e = rows.Scan(&m.ID, &m.RecordingID, &m.LocalSpeakerID, &m.SpeakerID, &m.DocumentDigest, &m.Revision); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func joinNames(d domainTable) string {
	cols := names(columns(d.typ()))
	var b bytes.Buffer
	for i, c := range cols {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(c)
	}
	return b.String()
}
func documentSpeakers(raw json.RawMessage) map[string]bool {
	out := map[string]bool{}
	var doc struct {
		Cues []struct {
			Attributions []struct {
				SpeakerID string `json:"speaker_id"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return out
	}
	for _, cue := range doc.Cues {
		for _, a := range cue.Attributions {
			out[a.SpeakerID] = true
		}
	}
	return out
}
func publicationList(id *string) json.RawMessage {
	out := []string{}
	if id != nil {
		out = append(out, *id)
	}
	raw, _ := json.Marshal(out)
	return raw
}
func (s *Store) validateRecordingSource(ctx context.Context, tx *sql.Tx, r Recording, exact bool) error {
	var entry LibraryEntry
	if e := s.domainTx(ctx, tx, "Library", r.ID, &entry); e != nil {
		return e
	}
	if entry.Digest != r.SourceDigest || (exact && entry.Revision != r.SourceRevision) || r.SourceRevision > entry.Revision {
		return contracts.Fail("conflict")
	}
	var importMap struct {
		Policy   string `json:"policy"`
		Supplied bool   `json:"supplied_document"`
	}
	json.Unmarshal(r.SourceMap, &importMap)
	// A supplied document owns its declared timing. Bind source identity without
	// replacing its media_timing or rejecting native cues outside measured audio.
	// Engine-produced mapped recordings retain the stricter assembly contract.
	if r.State == "ready" && importMap.Policy != "supplied-document-source-clock;no-retiming" && !importMap.Supplied {
		var doc struct {
			MediaTiming *struct {
				DurationMS *int64 `json:"duration_milliseconds"`
			} `json:"media_timing"`
			Cues []struct {
				Timing struct {
					EndMS int64 `json:"end_milliseconds"`
				} `json:"timing"`
			} `json:"cues"`
		}
		if json.Unmarshal(r.Document, &doc) != nil {
			return contracts.Fail("invalid_request")
		}
		if entry.DurationUS == nil && doc.MediaTiming != nil {
			return contracts.Fail("invalid_request")
		}
		if entry.DurationUS != nil {
			ms := *entry.DurationUS / 1000
			if doc.MediaTiming == nil || doc.MediaTiming.DurationMS == nil || *doc.MediaTiming.DurationMS != ms {
				return contracts.Fail("invalid_request")
			}
			for _, cue := range doc.Cues {
				if cue.Timing.EndMS > ms {
					return contracts.Fail("invalid_request")
				}
			}
		}
	}
	return s.availablePublications(ctx, tx, publicationList(r.MappedAudioPublicationID))
}
func (s *Store) saveRecording(ctx context.Context, tx *sql.Tx, expected int64, r *Recording, rev int64) error {
	var old Recording
	e := s.domainTx(ctx, tx, "Recordings", r.ID, &old)
	if e != nil && e != sql.ErrNoRows {
		return e
	}
	if (e == nil && old.Revision != expected) || (e == sql.ErrNoRows && expected != 0) {
		return contracts.Fail("conflict")
	}
	if e = s.validateRecordingSource(ctx, tx, *r, true); e != nil {
		return e
	}
	if e = s.queueRemoved(ctx, tx, r.ID, publicationList(old.MappedAudioPublicationID), publicationList(r.MappedAudioPublicationID), rev+1); e != nil {
		return e
	}
	r.Revision = rev + 1
	if e = s.putDomain(ctx, tx, "Recordings", *r); e != nil {
		return e
	}
	mappings, e := s.recordingMappings(ctx, tx, r.ID)
	if e != nil {
		return e
	}
	present := documentSpeakers(r.Document)
	for _, m := range mappings {
		if !present[m.LocalSpeakerID] {
			if _, e = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND id=?", s.workspace, m.ID); e != nil {
				return e
			}
		} else {
			m.DocumentDigest = r.DocumentDigest
			m.Revision = r.Revision
			if e = s.putDomain(ctx, tx, "SpeakerMappings", m); e != nil {
				return e
			}
		}
	}
	if e = s.reconcileRecordingEvidence(ctx, tx, old, *r, rev+1); e != nil {
		return e
	}
	return nil
}

func (s *Store) CommitRecording(ctx context.Context, claim Work, expected int64, r Recording) (Recording, error) {
	if expected < 0 || !validRecording(r) {
		return r, contracts.Fail("invalid_request")
	}
	r.Revision = 0
	digest, e := intent([]any{"current-recording", expected, r})
	if e != nil {
		return r, e
	}
	op := operationID("recording-current", claim.ID, r.ID)
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, e := s.workAuthority(ctx, tx, claim); e != nil {
			return e
		}
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Recordings", r.ID, &r)
		}
		if e = s.saveRecording(ctx, tx, expected, &r, rev); e != nil {
			return e
		}
		result, e := s.currentReceiptResult(ctx, tx, "recording_id", r.ID, r.Revision, recordingDigest(r))
		if e != nil {
			return e
		}
		mappings, e := s.recordingMappings(ctx, tx, r.ID)
		if e != nil {
			return e
		}
		result["mapping_digest"], e = intent(mappings)
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, result)
		return e
	})
	return r, e
}
func (s *Store) SetSpeakerMapping(ctx context.Context, op string, expected int64, m SpeakerMapping) (SpeakerMapping, error) {
	if !contracts.ValidID(op) || !contracts.ValidID(m.RecordingID) || !contracts.ValidLocalSpeakerID(m.LocalSpeakerID) || !contracts.ValidID(m.SpeakerID) || expected < 1 {
		return m, contracts.Fail("invalid_request")
	}
	m.ID = operationID("speaker-mapping", m.RecordingID, m.LocalSpeakerID)
	expectedMapping := m.Revision
	if expectedMapping < 0 {
		return m, contracts.Fail("invalid_request")
	}
	m.Revision = 0
	digest, e := intent([]any{"speaker-mapping", expected, expectedMapping, m})
	if e != nil {
		return m, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "SpeakerMappings", m.ID, &m)
		}
		var r Recording
		if e = s.domainTx(ctx, tx, "Recordings", m.RecordingID, &r); e != nil {
			return e
		}
		if r.Revision != expected || m.DocumentDigest != r.DocumentDigest || !documentSpeakers(r.Document)[m.LocalSpeakerID] {
			return contracts.Fail("conflict")
		}
		var existing SpeakerMapping
		e = s.domainTx(ctx, tx, "SpeakerMappings", m.ID, &existing)
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		if (e == sql.ErrNoRows && expectedMapping != 0) || (e == nil && existing.Revision != expectedMapping) {
			return contracts.Fail("conflict")
		}
		m.Revision = rev + 1
		if e = s.putDomain(ctx, tx, "SpeakerMappings", m); e != nil {
			return e
		}
		if e = s.reconcileRecordingEvidence(ctx, tx, r, r, rev+1); e != nil {
			return e
		}
		mappings, e := s.recordingMappings(ctx, tx, m.RecordingID)
		if e != nil {
			return e
		}
		md, e := intent(mappings)
		if e != nil {
			return e
		}
		result, e := s.currentReceiptResult(ctx, tx, "mapping_recording_id", m.RecordingID, rev+1, recordingDigest(r))
		if e != nil {
			return e
		}
		result["mapping_digest"] = md
		result["cleanup_entry_id"] = m.RecordingID
		_, e = s.accept(ctx, tx, op, digest, rev, result)
		return e
	})
	return m, e
}

// Latest accepted receipts bind both current document and external correlations.
// Historical receipts contain hashes/identities only and cannot revive old data.
func (s *Store) validateRecordingState(ctx context.Context, tx *sql.Tx, records Records) error {
	proofs := map[string]currentDomainProof{}
	mappingProofs := map[string]string{}
	rows, e := tx.QueryContext(ctx, s.query("SELECT id,revision,result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var id, result string
		var rev int64
		if e = rows.Scan(&id, &rev, &result); e != nil {
			rows.Close()
			return e
		}
		var p struct {
			RecordingID        string `json:"recording_id"`
			MappingRecordingID string `json:"mapping_recording_id"`
			RecordDigest       string `json:"record_digest"`
			RecordingDigest    string `json:"recording_digest"`
			MappingDigest      string `json:"mapping_digest"`
		}
		if json.Unmarshal([]byte(result), &p) != nil {
			rows.Close()
			return contracts.Fail("invalid_request")
		}
		if p.RecordingDigest != "" {
			p.RecordDigest = p.RecordingDigest
		}
		if p.RecordingID != "" {
			if !contracts.ValidID(p.RecordingID) || !digestPattern.MatchString(p.RecordDigest) || !digestPattern.MatchString(p.MappingDigest) {
				rows.Close()
				return contracts.Fail("invalid_request")
			}
			proofs[p.RecordingID] = currentDomainProof{id, rev, p.RecordDigest}
			mappingProofs[p.RecordingID] = p.MappingDigest
		}
		if p.MappingRecordingID != "" {
			if !contracts.ValidID(p.MappingRecordingID) || !digestPattern.MatchString(p.MappingDigest) {
				rows.Close()
				return contracts.Fail("invalid_request")
			}
			mappingProofs[p.MappingRecordingID] = p.MappingDigest
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, r := range records.Recordings {
		proof, ok := proofs[r.ID]
		if !ok || !validRecording(r) || proof.revision != r.Revision || proof.digest != recordingDigest(r) {
			return contracts.Fail("invalid_request")
		}
		delete(proofs, r.ID)
		if e = s.validateRecordingSource(ctx, tx, r, false); e != nil {
			return e
		}
		mappings, e := s.recordingMappings(ctx, tx, r.ID)
		if e != nil {
			return e
		}
		md, e := intent(mappings)
		if e != nil || mappingProofs[r.ID] != md {
			return contracts.Fail("invalid_request")
		}
		delete(mappingProofs, r.ID)
		present := documentSpeakers(r.Document)
		for _, m := range mappings {
			if !validSpeakerMapping(m) || m.DocumentDigest != r.DocumentDigest || !present[m.LocalSpeakerID] {
				return contracts.Fail("invalid_request")
			}
		}
	}
	if len(proofs)+len(mappingProofs) != 0 {
		return contracts.Fail("invalid_request")
	}
	return nil
}

// AcceptedRecordingWork reconciles a commit that reached the catalog before the
// worker checkpoint. Recovery does not repeat inference after accepted effects.
func (s *Store) AcceptedRecordingWork(ctx context.Context, workID, recordingID string) (json.RawMessage, bool, error) {
	if !contracts.ValidID(workID) || !contracts.ValidID(recordingID) {
		return nil, false, contracts.Fail("invalid_request")
	}
	var result string
	e := s.db.QueryRowContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, operationID("recording-current", workID, recordingID)).Scan(&result)
	if e == sql.ErrNoRows {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, sanitize(e)
	}
	return json.RawMessage(result), true, nil
}

// QueueDerivedCleanup records recoverable physical retirement of an unreferenced
// processing candidate. Current/source reference fences still govern retirement.
func (s *Store) QueueDerivedCleanup(ctx context.Context, op, entryID, publicationID string) error {
	if !contracts.ValidID(op) || !contracts.ValidID(entryID) || !contracts.ValidID(publicationID) {
		return contracts.Fail("invalid_request")
	}
	digest, _ := intent([]any{"derived-cleanup", entryID, publicationID})
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil || ok {
			return e
		}
		var entry LibraryEntry
		if e = s.domainTx(ctx, tx, "Library", entryID, &entry); e != nil {
			return e
		}
		p, e := s.publicationTx(ctx, tx, publicationID)
		if typed, ok := e.(*contracts.Error); ok && typed.Code == "not_found" {
			return nil
		}
		if e != nil {
			return e
		}
		if p.Kind != "mapped-audio" && p.Kind != "speaker-clip" && p.Kind != "derived-manifest" {
			return contracts.Fail("invalid_request")
		}
		referenced, e := s.libraryArtifactReference(ctx, tx, p.ArtifactID)
		if e != nil {
			return e
		}
		if referenced {
			return contracts.Fail("conflict")
		}
		var old Cleanup
		if e = s.domainTx(ctx, tx, "Cleanups", publicationID, &old); e == nil {
			return nil
		} else if e != sql.ErrNoRows {
			return e
		}
		if e = s.putDomain(ctx, tx, "Cleanups", Cleanup{ID: publicationID, EntryID: entryID, Revision: rev + 1, State: "pending"}); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"cleanup_entry_id": entryID, "cleanup_ids": []string{publicationID}})
		return e
	})
}
