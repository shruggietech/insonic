// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

func (s *Store) YieldSpeakerWorkflow(ctx context.Context, claim Work, phase string, result json.RawMessage) (out Work, e error) {
	if claim.Kind != "speakers.run" || !referenceResult(result) {
		return out, contracts.Fail("invalid_request")
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		var err error
		out, err = s.workAuthority(ctx, tx, claim)
		if err != nil {
			return err
		}
		out.State = "pending"
		out.Owner = ""
		out.LeaseUntil = 0
		out.Phase = phase
		out.Result = result
		return s.recordWork(ctx, tx, &out, rev)
	})
	return
}
func (s *Store) workflowParentCurrent(ctx context.Context, tx *sql.Tx, w Work) error {
	return s.validateWorkflowParent(ctx, tx, w, true)
}
func (s *Store) validateWorkflowParent(ctx context.Context, tx *sql.Tx, w Work, active bool) error {
	var p struct {
		Parent string `json:"parent_work_id"`
	}
	if json.Unmarshal(w.Payload, &p) != nil {
		return contracts.Fail("invalid_request")
	}
	if p.Parent == "" {
		return nil
	}
	if !contracts.ValidID(p.Parent) || p.Parent == w.ID || w.Kind != "speakers.discover" && w.Kind != "models.train" {
		return contracts.Fail("invalid_request")
	}
	var parent Work
	if e := s.domainTx(ctx, tx, "Works", p.Parent, &parent); e != nil {
		return e
	}
	if parent.Kind != "speakers.run" || active && (parent.State != "pending" && parent.State != "running" && parent.State != "interrupted") {
		return contracts.Fail("conflict")
	}
	return nil
}
func (s *Store) transitionWorkflowChildren(ctx context.Context, tx *sql.Tx, parent Work, retry bool, rev *int64) error {
	if parent.Kind != "speakers.run" {
		return nil
	}
	field := "payload"
	if s.backend == "postgresql" {
		field = "payload::jsonb"
	}
	cursor := ""
	for {
		ids, e := s.identityIDs(ctx, tx, "work_operation", s.evidenceJSONText(field, "parent_work_id")+"=?", []any{parent.ID}, cursor, 100)
		if e != nil {
			return e
		}
		for _, id := range ids {
			var child Work
			if e = s.domainTx(ctx, tx, "Works", id, &child); e != nil {
				return e
			}
			if child.State == "succeeded" {
				continue
			}
			if retry {
				if child.State != "failed" && child.State != "cancelled" && child.State != "interrupted" {
					continue
				}
				child.State = "pending"
				child.Error = ""
			} else {
				child.State = "cancelled"
			}
			child.Owner = ""
			child.LeaseUntil = 0
			if e = s.recordWork(ctx, tx, &child, *rev); e != nil {
				return e
			}
			*rev++
		}
		if len(ids) < 100 {
			break
		}
		cursor = ids[len(ids)-1]
	}
	return nil
}
func (s *Store) SetWorkflowSpeakerProfile(ctx context.Context, claim Work, op, speakerID string, expected int64, versionID string) (SpeakerProfile, error) {
	return s.setSpeakerProfile(ctx, op, speakerID, expected, versionID, &claim)
}
