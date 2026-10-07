// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"reflect"
	"strings"
	"time"
)

func domainNamed(field string) domainTable {
	for _, d := range domains {
		if d.field == field {
			return d
		}
	}
	panic("compiled domain missing")
}
func (s *Store) putDomain(ctx context.Context, tx *sql.Tx, field string, value any) error {
	d := domainNamed(field)
	cols := columns(d.typ())
	list := append([]string{"workspace_id"}, names(cols)...)
	args := []any{s.workspace}
	v := reflect.ValueOf(value)
	for _, c := range cols {
		args = append(args, fieldValue(v, c.path))
	}
	updates := []string{}
	for _, c := range cols {
		if c.name != "id" {
			updates = append(updates, c.name+"=excluded."+c.name)
		}
	}
	q := "INSERT INTO " + d.name + "(" + strings.Join(list, ",") + ") VALUES(" + strings.TrimSuffix(strings.Repeat("?,", len(list)), ",") + ") ON CONFLICT(workspace_id,id) DO UPDATE SET " + strings.Join(updates, ",")
	_, e := s.exec(ctx, tx, q, args...)
	return e
}
func (s *Store) domainTx(ctx context.Context, tx *sql.Tx, field, id string, out any) error {
	d := domainNamed(field)
	cols := columns(d.typ())
	vals := make([]any, len(cols))
	ptr := make([]any, len(cols))
	for i := range vals {
		ptr[i] = &vals[i]
	}
	if e := s.row(ctx, tx, "SELECT "+strings.Join(names(cols), ",")+" FROM "+d.name+" WHERE workspace_id=? AND id=?", s.workspace, id).Scan(ptr...); e != nil {
		return e
	}
	doc := map[string]any{}
	for i, c := range cols {
		v := vals[i]
		if b, ok := v.([]byte); ok {
			v = string(b)
		}
		if c.typ == rawType {
			raw := []byte(v.(string))
			if e := ValidateJSON(raw); e != nil {
				return e
			}
			v = json.RawMessage(raw)
		}
		doc[c.name] = v
	}
	b, e := json.Marshal(doc)
	if e != nil {
		return e
	}
	return strict(b, out)
}
func (s *Store) readOne(ctx context.Context, field, id string, out any) error {
	if !contracts.ValidID(id) {
		return contracts.Fail("invalid_request")
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return sanitize(e)
	}
	defer tx.Rollback()
	e = s.domainTx(ctx, tx, field, id, out)
	if e != nil {
		return sanitize(e)
	}
	return sanitize(tx.Commit())
}
func (s *Store) currentRecords(ctx context.Context, field string) (Records, error) {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return Records{}, sanitize(e)
	}
	defer tx.Rollback()
	r, e := s.readRecords(ctx, tx, field)
	if e == nil {
		e = tx.Commit()
	}
	return r, sanitize(e)
}
func (s *Store) Work(ctx context.Context, id string) (Work, error) {
	var out Work
	e := s.readOne(ctx, "Works", id, &out)
	return out, e
}
func (s *Store) Works(ctx context.Context) ([]Work, error) {
	r, e := s.currentRecords(ctx, "Works")
	return r.Works, e
}
func validWork(w Work) bool {
	return contracts.ValidID(w.ID) && contracts.ValidID(w.JournalReceiptID) && w.Kind != "" && len(w.Kind) < 128 && len(w.Payload) <= contracts.MaxWorkPayload && nonsecret(w.Payload) && referenceResult(w.Result) && w.Generation >= 0 && len(w.Phase) < 256 && (w.Owner == "" || contracts.ValidID(w.Owner)) && (w.Error == "" || w.Error == "operation_failed") && strings.Contains("|pending|running|succeeded|failed|cancelled|interrupted|", "|"+w.State+"|") && ((w.State == "running" && w.Owner != "" && w.Generation > 0) || (w.State != "running" && w.LeaseUntil == 0))
}
func referenceResult(raw json.RawMessage) bool {
	// A manifest can admit 10,000 independent items. Preserve their bounded
	// reference results while continuing to reject copied capture payloads.
	if len(raw) > 4<<20 || !nonsecret(raw) {
		return false
	}
	var doc any
	if strict(raw, &doc) != nil {
		return false
	}
	var visit func(any) bool
	visit = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				switch strings.ToLower(k) {
				case "metadata", "raw", "manifest", "payload", "facts", "dates", "observations", "document", "cues", "speaker_attributions", "assignments", "turns":
					return false
				}
				if !visit(v) {
					return false
				}
			}
		case []any:
			for _, v := range x {
				if !visit(v) {
					return false
				}
			}
		}
		return true
	}
	return visit(doc)
}
func (s *Store) EnqueueWork(ctx context.Context, op, kind string, payload json.RawMessage) (Work, error) {
	w := Work{JournalReceiptID: op, ID: op, Kind: kind, Payload: payload, State: "pending", Result: json.RawMessage(`null`)}
	if !validWork(w) {
		return w, contracts.Fail("invalid_request")
	}
	digest, e := intent([]any{"work", kind, payload})
	if e != nil {
		return w, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Works", op, &w)
		}
		if e = s.putDomain(ctx, tx, "Works", w); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"work_id": op, "state": "pending", "work_digest": workJournalDigest(w)})
		return e
	})
	return w, e
}
func (s *Store) workAuthority(ctx context.Context, tx *sql.Tx, claim Work) (Work, error) {
	var w Work
	if e := s.domainTx(ctx, tx, "Works", claim.ID, &w); e != nil {
		return w, e
	}
	now, e := s.now(ctx, tx)
	if e != nil {
		return w, e
	}
	if w.State != "running" || w.Owner != claim.Owner || w.Generation != claim.Generation || w.LeaseUntil <= now {
		return w, contracts.Fail("conflict")
	}
	return w, nil
}
func workJournalDigest(w Work) string {
	w.JournalReceiptID = ""
	w.LeaseUntil = 0
	digest, _ := intent(w)
	return digest
}
func (s *Store) recordWork(ctx context.Context, tx *sql.Tx, w *Work, rev int64) error {
	w.JournalReceiptID = contracts.ID()
	if !validWork(*w) {
		return contracts.Fail("invalid_request")
	}
	if e := s.putDomain(ctx, tx, "Works", *w); e != nil {
		return e
	}
	digest := workJournalDigest(*w)
	_, e := s.accept(ctx, tx, w.JournalReceiptID, digest, rev, map[string]any{"work_id": w.ID, "state": w.State, "generation": w.Generation, "work_digest": digest})
	return e
}
func (s *Store) ClaimWork(ctx context.Context, id, owner string, ttl time.Duration) (Work, error) {
	var w Work
	if !contracts.ValidID(id) || !contracts.ValidID(owner) || !validTTL(ttl) {
		return w, contracts.Fail("invalid_request")
	}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if e := s.domainTx(ctx, tx, "Works", id, &w); e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if w.State != "pending" && w.State != "interrupted" && (w.State != "running" || w.LeaseUntil > now) {
			return contracts.Fail("conflict")
		}
		if w.Generation == math.MaxInt64 {
			return contracts.Fail("conflict")
		}
		w.Owner = owner
		w.Generation++
		w.State = "running"
		w.LeaseUntil = now + int64(ttl)
		return s.recordWork(ctx, tx, &w, rev)
	})
	return w, e
}
func (s *Store) RenewWork(ctx context.Context, claim Work, ttl time.Duration) (Work, error) {
	var w Work
	if !validTTL(ttl) {
		return w, contracts.Fail("invalid_request")
	}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		var e error
		w, e = s.workAuthority(ctx, tx, claim)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		w.LeaseUntil = now + int64(ttl)
		return s.recordWork(ctx, tx, &w, rev)
	})
	return w, e
}
func (s *Store) CheckpointWork(ctx context.Context, claim Work, phase, state string, result json.RawMessage, ttl time.Duration) (Work, error) {
	var w Work
	if (state != "running" && state != "succeeded" && state != "failed") || !validTTL(ttl) || !referenceResult(result) {
		return w, contracts.Fail("invalid_request")
	}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		var e error
		w, e = s.workAuthority(ctx, tx, claim)
		if e != nil {
			return e
		}
		w.Phase = phase
		w.State = state
		w.Result = result
		w.LeaseUntil = 0
		if state == "running" {
			now, e := s.now(ctx, tx)
			if e != nil {
				return e
			}
			w.LeaseUntil = now + int64(ttl)
		}
		if state == "failed" {
			w.Error = "operation_failed"
		}
		return s.recordWork(ctx, tx, &w, rev)
	})
	return w, e
}
func (s *Store) transitionWork(ctx context.Context, op, id string, retry bool) (Work, error) {
	var w Work
	if !contracts.ValidID(op) || !contracts.ValidID(id) {
		return w, contracts.Fail("invalid_request")
	}
	digest, _ := intent([]any{"transition-work", id, retry})
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Works", id, &w)
		}
		if e = s.domainTx(ctx, tx, "Works", id, &w); e != nil {
			return e
		}
		if retry {
			if w.State != "failed" && w.State != "cancelled" && w.State != "interrupted" {
				return contracts.Fail("conflict")
			}
			w.State = "pending"
			w.Result = json.RawMessage(`null`)
			w.Error = ""
		} else {
			if w.State == "succeeded" || w.State == "failed" {
				return contracts.Fail("conflict")
			}
			w.State = "cancelled"
		}
		w.LeaseUntil = 0
		w.Owner = ""
		w.JournalReceiptID = op
		if e = s.putDomain(ctx, tx, "Works", w); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"work_id": id, "state": w.State, "work_digest": workJournalDigest(w)})
		return e
	})
	return w, e
}
func (s *Store) CancelWork(ctx context.Context, op, id string) (Work, error) {
	return s.transitionWork(ctx, op, id, false)
}
func (s *Store) RetryWork(ctx context.Context, op, id string) (Work, error) {
	return s.transitionWork(ctx, op, id, true)
}
