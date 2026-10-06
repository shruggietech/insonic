// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

// RegisterWorkspace admits immutable nonsecret selected profile revisions.
// Initialization/restore CLI paths deliberately leave an empty destination empty.
func (s *Store) RegisterWorkspace(ctx context.Context, w *workspace.Workspace) error {
	selected := []struct {
		role    string
		profile workspace.Profile
	}{{"storage", w.Config.Profiles.Storage}, {"catalog", w.Config.Profiles.Catalog}, {"graph", w.Config.Profiles.Graph}}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		records := Records{}
		for _, item := range selected {
			p := item.profile
			if p.Revision < 1 {
				return contracts.Fail("invalid_request")
			}
			raw, e := json.Marshal(p.Configuration)
			if e != nil {
				return contracts.Fail("invalid_request")
			}
			raw, e = canonical(raw)
			if e != nil {
				return e
			}
			var credential *string
			if id, ok := p.Configuration["credential_id"].(string); ok {
				credential = &id
			}
			next := Profile{ID: p.ID, Revision: int64(p.Revision), Role: item.role, Adapter: p.Adapter, Version: p.Version, Configuration: raw, CredentialID: credential}
			if p.ExpectedBackendVersion != "" {
				next.ExpectedBackendVersion = &p.ExpectedBackendVersion
			}
			if e = validateRecord(next); e != nil {
				return e
			}
			var config, role, adapter, version string
			var oldCredential, oldExpected sql.NullString
			e = s.row(ctx, tx, "SELECT configuration,role,adapter,version,credential_id,expected_backend_version FROM profile_revision WHERE workspace_id=? AND id=? AND revision=?", s.workspace, p.ID, p.Revision).Scan(&config, &role, &adapter, &version, &oldCredential, &oldExpected)
			if e == nil {
				oldCanonical, e := canonical([]byte(config))
				if e != nil {
					return e
				}
				sameCredential := oldCredential.Valid == (credential != nil)
				if credential != nil {
					sameCredential = sameCredential && *credential == oldCredential.String
				}
				if string(oldCanonical) != string(raw) || role != item.role || adapter != p.Adapter || version != p.Version || !sameCredential || oldExpected.Valid != (next.ExpectedBackendVersion != nil) || oldExpected.String != p.ExpectedBackendVersion {
					return contracts.Fail("conflict")
				}
				continue
			}
			if e != sql.ErrNoRows {
				return e
			}
			records.Profiles = append(records.Profiles, next)
		}
		if len(records.Profiles) == 0 {
			return nil
		}
		if e := s.insertRecords(ctx, tx, records); e != nil {
			return e
		}
		digest, e := intent(records)
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, operationID("profiles", digest), digest, rev, map[string]any{"profiles_admitted": len(records.Profiles)})
		return e
	})
}
