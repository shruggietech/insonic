// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"strconv"
	"time"
)

type OutboxClaim struct {
	WorkspaceID string          `json:"workspace_id"`
	Target      string          `json:"target"`
	OwnerID     string          `json:"owner_id"`
	Generation  int64           `json:"generation"`
	Sequence    int64           `json:"sequence"`
	Predecessor int64           `json:"predecessor"`
	Revision    int64           `json:"revision"`
	OperationID string          `json:"operation_id"`
	Document    json.RawMessage `json:"document"`
	Digest      string          `json:"digest"`
	LeaseUntil  int64           `json:"lease_until_ns"`
}

func (s *Store) ClaimOutbox(ctx context.Context, target, owner string, ttl time.Duration) (OutboxClaim, error) {
	if !contracts.ValidID(target) || !contracts.ValidID(owner) {
		return OutboxClaim{}, contracts.Fail("invalid_request")
	}
	var out OutboxClaim
	e := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		until, e := expiry(now, ttl)
		if e != nil {
			return e
		}
		var checkpoint, generation, expires, next int64
		var current sql.NullString
		if e = s.row(ctx, tx, "SELECT checkpoint,generation,lease_until,owner_id,next_sequence FROM graph_target WHERE workspace_id=? AND id=?", s.workspace, target).Scan(&checkpoint, &generation, &expires, &current, &next); e != nil {
			return e
		}
		if checkpoint == next {
			return contracts.Fail("not_found")
		}
		if expires > now && current.String != owner {
			return contracts.Fail("conflict")
		}
		if expires <= now || current.String != owner {
			if generation == math.MaxInt64 {
				return contracts.Fail("conflict")
			}
			generation++
		}
		out = OutboxClaim{WorkspaceID: s.workspace, Target: target, OwnerID: owner, Generation: generation, Sequence: checkpoint + 1, LeaseUntil: until}
		var doc string
		if e = s.row(ctx, tx, "SELECT predecessor,revision,operation_id,document,digest FROM graph_event WHERE workspace_id=? AND target_id=? AND sequence=?", s.workspace, target, out.Sequence).Scan(&out.Predecessor, &out.Revision, &out.OperationID, &doc, &out.Digest); e != nil {
			return e
		}
		out.Document = json.RawMessage(doc)
		if hash([]byte(doc)) != out.Digest {
			return contracts.Fail("operation_failed")
		}
		_, e = s.exec(ctx, tx, "UPDATE graph_target SET owner_id=?,generation=?,lease_until=? WHERE workspace_id=? AND id=?", owner, generation, until, s.workspace, target)
		return e
	})
	return out, e
}
func (s *Store) AcknowledgeOutbox(ctx context.Context, c OutboxClaim) error {
	if c.WorkspaceID != s.workspace {
		return contracts.Fail("workspace_mismatch")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		var checkpoint, generation, expires int64
		var owner sql.NullString
		if e = s.row(ctx, tx, "SELECT checkpoint,generation,lease_until,owner_id FROM graph_target WHERE workspace_id=? AND id=?", s.workspace, c.Target).Scan(&checkpoint, &generation, &expires, &owner); e != nil {
			return e
		}
		if owner.String != c.OwnerID || generation != c.Generation || expires <= now {
			return contracts.Fail("conflict")
		}
		var digest, operation, doc string
		var predecessor, revision int64
		if e = s.row(ctx, tx, "SELECT digest,operation_id,document,predecessor,revision FROM graph_event WHERE workspace_id=? AND target_id=? AND sequence=?", s.workspace, c.Target, c.Sequence).Scan(&digest, &operation, &doc, &predecessor, &revision); e != nil {
			return e
		}
		if digest != c.Digest || operation != c.OperationID || doc != string(c.Document) || predecessor != c.Predecessor || revision != c.Revision {
			return contracts.Fail("conflict")
		}
		if checkpoint == c.Sequence {
			return nil
		}
		if checkpoint != c.Predecessor || c.Sequence != checkpoint+1 {
			return contracts.Fail("conflict")
		}
		if _, e = s.exec(ctx, tx, "UPDATE graph_target SET checkpoint=? WHERE workspace_id=? AND id=?", c.Sequence, s.workspace, c.Target); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, operationID("ack", c.Target, c.OperationID, strconv.FormatInt(c.Sequence, 10)), hash([]byte("ack:"+c.Digest)), rev, map[string]any{"target": c.Target, "sequence": c.Sequence})
		return e
	})
}
