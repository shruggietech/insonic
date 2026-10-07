// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"github.com/shruggietech/insonic/internal/contracts"
)

func (s *Store) LegacyCleanupPublications(ctx context.Context, profile string, revision int64) ([]Publication, error) {
	if !contracts.ValidID(profile) || revision < 1 {
		return nil, contracts.Fail("invalid_request")
	}
	rows, e := s.db.QueryContext(ctx, s.query("SELECT l.id,l.artifact_id,l.profile_id,l.profile_revision,l.key,coalesce(l.version,''),a.digest,a.size,a.kind FROM library_cleanup c JOIN artifact_location l ON l.workspace_id=c.workspace_id AND l.id=c.legacy_location_id JOIN artifact a ON a.workspace_id=l.workspace_id AND a.id=l.artifact_id WHERE c.workspace_id=? AND c.state='pending' AND l.profile_id=? AND l.profile_revision=? ORDER BY l.id"), s.workspace, profile, revision)
	if e != nil {
		return nil, sanitize(e)
	}
	defer rows.Close()
	out := []Publication{}
	for rows.Next() {
		var p Publication
		if e = rows.Scan(&p.ID, &p.ArtifactID, &p.ProfileID, &p.ProfileRevision, &p.Key, &p.Version, &p.Digest, &p.Size, &p.Kind); e != nil {
			return nil, sanitize(e)
		}
		p.LocationID = p.ID
		out = append(out, p)
	}
	return out, sanitize(rows.Err())
}

func (s *Store) validateLegacyCleanup(ctx context.Context, tx *sql.Tx, c Cleanup) error {
	if c.LegacyLocationID == nil || *c.LegacyLocationID != c.ID || c.State != "pending" {
		return contracts.Fail("invalid_request")
	}
	var count int
	e := s.row(ctx, tx, "SELECT count(*) FROM artifact_location l JOIN artifact a ON a.workspace_id=l.workspace_id AND a.id=l.artifact_id WHERE l.workspace_id=? AND l.id=? AND l.state='available' AND NOT EXISTS(SELECT 1 FROM media_asset m WHERE m.workspace_id=l.workspace_id AND m.artifact_id=l.artifact_id AND m.role IN ('original','subtitle-source')) AND NOT EXISTS(SELECT 1 FROM model_artifact m WHERE m.workspace_id=l.workspace_id AND m.artifact_id=l.artifact_id AND m.role IN ('weights','checkpoint','model'))", s.workspace, c.ID).Scan(&count)
	if e != nil {
		return e
	}
	if count != 1 {
		return contracts.Fail("invalid_request")
	}
	return nil
}

// A legacy location may be reused only for an accepted cleanup obligation and
// an exact immutable match. Callers still supply an independently checked
// sha256-readback and a current fenced publication claim to AdmitPublication.
func (s *Store) reconcileLegacyAdmission(ctx context.Context, tx *sql.Tx, p Publication) (bool, error) {
	var c Cleanup
	e := s.domainTx(ctx, tx, "Cleanups", p.ID, &c)
	if e == sql.ErrNoRows {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if c.LegacyLocationID == nil {
		return false, nil
	}
	if e = s.validateLegacyCleanup(ctx, tx, c); e != nil {
		return false, e
	}
	var count int
	e = s.row(ctx, tx, "SELECT count(*) FROM artifact a JOIN artifact_location l ON l.workspace_id=a.workspace_id AND l.artifact_id=a.id WHERE a.workspace_id=? AND a.id=? AND a.digest=? AND a.size=? AND a.kind=? AND l.id=? AND l.key=? AND l.profile_id=? AND l.profile_revision=? AND coalesce(l.version,'')=? AND l.state='available'", s.workspace, p.ArtifactID, p.Digest, p.Size, p.Kind, p.LocationID, p.Key, p.ProfileID, p.ProfileRevision, p.Version).Scan(&count)
	if e != nil {
		return false, e
	}
	if count != 1 || p.ID != p.LocationID {
		return false, contracts.Fail("conflict")
	}
	return true, nil
}
