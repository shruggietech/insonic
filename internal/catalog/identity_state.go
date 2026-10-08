// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

func (s *Store) speakerProofs(ctx context.Context, tx *sql.Tx) (map[string]string, error) {
	records, e := s.readRecords(ctx, tx, "Speakers")
	if e != nil {
		return nil, e
	}
	aliases, e := s.readRecords(ctx, tx, "Aliases")
	if e != nil {
		return nil, e
	}
	bySpeaker := map[string][]SpeakerAlias{}
	for _, alias := range aliases.Aliases {
		if validateRecord(alias) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		bySpeaker[alias.SpeakerID] = append(bySpeaker[alias.SpeakerID], alias)
	}
	proofs := map[string]string{}
	for _, sp := range records.Speakers {
		if validateRecord(sp) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		a := bySpeaker[sp.ID]
		if a == nil {
			a = []SpeakerAlias{}
		}
		identity := SpeakerIdentity{sp, a}
		raw, err := json.Marshal(identity)
		if err != nil || len(a) > 1000 || len(raw) > 256<<10 {
			return nil, contracts.Fail("invalid_request")
		}
		proofs[sp.ID], err = intent(identity)
		if err != nil {
			return nil, err
		}
	}
	return proofs, nil
}
func (s *Store) validateIdentityState(ctx context.Context, tx *sql.Tx) error {
	expected := map[string]map[string]string{"speaker_proofs": {}, "pipeline_proofs": {}, "term_proofs": {}}
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var result map[string]json.RawMessage
		if strict([]byte(raw), &result) != nil {
			e = contracts.Fail("invalid_request")
			break
		}
		for key, proofs := range expected {
			if data, ok := result[key]; ok {
				var values map[string]string
				if strict(data, &values) != nil {
					e = contracts.Fail("invalid_request")
					break
				}
				for id, digest := range values {
					if !contracts.ValidID(id) || !digestPattern.MatchString(digest) {
						e = contracts.Fail("invalid_request")
						break
					}
					proofs[id] = digest
				}
			}
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
	speakers, e := s.speakerProofs(ctx, tx)
	if e != nil {
		return e
	}
	actual := map[string]map[string]string{"speaker_proofs": speakers, "pipeline_proofs": {}, "term_proofs": {}}
	aliases, e := s.readRecords(ctx, tx, "Aliases")
	if e != nil {
		return e
	}
	aliasByID := map[string]SpeakerAlias{}
	for _, a := range aliases.Aliases {
		if validateRecord(a) != nil {
			return contracts.Fail("invalid_request")
		}
		aliasByID[a.ID] = a
	}
	for _, field := range []string{"Pipelines", "Terms"} {
		records, err := s.readRecords(ctx, tx, field)
		if err != nil {
			return err
		}
		for _, p := range records.Pipelines {
			if validateRecord(p) != nil {
				return contracts.Fail("invalid_request")
			}
			actual["pipeline_proofs"][p.ID], _ = intent(p)
		}
		for _, t := range records.Terms {
			if validateRecord(t) != nil {
				return contracts.Fail("invalid_request")
			}
			if t.AliasID != nil {
				a, ok := aliasByID[*t.AliasID]
				if !ok {
					return contracts.Fail("invalid_request")
				}
				if t.SpeakerID != nil && *t.SpeakerID != a.SpeakerID {
					return contracts.Fail("invalid_request")
				}
			}
			actual["term_proofs"][t.ID], _ = intent(t)
		}
	}
	for key, values := range actual {
		if len(values) != len(expected[key]) {
			return contracts.Fail("invalid_request")
		}
		for id, digest := range values {
			if expected[key][id] != digest {
				return contracts.Fail("invalid_request")
			}
		}
	}
	return nil
}
func (s *Store) migrateIdentities(ctx context.Context, tx *sql.Tx) error {
	for _, q := range []string{"ALTER TABLE speaker ADD COLUMN revision BIGINT NOT NULL DEFAULT 1", "ALTER TABLE speaker ADD COLUMN state TEXT NOT NULL DEFAULT 'active'"} {
		if _, e := tx.ExecContext(ctx, q); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) attestMigratedIdentities(ctx context.Context, tx *sql.Tx) error {
	q := "SELECT id,revision FROM workspace ORDER BY id"
	if s.backend == "postgresql" {
		q += " FOR UPDATE"
	}
	rows, e := tx.QueryContext(ctx, q)
	if e != nil {
		return e
	}
	type scopeRevision struct {
		id       string
		revision int64
	}
	scopes := []scopeRevision{}
	for rows.Next() {
		var scope scopeRevision
		if e = rows.Scan(&scope.id, &scope.revision); e != nil {
			break
		}
		scopes = append(scopes, scope)
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	for _, scope := range scopes {
		copyStore := *s
		copyStore.workspace = scope.id
		var count int
		if e = copyStore.row(ctx, tx, "SELECT count(*) FROM speaker WHERE workspace_id=?", scope.id).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			continue
		}
		if _, e = copyStore.exec(ctx, tx, "UPDATE speaker SET revision=? WHERE workspace_id=?", scope.revision+1, scope.id); e != nil {
			return e
		}
		proofs, err := copyStore.speakerProofs(ctx, tx)
		if err != nil {
			return err
		}
		digest, _ := intent([]any{"schema5-identities", scope.id, proofs})
		if _, e = copyStore.accept(ctx, tx, operationID("identity-migration", scope.id), digest, scope.revision, map[string]any{"speaker_proofs": proofs}); e != nil {
			return e
		}
	}
	return nil
}
