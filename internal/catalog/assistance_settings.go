// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
)

func (s *Store) NamedSetting(ctx context.Context, name string) (json.RawMessage, int64, error) {
	var raw string
	var revision int64
	e := s.db.QueryRowContext(ctx, s.query("SELECT value,revision FROM setting_revision WHERE workspace_id=? AND name=? ORDER BY revision DESC LIMIT 1"), s.workspace, name).Scan(&raw, &revision)
	if e == sql.ErrNoRows {
		return nil, 0, nil
	}
	return json.RawMessage(raw), revision, sanitize(e)
}
func (s *Store) PutNamedSetting(ctx context.Context, op, name string, expected int64, value json.RawMessage) (Receipt, error) {
	if !contracts.ValidID(op) || name == "" || len(name) > 256 || expected < 0 || expected == math.MaxInt64 || !nonsecret(value) {
		return Receipt{}, contracts.Fail("invalid_request")
	}
	canonicalValue, e := canonical(value)
	if e != nil {
		return Receipt{}, e
	}
	digest, e := intent(map[string]any{"operation_id": op, "name": name, "expected_revision": expected, "value": json.RawMessage(canonicalValue)})
	if e != nil {
		return Receipt{}, e
	}
	var out Receipt
	e = s.write(ctx, func(tx *sql.Tx, revision int64) error {
		replay, found, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if found {
			out = replay
			return nil
		}
		var current int64
		if e = s.row(ctx, tx, "SELECT coalesce(max(revision),0) FROM setting_revision WHERE workspace_id=? AND name=?", s.workspace, name).Scan(&current); e != nil {
			return e
		}
		if current != expected {
			return contracts.Fail("conflict")
		}
		if revision == math.MaxInt64 {
			return contracts.Fail("conflict")
		}
		if _, e = s.exec(ctx, tx, "INSERT INTO setting_revision(workspace_id,name,revision,value) VALUES(?,?,?,?)", s.workspace, name, revision+1, string(canonicalValue)); e != nil {
			return e
		}
		out, e = s.accept(ctx, tx, op, digest, revision, map[string]any{"accepted": true, "setting": name})
		return e
	})
	return out, e
}
