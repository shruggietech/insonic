// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"time"
)

func (s *Store) GraphTarget() string { return operationID("graph-evidence", s.workspace) }
func (s *Store) appendGraphEvent(ctx context.Context, tx *sql.Tx, op string, revision int64, raw json.RawMessage) error {
	doc, e := canonical(raw)
	if e != nil {
		return e
	}
	target := s.GraphTarget()
	if _, e = s.exec(ctx, tx, "INSERT INTO graph_target(workspace_id,id,next_sequence,checkpoint,owner_id,generation,lease_until) VALUES(?,?,0,0,NULL,0,0) ON CONFLICT(workspace_id,id) DO NOTHING", s.workspace, target); e != nil {
		return e
	}
	var seq int64
	if e = s.row(ctx, tx, "SELECT next_sequence FROM graph_target WHERE workspace_id=? AND id=?", s.workspace, target).Scan(&seq); e != nil {
		return e
	}
	if seq == math.MaxInt64 {
		return contracts.Fail("conflict")
	}
	if _, e = s.exec(ctx, tx, "INSERT INTO graph_event(workspace_id,target_id,sequence,predecessor,revision,operation_id,document,digest) VALUES(?,?,?,?,?,?,?,?)", s.workspace, target, seq+1, seq, revision, op, string(doc), hash(doc)); e != nil {
		return e
	}
	_, e = s.exec(ctx, tx, "UPDATE graph_target SET next_sequence=? WHERE workspace_id=? AND id=?", seq+1, s.workspace, target)
	return e
}
func (s *Store) explorationMutation(ctx context.Context, tx *sql.Tx, op string, revision int64, data json.RawMessage) (json.RawMessage, error) {
	var result map[string]json.RawMessage
	if strict(data, &result) != nil {
		return data, nil
	}
	dirty := false
	for _, key := range []string{"media_id", "model_id", "recording_id", "mapping_recording_id", "speaker_proofs", "roster_proofs", "term_proofs", "extraction_proofs", "model_alias_proofs", "model_source_proofs", "graph_dirty", "accepted"} {
		if _, ok := result[key]; ok {
			dirty = true
		}
	}
	for _, key := range []string{"recording_id", "mapping_recording_id"} {
		if raw, ok := result[key]; ok {
			var id string
			if strict(raw, &id) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			if _, e := s.exec(ctx, tx, "DELETE FROM current_extraction WHERE workspace_id=? AND id=?", s.workspace, id); e != nil {
				return nil, e
			}
			result["extraction_proofs"], _ = json.Marshal(map[string]string{id: ""})
		}
	}
	data, _ = json.Marshal(result)
	if dirty {
		raw, _ := json.Marshal(map[string]any{"kind": "invalidate", "catalog_revision": revision})
		if e := s.appendGraphEvent(ctx, tx, op, revision, raw); e != nil {
			return nil, e
		}
	}
	return data, nil
}
func (s *Store) EnqueueGraphRefresh(ctx context.Context, op string, expected int64, document json.RawMessage) (Receipt, error) {
	if len(document) > contracts.MaxGraphSnapshot {
		return Receipt{}, contracts.Fail("input_limit")
	}
	if !contracts.ValidID(op) || !referenceMetadata(document) {
		return Receipt{}, contracts.Fail("invalid_request")
	}
	digest, e := intent([]any{"graph-refresh", expected, json.RawMessage(document)})
	if e != nil {
		return Receipt{}, e
	}
	var out Receipt
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			out = r
			return nil
		}
		if rev != expected {
			return contracts.Fail("conflict")
		}
		out, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"graph_refresh_revision": expected})
		if e != nil {
			return e
		}
		return s.appendGraphEvent(ctx, tx, op, out.Revision, document)
	})
	return out, e
}
func (s *Store) GraphStatus(ctx context.Context) (map[string]int64, error) {
	var next, checkpoint, generation int64
	e := s.db.QueryRowContext(ctx, s.query("SELECT next_sequence,checkpoint,generation FROM graph_target WHERE workspace_id=? AND id=?"), s.workspace, s.GraphTarget()).Scan(&next, &checkpoint, &generation)
	if e != nil && e != sql.ErrNoRows {
		return nil, sanitize(e)
	}
	return map[string]int64{"next_sequence": next, "checkpoint": checkpoint, "generation": generation, "pending_events": next - checkpoint}, nil
}
func (s *Store) RenewOutbox(ctx context.Context, c OutboxClaim, ttl time.Duration) (OutboxClaim, error) {
	if c.WorkspaceID != s.workspace {
		return c, contracts.Fail("workspace_mismatch")
	}
	e := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		until, e := expiry(now, ttl)
		if e != nil {
			return e
		}
		r, e := s.exec(ctx, tx, "UPDATE graph_target SET lease_until=? WHERE workspace_id=? AND id=? AND owner_id=? AND generation=? AND lease_until>?", until, s.workspace, c.Target, c.OwnerID, c.Generation, now)
		if e != nil {
			return e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return e
		}
		if n != 1 {
			return contracts.Fail("conflict")
		}
		c.LeaseUntil = until
		return nil
	})
	return c, e
}

func (s *Store) EnqueueGraphRebuild(ctx context.Context, op string) (Receipt, error) {
	if !contracts.ValidID(op) {
		return Receipt{}, contracts.Fail("invalid_request")
	}
	digest, _ := intent([]any{"graph-rebuild", op})
	var out Receipt
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			out = r
			return nil
		}
		out, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"graph_rebuild": true, "graph_refresh_revision": rev})
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(map[string]any{"kind": "invalidate", "catalog_revision": rev})
		return s.appendGraphEvent(ctx, tx, op, out.Revision, raw)
	})
	return out, e
}
