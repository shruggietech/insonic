// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
)

func hash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func intent(value any) (string, error) {
	data, e := json.Marshal(value)
	if e != nil {
		return "", contracts.Fail("invalid_request")
	}
	data, e = canonical(data)
	if e != nil {
		return "", e
	}
	return hash(data), nil
}
func (s *Store) replay(ctx context.Context, tx *sql.Tx, id, digest string) (Receipt, bool, error) {
	var r Receipt
	var found string
	var result string
	r.OperationID = id
	e := s.row(ctx, tx, "SELECT digest,revision,result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, id).Scan(&found, &r.Revision, &result)
	if e == sql.ErrNoRows {
		return r, false, nil
	}
	if e != nil {
		return r, false, e
	}
	if found != digest {
		return r, false, contracts.Fail("conflict")
	}
	r.Result = json.RawMessage(result)
	return r, true, nil
}
func (s *Store) accept(ctx context.Context, tx *sql.Tx, id, digest string, revision int64, result any) (Receipt, error) {
	if revision == math.MaxInt64 {
		return Receipt{}, contracts.Fail("conflict")
	}
	data, e := json.Marshal(result)
	if e != nil {
		return Receipt{}, contracts.Fail("invalid_request")
	}
	r := Receipt{id, revision + 1, data}
	if _, e = s.exec(ctx, tx, "INSERT INTO operation_receipt(workspace_id,id,digest,revision,result) VALUES(?,?,?,?,?)", s.workspace, id, digest, r.Revision, string(data)); e != nil {
		return r, e
	}
	_, e = s.exec(ctx, tx, "UPDATE workspace SET revision=? WHERE id=?", r.Revision, s.workspace)
	return r, e
}
func (s *Store) Commit(ctx context.Context, m Mutation) (Receipt, error) {
	if !contracts.ValidID(m.OperationID) || m.Expected < 0 || m.Expected == math.MaxInt64 {
		return Receipt{}, contracts.Fail("invalid_request")
	}
	digest, e := intent(m)
	if e != nil {
		return Receipt{}, e
	}
	var out Receipt
	e = s.write(ctx, func(tx *sql.Tx, revision int64) error {
		r, ok, e := s.replay(ctx, tx, m.OperationID, digest)
		if e != nil {
			return e
		}
		if ok {
			out = r
			return nil
		}
		if revision != m.Expected {
			return contracts.Fail("conflict")
		}
		for _, setting := range m.Settings {
			if setting.Name == "" || len(setting.Name) > 256 || !nonsecret(setting.Value) {
				return contracts.Fail("invalid_request")
			}
			value, e := canonical(setting.Value)
			if e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "INSERT INTO setting_revision(workspace_id,name,revision,value) VALUES(?,?,?,?)", s.workspace, setting.Name, revision+1, string(value)); e != nil {
				return e
			}
		}
		if e = s.insertRecords(ctx, tx, m.Records); e != nil {
			return e
		}
		out, e = s.accept(ctx, tx, m.OperationID, digest, revision, map[string]any{"accepted": true})
		if e != nil {
			return e
		}
		for _, p := range m.Projections {
			if !contracts.ValidID(p.Target) {
				return contracts.Fail("invalid_request")
			}
			doc, e := canonical(p.Document)
			if e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "INSERT INTO graph_target(workspace_id,id,next_sequence,checkpoint,owner_id,generation,lease_until) VALUES(?,?,0,0,NULL,0,0) ON CONFLICT(workspace_id,id) DO NOTHING", s.workspace, p.Target); e != nil {
				return e
			}
			var seq int64
			if e = s.row(ctx, tx, "SELECT next_sequence FROM graph_target WHERE workspace_id=? AND id=?", s.workspace, p.Target).Scan(&seq); e != nil {
				return e
			}
			if seq == math.MaxInt64 {
				return contracts.Fail("conflict")
			}
			if _, e = s.exec(ctx, tx, "INSERT INTO graph_event(workspace_id,target_id,sequence,predecessor,revision,operation_id,document,digest) VALUES(?,?,?,?,?,?,?,?)", s.workspace, p.Target, seq+1, seq, out.Revision, m.OperationID, string(doc), hash(doc)); e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "UPDATE graph_target SET next_sequence=? WHERE workspace_id=? AND id=?", seq+1, s.workspace, p.Target); e != nil {
				return e
			}
		}
		return nil
	})
	return out, e
}
func (s *Store) Settings(ctx context.Context) ([]Setting, error) {
	rows, e := s.db.QueryContext(ctx, s.query("SELECT name,value FROM setting_revision s WHERE workspace_id=? AND revision=(SELECT max(revision) FROM setting_revision p WHERE p.workspace_id=s.workspace_id AND p.name=s.name) ORDER BY name"), s.workspace)
	if e != nil {
		return nil, sanitize(e)
	}
	defer rows.Close()
	out := []Setting{}
	for rows.Next() {
		var r Setting
		var v string
		if e = rows.Scan(&r.Name, &v); e != nil {
			return nil, sanitize(e)
		}
		r.Value = json.RawMessage(v)
		out = append(out, r)
	}
	return out, sanitize(rows.Err())
}
