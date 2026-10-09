// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

// The workspace write transaction serializes publication admission and
// retirement. Expand only current reference arrays in SQL, keeping the guard
// bounded to one query instead of loading every publication into Go.
func (s *Store) libraryArtifactReference(ctx context.Context, tx *sql.Tx, artifact string) (bool, error) {
	libraryRefs := "json_each(l.report_publication_ids) AS refs"
	modelRefs := "json_each(m.publication_ids) AS refs"
	if s.backend == "postgresql" {
		libraryRefs = "jsonb_array_elements_text(CAST(l.report_publication_ids AS jsonb)) AS refs(value)"
		modelRefs = "jsonb_array_elements_text(CAST(m.publication_ids AS jsonb)) AS refs(value)"
	}
	q := "SELECT EXISTS (SELECT 1 FROM artifact_publication p WHERE p.workspace_id=? AND p.artifact_id=? AND (" +
		"EXISTS (SELECT 1 FROM library_entry l WHERE l.workspace_id=p.workspace_id AND (" +
		"l.original_publication_id=p.id OR l.subtitle_publication_id=p.id OR " +
		"EXISTS (SELECT 1 FROM " + libraryRefs + " WHERE refs.value=p.id))) OR " +
		"EXISTS (SELECT 1 FROM current_recording r WHERE r.workspace_id=p.workspace_id AND r.mapped_audio_publication_id=p.id) OR " +
		"EXISTS (SELECT 1 FROM base_model_install m WHERE m.workspace_id=p.workspace_id AND " +
		"EXISTS (SELECT 1 FROM " + modelRefs + " WHERE refs.value=p.id))))"
	var referenced bool
	e := s.row(ctx, tx, q, s.workspace, artifact).Scan(&referenced)
	return referenced, e
}
func libraryDigest(e LibraryEntry) string   { e.Revision = 0; d, _ := intent(e); return d }
func modelDigest(m BaseModelInstall) string { m.Revision = 0; d, _ := intent(m); return d }

type currentDomainProof struct {
	receiptID string
	revision  int64
	digest    string
}
type cleanupDomainProof struct {
	entryID   string
	revision  int64
	completed bool
}

// Keep historical receipts for idempotency, but bind mutable records to their
// newest accepted state. An older valid proof must not revive cancelled work or
// superseded metadata when later receipts remain in a portable snapshot.
func (s *Store) latestCurrentProofs(ctx context.Context, tx *sql.Tx) (map[string]currentDomainProof, map[string]currentDomainProof, map[string]currentDomainProof, map[string]cleanupDomainProof, error) {
	library, models, works := map[string]currentDomainProof{}, map[string]currentDomainProof{}, map[string]currentDomainProof{}
	cleanups := map[string]cleanupDomainProof{}
	rows, e := tx.QueryContext(ctx, s.query("SELECT id,revision,digest,result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return nil, nil, nil, nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var id, digest, result string
		var revision int64
		if e = rows.Scan(&id, &revision, &digest, &result); e != nil {
			return nil, nil, nil, nil, e
		}
		var envelope struct {
			CleanupEntryID string   `json:"cleanup_entry_id"`
			RecordingID    string   `json:"recording_id"`
			MediaID        string   `json:"media_id"`
			ModelID        string   `json:"model_id"`
			WorkID         string   `json:"work_id"`
			RecordDigest   string   `json:"record_digest"`
			LibraryDigest  string   `json:"library_digest"`
			WorkDigest     string   `json:"work_digest"`
			CleanupIDs     []string `json:"cleanup_ids"`
			PublicationID  string   `json:"publication_id"`
			State          string   `json:"state"`
		}
		if json.Unmarshal([]byte(result), &envelope) != nil {
			return nil, nil, nil, nil, contracts.Fail("invalid_request")
		}
		if envelope.LibraryDigest != "" {
			envelope.RecordDigest = envelope.LibraryDigest
		}
		for _, entity := range []struct {
			id     string
			proofs map[string]currentDomainProof
		}{{envelope.MediaID, library}, {envelope.ModelID, models}} {
			if entity.id != "" {
				if !contracts.ValidID(entity.id) || !digestPattern.MatchString(envelope.RecordDigest) {
					return nil, nil, nil, nil, contracts.Fail("invalid_request")
				}
				entity.proofs[entity.id] = currentDomainProof{id, revision, envelope.RecordDigest}
			}
		}
		if envelope.WorkID != "" {
			if !contracts.ValidID(envelope.WorkID) || !digestPattern.MatchString(envelope.WorkDigest) {
				return nil, nil, nil, nil, contracts.Fail("invalid_request")
			}
			works[envelope.WorkID] = currentDomainProof{id, revision, envelope.WorkDigest}
		}
		for _, publicationID := range envelope.CleanupIDs {
			entryID := envelope.CleanupEntryID
			if entryID == "" {
				entryID = envelope.RecordingID
			}
			if entryID == "" {
				entryID = envelope.MediaID
			}
			if entryID == "" {
				entryID = envelope.ModelID
			}
			if !contracts.ValidID(publicationID) || !contracts.ValidID(entryID) || cleanups[publicationID].completed {
				return nil, nil, nil, nil, contracts.Fail("invalid_request")
			}
			cleanups[publicationID] = cleanupDomainProof{entryID, revision, false}
		}
		if envelope.PublicationID != "" {
			proof, exists := cleanups[envelope.PublicationID]
			want, _ := intent([]any{"cleanup", envelope.PublicationID})
			if !exists || envelope.State != "done" || id != operationID("cleanup-done", envelope.PublicationID) || digest != want {
				return nil, nil, nil, nil, contracts.Fail("invalid_request")
			}
			proof.completed = true
			cleanups[envelope.PublicationID] = proof
		}
	}
	return library, models, works, cleanups, rows.Err()
}

func (s *Store) validateLibraryState(ctx context.Context, tx *sql.Tx, expire bool) error {
	r, e := s.readRecords(ctx, tx)
	if e != nil {
		return e
	}
	if e = s.validateRecordingState(ctx, tx, r); e != nil {
		return e
	}
	libraryProofs, modelProofs, workProofs, cleanupProofs, e := s.latestCurrentProofs(ctx, tx)
	if e != nil {
		return e
	}
	for _, entry := range r.Library {
		if !validLibrary(entry) || entry.Revision < 1 {
			return contracts.Fail("invalid_request")
		}
		proof, ok := libraryProofs[entry.ID]
		if !ok || proof.revision != entry.Revision || proof.digest != libraryDigest(entry) {
			return contracts.Fail("invalid_request")
		}
		delete(libraryProofs, entry.ID)
		if e = s.availablePublications(ctx, tx, entry.ReportPublicationIDs); e != nil {
			return e
		}
		if e = s.librarySource(ctx, tx, entry, false); e != nil {
			return e
		}
	}
	for _, m := range r.BaseModels {
		if !validModel(m) || m.Revision < 1 {
			return contracts.Fail("invalid_request")
		}
		proof, ok := modelProofs[m.ID]
		if !ok || proof.revision != m.Revision || proof.digest != modelDigest(m) {
			return contracts.Fail("invalid_request")
		}
		delete(modelProofs, m.ID)
		if e = s.modelPublicationIdentity(ctx, tx, m); e != nil {
			return e
		}
		if e = s.availablePublications(ctx, tx, m.PublicationIDs); e != nil {
			return e
		}
	}
	for _, w := range r.Works {
		if !validWork(w) {
			return contracts.Fail("invalid_request")
		}
		latest, ok := workProofs[w.ID]
		if !ok || latest.receiptID != w.JournalReceiptID || latest.digest != workJournalDigest(w) {
			return contracts.Fail("invalid_request")
		}
		delete(workProofs, w.ID)
		var initialDigest, result string
		if e = s.row(ctx, tx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.ID).Scan(&initialDigest); e != nil {
			return contracts.Fail("invalid_request")
		}
		if e = s.validateWorkInput(ctx, tx, w, initialDigest); e != nil {
			return e
		}
		if e = s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.JournalReceiptID).Scan(&result); e != nil {
			return contracts.Fail("invalid_request")
		}
		var proof map[string]any
		if strict([]byte(result), &proof) != nil || proof["work_id"] != w.ID || proof["work_digest"] != workJournalDigest(w) {
			return contracts.Fail("invalid_request")
		}
		if expire {
			w.LeaseUntil = 0
			if e = s.putDomain(ctx, tx, "Works", w); e != nil {
				return e
			}
		}
	}
	if len(libraryProofs)+len(modelProofs)+len(workProofs) != 0 {
		return contracts.Fail("invalid_request")
	}
	for _, c := range r.Cleanups {
		latest, ok := cleanupProofs[c.ID]
		if !ok || latest.entryID != c.EntryID || latest.revision != c.Revision || latest.completed != (c.State == "done") {
			return contracts.Fail("invalid_request")
		}
		delete(cleanupProofs, c.ID)
		var result string
		if e = s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND revision=?", s.workspace, c.Revision).Scan(&result); e != nil {
			return contracts.Fail("invalid_request")
		}
		var proof struct {
			CleanupEntryID string   `json:"cleanup_entry_id"`
			RecordingID    string   `json:"recording_id"`
			MediaID        string   `json:"media_id"`
			ModelID        string   `json:"model_id"`
			CleanupIDs     []string `json:"cleanup_ids"`
		}
		if json.Unmarshal([]byte(result), &proof) != nil || (proof.CleanupEntryID != c.EntryID && proof.RecordingID != c.EntryID && proof.MediaID != c.EntryID && proof.ModelID != c.EntryID) {
			return contracts.Fail("invalid_request")
		}
		bound := false
		for _, id := range proof.CleanupIDs {
			if id == c.ID {
				bound = true
			}
		}
		if !bound {
			return contracts.Fail("invalid_request")
		}
		p, err := s.publicationTx(ctx, tx, c.ID)
		missing := err == sql.ErrNoRows
		if typed, ok := err.(*contracts.Error); ok && typed.Code == "not_found" {
			missing = true
		}
		if missing && c.LegacyLocationID != nil && c.State == "pending" {
			if e := s.validateLegacyCleanup(ctx, tx, c); e != nil {
				return e
			}
			continue
		}
		if err != nil {
			return contracts.Fail("invalid_request")
		}
		if c.State == "done" && p.State != "retired" && !(c.LegacyLocationID == nil && p.State == "aborted" && (p.Kind == "mapped-audio" || p.Kind == "speaker-clip" || p.Kind == "derived-manifest")) {
			return contracts.Fail("invalid_request")
		}
		var n int
		if e = s.row(ctx, tx, "SELECT (SELECT count(*) FROM library_entry WHERE workspace_id=? AND id=?)+(SELECT count(*) FROM base_model_install WHERE workspace_id=? AND id=?)", s.workspace, c.EntryID, s.workspace, c.EntryID).Scan(&n); e != nil {
			return e
		}
		if n < 1 && !(c.EntryID == s.workspace && proof.CleanupEntryID == s.workspace) {
			return contracts.Fail("invalid_request")
		}
	}
	if len(cleanupProofs) != 0 {
		return contracts.Fail("invalid_request")
	}
	return nil
}

func (s *Store) modelPublicationIdentity(ctx context.Context, tx *sql.Tx, m BaseModelInstall) error {
	var manifest struct {
		Kind         string `json:"kind"`
		Version      string `json:"schema_version"`
		Name         string `json:"name"`
		ModelVersion string `json:"model_version"`
		Files        []struct {
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"files"`
	}
	if json.Unmarshal(m.Manifest, &manifest) != nil || manifest.Kind != "base-model-manifest" || manifest.Version != contracts.Version || manifest.Name != m.Name || manifest.ModelVersion != m.Version || len(manifest.Files) == 0 {
		return contracts.Fail("invalid_request")
	}
	ids, e := PublicationIDs(m.PublicationIDs)
	if e != nil {
		return e
	}
	if m.State == "registered" {
		if len(ids) != 0 {
			return contracts.Fail("invalid_request")
		}
		return nil
	}
	if len(ids) != len(manifest.Files) {
		return contracts.Fail("invalid_request")
	}
	for i, id := range ids {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		if p.State != "available" || p.Digest != manifest.Files[i].SHA256 || p.Size != manifest.Files[i].Size {
			return contracts.Fail("conflict")
		}
	}
	return nil
}

func (s *Store) currentReceiptResult(ctx context.Context, tx *sql.Tx, key, id string, revision int64, digest string) (map[string]any, error) {
	rows, e := tx.QueryContext(ctx, s.query("SELECT id FROM library_cleanup WHERE workspace_id=? AND entry_id=? AND revision=? ORDER BY id"), s.workspace, id, revision)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var value string
		if e = rows.Scan(&value); e != nil {
			break
		}
		ids = append(ids, value)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if err != nil {
		return nil, err
	}
	result := map[string]any{key: id, "revision": revision, "record_digest": digest, "cleanup_ids": ids}
	invalidated, e := s.invalidatedDatasetIDs(ctx, tx, id, revision)
	if e != nil {
		return nil, e
	}
	if len(invalidated) > 0 {
		result["invalidated_dataset_ids"] = invalidated
	}
	return result, nil
}
