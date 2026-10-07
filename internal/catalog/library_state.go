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
		"EXISTS (SELECT 1 FROM base_model_install m WHERE m.workspace_id=p.workspace_id AND " +
		"EXISTS (SELECT 1 FROM " + modelRefs + " WHERE refs.value=p.id))))"
	var referenced bool
	e := s.row(ctx, tx, q, s.workspace, artifact).Scan(&referenced)
	return referenced, e
}
func (s *Store) recordProof(ctx context.Context, tx *sql.Tx, revision int64, id, key, digest string) error {
	var result string
	if e := s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND revision=?", s.workspace, revision).Scan(&result); e != nil {
		return contracts.Fail("invalid_request")
	}
	var value map[string]any
	if strict([]byte(result), &value) != nil || value[key] != id || value["record_digest"] != digest {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func libraryDigest(e LibraryEntry) string   { e.Revision = 0; d, _ := intent(e); return d }
func modelDigest(m BaseModelInstall) string { m.Revision = 0; d, _ := intent(m); return d }
func (s *Store) validateLibraryState(ctx context.Context, tx *sql.Tx, expire bool) error {
	r, e := s.readRecords(ctx, tx)
	if e != nil {
		return e
	}
	for _, entry := range r.Library {
		if !validLibrary(entry) || entry.Revision < 1 {
			return contracts.Fail("invalid_request")
		}
		if e = s.recordProof(ctx, tx, entry.Revision, entry.ID, "media_id", libraryDigest(entry)); e != nil {
			return e
		}
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
		if e = s.recordProof(ctx, tx, m.Revision, m.ID, "model_id", modelDigest(m)); e != nil {
			return e
		}
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
		var initialDigest, result string
		if e = s.row(ctx, tx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.ID).Scan(&initialDigest); e != nil {
			return contracts.Fail("invalid_request")
		}
		want, _ := intent([]any{"work", w.Kind, w.Payload})
		if initialDigest != want {
			return contracts.Fail("invalid_request")
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
	for _, c := range r.Cleanups {
		var result string
		if e = s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND revision=?", s.workspace, c.Revision).Scan(&result); e != nil {
			return contracts.Fail("invalid_request")
		}
		var proof struct {
			MediaID    string   `json:"media_id"`
			ModelID    string   `json:"model_id"`
			CleanupIDs []string `json:"cleanup_ids"`
		}
		if json.Unmarshal([]byte(result), &proof) != nil || (proof.MediaID != c.EntryID && proof.ModelID != c.EntryID) {
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
		if err != nil {
			return contracts.Fail("invalid_request")
		}
		if c.State == "done" && p.State != "retired" {
			return contracts.Fail("invalid_request")
		}
		var n int
		if e = s.row(ctx, tx, "SELECT (SELECT count(*) FROM library_entry WHERE workspace_id=? AND id=?)+(SELECT count(*) FROM base_model_install WHERE workspace_id=? AND id=?)", s.workspace, c.EntryID, s.workspace, c.EntryID).Scan(&n); e != nil {
			return e
		}
		if n < 1 {
			return contracts.Fail("invalid_request")
		}
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
	return map[string]any{key: id, "revision": revision, "record_digest": digest, "cleanup_ids": ids}, nil
}
