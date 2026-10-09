// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

func PublicationIDs(raw json.RawMessage) ([]string, error) {
	var out []string
	if strict(raw, &out) != nil || out == nil {
		return nil, contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for _, id := range out {
		if !contracts.ValidID(id) || seen[id] {
			return nil, contracts.Fail("invalid_request")
		}
		seen[id] = true
	}
	return out, nil
}
func validLibrary(e LibraryEntry) bool {
	_, err := PublicationIDs(e.ReportPublicationIDs)
	return err == nil && contracts.ValidID(e.ID) && contracts.ValidID(e.AssetID) && digestPattern.MatchString(e.Digest) && e.Size >= 0 && (e.DurationUS == nil || *e.DurationUS >= 0) && (e.Mode == "copy" || e.Mode == "reference") && e.Title != "" && (e.Class == "audio" || e.Class == "video" || e.Class == "unknown") && nonsecret(e.Facts) && validJSON(e.Metadata) && validJSON(e.Dates) && (e.OriginalPublicationID == nil || contracts.ValidID(*e.OriginalPublicationID)) && (e.SubtitlePublicationID == nil || contracts.ValidID(*e.SubtitlePublicationID))
}
func validModel(m BaseModelInstall) bool {
	_, e := PublicationIDs(m.PublicationIDs)
	return e == nil && contracts.ValidID(m.ID) && m.Name != "" && m.Version != "" && digestPattern.MatchString(m.Digest) && hash(m.Manifest) == m.Digest && nonsecret(m.Manifest) && (m.State == "registered" || m.State == "available")
}
func (s *Store) Library(ctx context.Context, id string) (LibraryEntry, error) {
	var out LibraryEntry
	e := s.readOne(ctx, "Library", id, &out)
	return out, e
}
func (s *Store) Libraries(ctx context.Context) ([]LibraryEntry, error) {
	r, e := s.currentRecords(ctx, "Library")
	return r.Library, e
}
func (s *Store) BaseModel(ctx context.Context, id string) (BaseModelInstall, error) {
	var out BaseModelInstall
	e := s.readOne(ctx, "BaseModels", id, &out)
	return out, e
}
func (s *Store) BaseModels(ctx context.Context) ([]BaseModelInstall, error) {
	r, e := s.currentRecords(ctx, "BaseModels")
	return r.BaseModels, e
}
func (s *Store) Cleanups(ctx context.Context) ([]Cleanup, error) {
	r, e := s.currentRecords(ctx, "Cleanups")
	return r.Cleanups, e
}
func (s *Store) availablePublications(ctx context.Context, tx *sql.Tx, raw json.RawMessage) error {
	ids, e := PublicationIDs(raw)
	if e != nil {
		return e
	}
	for _, id := range ids {
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		if p.State != "available" {
			return contracts.Fail("conflict")
		}
	}
	return nil
}
func (s *Store) queueRemoved(ctx context.Context, tx *sql.Tx, entry string, old, new json.RawMessage, revision int64) error {
	a, e := PublicationIDs(old)
	if e != nil {
		return e
	}
	b, e := PublicationIDs(new)
	if e != nil {
		return e
	}
	keep := map[string]bool{}
	for _, id := range b {
		keep[id] = true
	}
	for _, id := range a {
		if !keep[id] {
			if e = s.putDomain(ctx, tx, "Cleanups", Cleanup{ID: id, EntryID: entry, State: "pending", Revision: revision}); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Store) librarySource(ctx context.Context, tx *sql.Tx, e LibraryEntry, create bool) error {
	artifactID := e.AssetID
	if e.OriginalPublicationID != nil {
		p, err := s.publicationTx(ctx, tx, *e.OriginalPublicationID)
		if err != nil {
			return err
		}
		if p.State != "available" || p.Digest != e.Digest || p.Size != e.Size {
			return contracts.Fail("conflict")
		}
		artifactID = p.ArtifactID
	} else if e.Mode != "reference" {
		return contracts.Fail("invalid_request")
	}
	if e.SubtitlePublicationID != nil {
		p, err := s.publicationTx(ctx, tx, *e.SubtitlePublicationID)
		if err != nil {
			return err
		}
		if p.State != "available" {
			return contracts.Fail("conflict")
		}
	}
	var n int
	if err := s.row(ctx, tx, "SELECT count(*) FROM artifact WHERE workspace_id=? AND id=?", s.workspace, artifactID).Scan(&n); err != nil {
		return err
	}
	rows := Records{}
	if e.SubtitlePublicationID != nil {
		p, err := s.publicationTx(ctx, tx, *e.SubtitlePublicationID)
		if err != nil {
			return err
		}
		var count int
		if err = s.row(ctx, tx, "SELECT count(*) FROM media_asset WHERE workspace_id=? AND id=?", s.workspace, p.ID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			rows.Assets = append(rows.Assets, Asset{ID: p.ID, ArtifactID: p.ArtifactID, Role: "subtitle-source", DurationUS: 0})
		} else {
			var artifact string
			if err = s.row(ctx, tx, "SELECT artifact_id FROM media_asset WHERE workspace_id=? AND id=?", s.workspace, p.ID).Scan(&artifact); err != nil {
				return err
			}
			if artifact != p.ArtifactID {
				return contracts.Fail("conflict")
			}
		}
	}
	if n == 0 {
		rows.Artifacts = []Artifact{{ID: artifactID, Digest: e.Digest, Size: e.Size, Kind: "source"}}
	} else {
		var digest string
		var size int64
		if err := s.row(ctx, tx, "SELECT digest,size FROM artifact WHERE workspace_id=? AND id=?", s.workspace, artifactID).Scan(&digest, &size); err != nil {
			return err
		}
		if digest != e.Digest || size != e.Size {
			return contracts.Fail("conflict")
		}
	}
	if err := s.row(ctx, tx, "SELECT count(*) FROM media_asset WHERE workspace_id=? AND id=?", s.workspace, e.AssetID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		duration := int64(0)
		if e.DurationUS != nil {
			duration = *e.DurationUS
		}
		rows.Assets = append(rows.Assets, Asset{ID: e.AssetID, ArtifactID: artifactID, Role: "original", DurationUS: duration})
	} else {
		var original string
		if err := s.row(ctx, tx, "SELECT artifact_id FROM media_asset WHERE workspace_id=? AND id=?", s.workspace, e.AssetID).Scan(&original); err != nil {
			return err
		}
		if original != artifactID {
			return contracts.Fail("conflict")
		}
	}
	if err := s.row(ctx, tx, "SELECT count(*) FROM media_entry WHERE workspace_id=? AND id=?", s.workspace, e.ID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		rows.Media = []Media{{ID: e.ID, Title: e.Title, Class: e.Class}}
		rows.MediaAssets = []MediaAsset{{ID: e.ID, MediaID: e.ID, AssetID: e.AssetID, Role: "original"}}
	}
	if !create && len(rows.Artifacts)+len(rows.Assets)+len(rows.Media)+len(rows.MediaAssets) > 0 {
		return contracts.Fail("invalid_request")
	}
	return s.insertRecords(ctx, tx, rows)
}
func (s *Store) saveLibrary(ctx context.Context, tx *sql.Tx, entry *LibraryEntry, expected, rev int64) error {
	return s.saveElectedLibrary(ctx, tx, entry, expected, rev, false)
}
func (s *Store) saveElectedLibrary(ctx context.Context, tx *sql.Tx, entry *LibraryEntry, expected, rev int64, replace bool) error {
	if !validLibrary(*entry) {
		return contracts.Fail("invalid_request")
	}
	var old LibraryEntry
	err := s.domainTx(ctx, tx, "Library", entry.ID, &old)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		if old.Revision != expected || !replace && (old.AssetID != entry.AssetID || old.Digest != entry.Digest || old.Size != entry.Size || old.Mode != entry.Mode) {
			return contracts.Fail("conflict")
		}
		if e := s.queueRemoved(ctx, tx, entry.ID, old.ReportPublicationIDs, entry.ReportPublicationIDs, rev+1); e != nil {
			return e
		}
	} else if expected != 0 {
		return contracts.Fail("conflict")
	}
	if e := s.availablePublications(ctx, tx, entry.ReportPublicationIDs); e != nil {
		return e
	}
	if e := s.librarySource(ctx, tx, *entry, true); e != nil {
		return e
	}
	if e := s.removeLegacyMetadata(ctx, tx, *entry, rev+1); e != nil {
		return e
	}
	entry.Revision = rev + 1
	if e := s.putDomain(ctx, tx, "Library", *entry); e != nil {
		return e
	}
	if replace {
		if _, e := s.exec(ctx, tx, "UPDATE media_entry_asset SET asset_id=? WHERE workspace_id=? AND media_id=? AND role='original'", entry.AssetID, s.workspace, entry.ID); e != nil {
			return e
		}
		if e := s.queueRemoved(ctx, tx, entry.ID, publicationList(old.OriginalPublicationID), publicationList(entry.OriginalPublicationID), rev+1); e != nil {
			return e
		}
		if old.AssetID != entry.AssetID {
			var owners int
			if e := s.row(ctx, tx, "SELECT count(*) FROM media_entry_asset WHERE workspace_id=? AND asset_id=?", s.workspace, old.AssetID).Scan(&owners); e != nil {
				return e
			}
			if owners == 0 {
				if e := s.removeLegacyMetadata(ctx, tx, old, rev+1); e != nil {
					return e
				}
				if _, e := s.exec(ctx, tx, "DELETE FROM media_asset WHERE workspace_id=? AND id=? AND NOT EXISTS(SELECT 1 FROM library_entry WHERE workspace_id=? AND asset_id=?)", s.workspace, old.AssetID, s.workspace, old.AssetID); e != nil {
					return e
				}
			}
		}
	}
	var current Recording
	if e := s.domainTx(ctx, tx, "Recordings", entry.ID, &current); e == nil && !replace {
		if e = s.validateRecordingSource(ctx, tx, current, false); e != nil {
			return e
		}
	} else if e != nil && e != sql.ErrNoRows {
		return e
	}
	_, e := s.exec(ctx, tx, "UPDATE media_entry SET title=?,class=? WHERE workspace_id=? AND id=?", entry.Title, entry.Class, s.workspace, entry.ID)
	return e
}
func (s *Store) CommitLibrary(ctx context.Context, claim Work, entry LibraryEntry) (LibraryEntry, error) {
	expected := entry.Revision
	entry.Revision = 0
	digest, e := intent(entry)
	if e != nil {
		return entry, e
	}
	op := operationID("library-current", claim.ID, entry.ID)
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, e := s.workAuthority(ctx, tx, claim); e != nil {
			return e
		}
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Library", entry.ID, &entry)
		}
		if e = s.saveLibrary(ctx, tx, &entry, expected, rev); e != nil {
			return e
		}
		result, e := s.currentReceiptResult(ctx, tx, "media_id", entry.ID, entry.Revision, libraryDigest(entry))
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, result)
		return e
	})
	return entry, e
}
func (s *Store) UpdateLibrary(ctx context.Context, op string, expected int64, entry LibraryEntry) (LibraryEntry, error) {
	if !contracts.ValidID(op) || expected < 0 {
		return entry, contracts.Fail("invalid_request")
	}
	entry.Revision = 0
	digest, e := intent([]any{"update-library", expected, entry})
	if e != nil {
		return entry, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Library", entry.ID, &entry)
		}
		if e = s.saveLibrary(ctx, tx, &entry, expected, rev); e != nil {
			return e
		}
		result, e := s.currentReceiptResult(ctx, tx, "media_id", entry.ID, entry.Revision, libraryDigest(entry))
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, result)
		return e
	})
	return entry, e
}
func (s *Store) CommitBaseModel(ctx context.Context, claim Work, m BaseModelInstall) (BaseModelInstall, error) {
	expected := m.Revision
	m.Revision = 0
	if !validModel(m) {
		return m, contracts.Fail("invalid_request")
	}
	digest, e := intent(m)
	if e != nil {
		return m, e
	}
	op := operationID("base-model-current", claim.ID, m.ID)
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, e := s.workAuthority(ctx, tx, claim); e != nil {
			return e
		}
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "BaseModels", m.ID, &m)
		}
		var old BaseModelInstall
		e = s.domainTx(ctx, tx, "BaseModels", m.ID, &old)
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		if e == nil {
			if old.Revision != expected {
				return contracts.Fail("conflict")
			}
			if e = s.queueRemoved(ctx, tx, m.ID, old.PublicationIDs, m.PublicationIDs, rev+1); e != nil {
				return e
			}
		} else if expected != 0 {
			return contracts.Fail("conflict")
		}
		if e = s.modelPublicationIdentity(ctx, tx, m); e != nil {
			return e
		}
		if e = s.availablePublications(ctx, tx, m.PublicationIDs); e != nil {
			return e
		}
		m.Revision = rev + 1
		if e = s.putDomain(ctx, tx, "BaseModels", m); e != nil {
			return e
		}
		result, e := s.currentReceiptResult(ctx, tx, "model_id", m.ID, m.Revision, modelDigest(m))
		if e != nil {
			return e
		}
		result["state"] = m.State
		_, e = s.accept(ctx, tx, op, digest, rev, result)
		return e
	})
	return m, e
}
func (s *Store) FinishCleanup(ctx context.Context, id string) error {
	if !contracts.ValidID(id) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		var c Cleanup
		if e := s.domainTx(ctx, tx, "Cleanups", id, &c); e != nil {
			return e
		}
		p, e := s.publicationTx(ctx, tx, id)
		if e != nil {
			return e
		}
		if p.State != "retired" && !(p.State == "aborted" && c.LegacyLocationID == nil && (p.Kind == "mapped-audio" || p.Kind == "speaker-clip" || p.Kind == "derived-manifest")) {
			return contracts.Fail("conflict")
		}
		if c.State == "done" {
			return nil
		}
		c.State = "done"
		if e = s.putDomain(ctx, tx, "Cleanups", c); e != nil {
			return e
		}
		digest, _ := intent([]any{"cleanup", id})
		_, e = s.accept(ctx, tx, operationID("cleanup-done", id), digest, rev, map[string]any{"publication_id": id, "state": "done"})
		return e
	})
}
