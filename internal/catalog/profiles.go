// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func profileRecord(role string, p workspace.Profile) (Profile, error) {
	raw, e := json.Marshal(p.Configuration)
	if e != nil {
		return Profile{}, contracts.Fail("invalid_request")
	}
	raw, e = canonical(raw)
	if e != nil {
		return Profile{}, e
	}
	next := Profile{ID: p.ID, Revision: p.Revision, Role: role, Adapter: p.Adapter, Version: p.Version, Configuration: raw}
	if id, ok := p.Configuration["credential_id"].(string); ok {
		next.CredentialID = &id
	}
	if p.ExpectedBackendVersion != "" {
		next.ExpectedBackendVersion = &p.ExpectedBackendVersion
	}
	return next, validateRecord(next)
}
func (s *Store) profileTx(ctx context.Context, tx *sql.Tx, id string, revision int64) (Profile, error) {
	p := Profile{ID: id, Revision: revision}
	var raw string
	var credential, expected sql.NullString
	query := "SELECT role,adapter,version,configuration,credential_id,expected_backend_version FROM profile_revision WHERE workspace_id=? AND id=? AND revision=?"
	var row *sql.Row
	if tx == nil {
		row = s.db.QueryRowContext(ctx, s.query(query), s.workspace, id, revision)
	} else {
		row = s.row(ctx, tx, query, s.workspace, id, revision)
	}
	e := row.Scan(&p.Role, &p.Adapter, &p.Version, &raw, &credential, &expected)
	if e == sql.ErrNoRows {
		return p, contracts.Fail("not_found")
	}
	if e != nil {
		return p, sanitize(e)
	}
	p.Configuration = json.RawMessage(raw)
	if credential.Valid {
		p.CredentialID = &credential.String
	}
	if expected.Valid {
		p.ExpectedBackendVersion = &expected.String
	}
	return p, nil
}
func (s *Store) Profile(ctx context.Context, id string, revision int64) (Profile, error) {
	if !contracts.ValidID(id) || revision < 1 {
		return Profile{}, contracts.Fail("invalid_request")
	}
	return s.profileTx(ctx, nil, id, revision)
}
func (s *Store) registerProfileTx(ctx context.Context, tx *sql.Tx, p Profile) (bool, error) {
	old, e := s.profileTx(ctx, tx, p.ID, p.Revision)
	if e == nil {
		a, _ := intent(old)
		b, _ := intent(p)
		if a != b {
			return false, contracts.Fail("conflict")
		}
		return false, nil
	}
	if typed, ok := e.(*contracts.Error); !ok || typed.Code != "not_found" {
		return false, e
	}
	return true, s.insertRecords(ctx, tx, Records{Profiles: []Profile{p}})
}

// RegisterWorkspace admits immutable nonsecret selected profile revisions.
func (s *Store) RegisterWorkspace(ctx context.Context, w *workspace.Workspace) error {
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		records := Records{}
		for _, selected := range []struct {
			role string
			p    workspace.Profile
		}{{"storage", w.Config.Profiles.Storage}, {"catalog", w.Config.Profiles.Catalog}, {"graph", w.Config.Profiles.Graph}} {
			p, e := profileRecord(selected.role, selected.p)
			if e != nil {
				return e
			}
			added, e := s.registerProfileTx(ctx, tx, p)
			if e != nil {
				return e
			}
			if added {
				records.Profiles = append(records.Profiles, p)
			}
		}
		if len(records.Profiles) == 0 {
			return nil
		}
		digest, e := intent(records)
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, operationID("profiles", digest), digest, rev, map[string]any{"profiles_admitted": len(records.Profiles)})
		return e
	})
}
