// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"strings"
	"time"
)

type Attempt struct {
	JobID       string `json:"job_id"`
	AttemptID   string `json:"attempt_id"`
	WorkspaceID string `json:"workspace_id"`
	SessionID   string `json:"runtime_session_id"`
	Generation  int64  `json:"claim_generation"`
	State       string `json:"state"`
	DurationMS  int    `json:"duration_ms"`
	LeaseUntil  int64  `json:"lease_until_ns"`
	StartedNS   int64  `json:"started_ns"`
	EndedNS     *int64 `json:"ended_ns"`
	Reason      string `json:"reason"`
}

const attemptColumns = "a.job_id,a.id,a.workspace_id,a.owner_id,a.generation,a.state,j.duration_ms,a.lease_until,a.started_ns,a.ended_ns,a.reason"

func scanAttempt(row interface{ Scan(...any) error }) (Attempt, error) {
	var a Attempt
	e := row.Scan(&a.JobID, &a.AttemptID, &a.WorkspaceID, &a.SessionID, &a.Generation, &a.State, &a.DurationMS, &a.LeaseUntil, &a.StartedNS, &a.EndedNS, &a.Reason)
	return a, e
}
func (s *Store) attempt(ctx context.Context, tx *sql.Tx, id string) (Attempt, error) {
	return scanAttempt(s.row(ctx, tx, "SELECT "+attemptColumns+" FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id AND j.generation=a.generation WHERE a.workspace_id=? AND a.job_id=?", s.workspace, id))
}
func expiry(now int64, ttl time.Duration) (int64, error) {
	if ttl < time.Millisecond || ttl > time.Minute || now > math.MaxInt64-int64(ttl) {
		return 0, contracts.Fail("invalid_request")
	}
	return now + int64(ttl), nil
}
func (s *Store) newAttempt(ctx context.Context, tx *sql.Tx, job, owner, reason string, generation int64, duration int, ttl time.Duration) (Attempt, error) {
	now, e := s.now(ctx, tx)
	if e != nil {
		return Attempt{}, e
	}
	until, e := expiry(now, ttl)
	if e != nil {
		return Attempt{}, e
	}
	a := Attempt{JobID: job, AttemptID: contracts.ID(), WorkspaceID: s.workspace, SessionID: owner, Generation: generation, State: "running", DurationMS: duration, LeaseUntil: until, StartedNS: now, Reason: reason}
	_, e = s.exec(ctx, tx, "INSERT INTO job_attempt(workspace_id,id,job_id,owner_id,generation,state,lease_until,started_ns,ended_ns,reason) VALUES(?,?,?,?,?,'running',?,?,NULL,?)", s.workspace, a.AttemptID, job, owner, generation, until, now, reason)
	return a, e
}
func (s *Store) StartJob(ctx context.Context, op, owner string, duration int, ttl time.Duration) (Attempt, error) {
	if !contracts.ValidID(op) || !contracts.ValidID(owner) || duration < 1 || duration > 60000 {
		return Attempt{}, contracts.Fail("invalid_request")
	}
	if _, e := expiry(0, ttl); e != nil {
		return Attempt{}, e
	}
	// Ownership and renewal policy are execution details, not client intent.
	// A lost response remains reconcilable after the runtime owner changes.
	digest, _ := intent([]any{"start", duration})
	var out Attempt
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return strict(r.Result, &out)
		}
		id := contracts.ID()
		if _, e = s.exec(ctx, tx, "INSERT INTO job(workspace_id,id,operation_id,duration_ms,generation,state) VALUES(?,?,?,?,1,'running')", s.workspace, id, op, duration); e != nil {
			return e
		}
		out, e = s.newAttempt(ctx, tx, id, owner, "start", 1, duration, ttl)
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, out)
		return e
	})
	return out, e
}
func (s *Store) ShowJob(ctx context.Context, id string) (Attempt, error) {
	if !contracts.ValidID(id) {
		return Attempt{}, contracts.Fail("invalid_request")
	}
	a, e := scanAttempt(s.db.QueryRowContext(ctx, s.query("SELECT "+attemptColumns+" FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id AND j.generation=a.generation WHERE a.workspace_id=? AND a.job_id=?"), s.workspace, id))
	return a, sanitize(e)
}

// ReconcileJob reads an accepted start/retry without acquiring execution capacity.
// Changed intent still conflicts; an absent receipt never creates work.
func (s *Store) ReconcileJob(ctx context.Context, op, operation, job string, duration int) (Attempt, error) {
	var out Attempt
	if !contracts.ValidID(op) {
		return out, contracts.Fail("invalid_request")
	}
	var value any
	switch operation {
	case "start":
		if duration < 1 || duration > 60000 || job != "" {
			return out, contracts.Fail("invalid_request")
		}
		value = []any{"start", duration}
	case "retry":
		if !contracts.ValidID(job) || duration != 0 {
			return out, contracts.Fail("invalid_request")
		}
		value = []any{"retry", job}
	default:
		return out, contracts.Fail("invalid_request")
	}
	digest, e := intent(value)
	if e != nil {
		return out, e
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	receipt, found, e := s.replay(ctx, tx, op, digest)
	if e != nil {
		return out, sanitize(e)
	}
	if !found {
		return out, contracts.Fail("conflict")
	}
	if e = strict(receipt.Result, &out); e != nil {
		return out, e
	}
	return out, sanitize(tx.Commit())
}
func (s *Store) History(ctx context.Context, id string) ([]Attempt, error) {
	if !contracts.ValidID(id) {
		return nil, contracts.Fail("invalid_request")
	}
	rows, e := s.db.QueryContext(ctx, s.query("SELECT "+attemptColumns+" FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id WHERE a.workspace_id=? AND a.job_id=? ORDER BY a.generation"), s.workspace, id)
	if e != nil {
		return nil, sanitize(e)
	}
	defer rows.Close()
	out := []Attempt{}
	for rows.Next() {
		a, e := scanAttempt(rows)
		if e != nil {
			return nil, sanitize(e)
		}
		out = append(out, a)
	}
	if e = rows.Err(); e != nil {
		return nil, sanitize(e)
	}
	if len(out) == 0 {
		return nil, contracts.Fail("not_found")
	}
	return out, nil
}

type HistoryPage struct {
	Attempts       []Attempt `json:"attempts"`
	NextGeneration *int64    `json:"next_generation"`
}

func (s *Store) HistoryPage(ctx context.Context, id string, after int64) (HistoryPage, error) {
	out := HistoryPage{Attempts: []Attempt{}}
	if after < 0 {
		return out, contracts.Fail("invalid_request")
	}
	if _, err := s.ShowJob(ctx, id); err != nil {
		return out, err
	}
	rows, err := s.db.QueryContext(ctx, s.query("SELECT "+attemptColumns+" FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id WHERE a.workspace_id=? AND a.job_id=? AND a.generation>? ORDER BY a.generation LIMIT 129"), s.workspace, id, after)
	if err != nil {
		return out, sanitize(err)
	}
	defer rows.Close()
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return out, sanitize(err)
		}
		out.Attempts = append(out.Attempts, a)
	}
	if err = rows.Err(); err != nil {
		return out, sanitize(err)
	}
	if len(out.Attempts) > 128 {
		next := out.Attempts[127].Generation
		out.NextGeneration = &next
		out.Attempts = out.Attempts[:128]
	}
	return out, nil
}
func (s *Store) CancelJob(ctx context.Context, op, id string) (Attempt, error) {
	if !contracts.ValidID(op) || !contracts.ValidID(id) {
		return Attempt{}, contracts.Fail("invalid_request")
	}
	digest, _ := intent([]any{"cancel", id})
	var out Attempt
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return strict(r.Result, &out)
		}
		out, e = s.attempt(ctx, tx, id)
		if e != nil {
			return e
		}
		if out.State == "running" || out.State == "interrupted" {
			now, e := s.now(ctx, tx)
			if e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "UPDATE job_attempt SET state='cancelled',lease_until=0,ended_ns=?,reason='cancel' WHERE workspace_id=? AND id=?", now, s.workspace, out.AttemptID); e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "UPDATE job SET state='cancelled' WHERE workspace_id=? AND id=?", s.workspace, id); e != nil {
				return e
			}
			out.State = "cancelled"
			out.Reason = "cancel"
			out.LeaseUntil = 0
			out.EndedNS = &now
		}
		_, e = s.accept(ctx, tx, op, digest, rev, out)
		return e
	})
	return out, e
}
func (s *Store) RetryJob(ctx context.Context, op, id, owner string, ttl time.Duration) (Attempt, error) {
	if !contracts.ValidID(op) || !contracts.ValidID(id) || !contracts.ValidID(owner) {
		return Attempt{}, contracts.Fail("invalid_request")
	}
	if _, e := expiry(0, ttl); e != nil {
		return Attempt{}, e
	}
	digest, _ := intent([]any{"retry", id})
	var out Attempt
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return strict(r.Result, &out)
		}
		old, e := s.attempt(ctx, tx, id)
		if e != nil {
			return e
		}
		if (old.State != "cancelled" && old.State != "failed") || old.Generation == math.MaxInt64 {
			return contracts.Fail("conflict")
		}
		if _, e = s.exec(ctx, tx, "UPDATE job SET generation=?,state='running' WHERE workspace_id=? AND id=?", old.Generation+1, s.workspace, id); e != nil {
			return e
		}
		out, e = s.newAttempt(ctx, tx, id, owner, "retry", old.Generation+1, old.DurationMS, ttl)
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, out)
		return e
	})
	return out, e
}
func operationID(parts ...string) string {
	h := hash([]byte(joinParts(parts)))
	return h[:8] + "-" + h[8:12] + "-5" + h[13:16] + "-8" + h[17:20] + "-" + h[20:32]
}
func joinParts(parts []string) string { data, _ := json.Marshal(parts); return string(data) }
func (s *Store) current(ctx context.Context, tx *sql.Tx, claim Attempt) (Attempt, error) {
	if claim.WorkspaceID != s.workspace {
		return Attempt{}, contracts.Fail("workspace_mismatch")
	}
	a, e := s.attempt(ctx, tx, claim.JobID)
	if e != nil {
		return a, e
	}
	now, e := s.now(ctx, tx)
	if e != nil {
		return a, e
	}
	if a.AttemptID != claim.AttemptID || a.SessionID != claim.SessionID || a.Generation != claim.Generation || a.State != "running" || a.LeaseUntil <= now {
		return a, contracts.Fail("conflict")
	}
	return a, nil
}
func (s *Store) Renew(ctx context.Context, claim Attempt, ttl time.Duration) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		a, e := s.current(ctx, tx, claim)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		until, e := expiry(now, ttl)
		if e != nil {
			return e
		}
		_, e = s.exec(ctx, tx, "UPDATE job_attempt SET lease_until=? WHERE workspace_id=? AND id=?", until, s.workspace, a.AttemptID)
		return e
	})
}

// RenewClaims renews a bounded runtime's attached attempts in one transaction.
// Expired attempts are excluded rather than revived by a late heartbeat.
func (s *Store) RenewClaims(ctx context.Context, owner string, claims []Attempt, ttl time.Duration) (map[string]bool, error) {
	active := map[string]bool{}
	if len(claims) == 0 {
		return active, nil
	}
	if len(claims) > 1024 || !contracts.ValidID(owner) {
		return nil, contracts.Fail("invalid_request")
	}
	ids := make([]any, len(claims))
	for i, c := range claims {
		if c.WorkspaceID != s.workspace || c.SessionID != owner || !contracts.ValidID(c.AttemptID) {
			return nil, contracts.Fail("invalid_request")
		}
		ids[i] = c.AttemptID
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	err := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		active = map[string]bool{}
		now, err := s.now(ctx, tx)
		if err != nil {
			return err
		}
		until, err := expiry(now, ttl)
		if err != nil {
			return err
		}
		args := append([]any{s.workspace, owner, now}, ids...)
		rows, err := tx.QueryContext(ctx, s.query("SELECT a.id FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id AND j.generation=a.generation WHERE a.workspace_id=? AND a.owner_id=? AND a.state='running' AND j.state='running' AND a.lease_until>? AND a.id IN ("+marks+")"), args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			active[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		updateArgs := append([]any{until, s.workspace, owner, now}, ids...)
		_, err = s.exec(ctx, tx, "UPDATE job_attempt SET lease_until=? WHERE workspace_id=? AND owner_id=? AND state='running' AND lease_until>? AND id IN ("+marks+")", updateArgs...)
		return err
	})
	return active, err
}
func (s *Store) Complete(ctx context.Context, claim Attempt, state string) error {
	if state != "succeeded" && state != "failed" {
		return contracts.Fail("invalid_request")
	}
	op := operationID("complete", claim.AttemptID, state)
	digest, _ := intent([]any{"complete", claim.JobID, claim.AttemptID, claim.SessionID, claim.Generation, state})
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return nil
		}
		a, e := s.current(ctx, tx, claim)
		if e != nil {
			return e
		}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		if _, e = s.exec(ctx, tx, "UPDATE job_attempt SET state=?,lease_until=0,ended_ns=? WHERE workspace_id=? AND id=?", state, now, s.workspace, a.AttemptID); e != nil {
			return e
		}
		if _, e = s.exec(ctx, tx, "UPDATE job SET state=? WHERE workspace_id=? AND id=?", state, s.workspace, a.JobID); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"attempt_id": a.AttemptID, "state": state})
		return e
	})
}
func (s *Store) InterruptOwner(ctx context.Context, owner string) error {
	if !contracts.ValidID(owner) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, s.query("SELECT id,job_id FROM job_attempt WHERE workspace_id=? AND owner_id=? AND state='running' AND lease_until>?"), s.workspace, owner, now)
		if e != nil {
			return e
		}
		type pair struct{ attempt, job string }
		var pairs []pair
		for rows.Next() {
			var p pair
			if e = rows.Scan(&p.attempt, &p.job); e != nil {
				rows.Close()
				return e
			}
			pairs = append(pairs, p)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, p := range pairs {
			if _, e = s.exec(ctx, tx, "UPDATE job_attempt SET state='interrupted',lease_until=0,ended_ns=?,reason='shutdown' WHERE workspace_id=? AND id=?", now, s.workspace, p.attempt); e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "UPDATE job SET state='interrupted' WHERE workspace_id=? AND id=?", s.workspace, p.job); e != nil {
				return e
			}
			if _, e = s.accept(ctx, tx, operationID("interrupt", p.attempt), hash([]byte("interrupt:"+p.attempt)), rev, map[string]any{"attempt_id": p.attempt, "state": "interrupted"}); e != nil {
				return e
			}
			rev++
		}
		// Interrupt current real-work claims in the same authoritative transaction.
		workRows, e := tx.QueryContext(ctx, s.query("SELECT id FROM work_operation WHERE workspace_id=? AND owner=? AND state='running'"), s.workspace, owner)
		if e != nil {
			return e
		}
		ids := []string{}
		for workRows.Next() {
			var id string
			if e = workRows.Scan(&id); e != nil {
				workRows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = workRows.Err()
		workRows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			var w Work
			if e = s.domainTx(ctx, tx, "Works", id, &w); e != nil {
				return e
			}
			w.State = "interrupted"
			w.Owner = ""
			w.LeaseUntil = 0
			w.Phase = "shutdown"
			if e = s.recordWork(ctx, tx, w, rev); e != nil {
				return e
			}
			rev++
		}
		return nil
	})
}
func (s *Store) Recover(ctx context.Context, owner string, ttl time.Duration) ([]Attempt, error) {
	return s.RecoverLimit(ctx, owner, ttl, 128)
}
func (s *Store) RecoverLimit(ctx context.Context, owner string, ttl time.Duration, limit int) ([]Attempt, error) {
	if limit < 1 || limit > 1024 {
		return nil, contracts.Fail("invalid_request")
	}
	if !contracts.ValidID(owner) {
		return nil, contracts.Fail("invalid_request")
	}
	if _, e := expiry(0, ttl); e != nil {
		return nil, e
	}
	out := []Attempt{}
	e := s.write(ctx, func(tx *sql.Tx, rev int64) error {
		out = []Attempt{}
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, s.query("SELECT "+attemptColumns+" FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id AND j.generation=a.generation WHERE a.workspace_id=? AND (a.state='interrupted' OR (a.state='running' AND a.lease_until<=?)) ORDER BY a.job_id LIMIT ?"), s.workspace, now, limit)
		if e != nil {
			return e
		}
		old := []Attempt{}
		for rows.Next() {
			a, e := scanAttempt(rows)
			if e != nil {
				rows.Close()
				return e
			}
			old = append(old, a)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, a := range old {
			if a.Generation == math.MaxInt64 {
				return contracts.Fail("conflict")
			}
			if _, e = s.exec(ctx, tx, "UPDATE job_attempt SET state='interrupted',lease_until=0,ended_ns=?,reason='recovery' WHERE workspace_id=? AND id=? AND state='running'", now, s.workspace, a.AttemptID); e != nil {
				return e
			}
			if _, e = s.exec(ctx, tx, "UPDATE job SET generation=?,state='running' WHERE workspace_id=? AND id=?", a.Generation+1, s.workspace, a.JobID); e != nil {
				return e
			}
			next, e := s.newAttempt(ctx, tx, a.JobID, owner, "recovery", a.Generation+1, a.DurationMS, ttl)
			if e != nil {
				return e
			}
			if _, e = s.accept(ctx, tx, operationID("recover", a.AttemptID), hash([]byte("recover:"+a.AttemptID)), rev, next); e != nil {
				return e
			}
			rev++
			out = append(out, next)
		}
		return nil
	})
	return out, e
}

type Lease struct {
	Role       string
	OwnerID    string
	Generation int64
	Until      int64
}

func (s *Store) ClaimLease(ctx context.Context, role, owner string, ttl time.Duration) (Lease, error) {
	if role == "" || !contracts.ValidID(owner) {
		return Lease{}, contracts.Fail("invalid_request")
	}
	var out Lease
	e := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		until, e := expiry(now, ttl)
		if e != nil {
			return e
		}
		var current Lease
		current.Role = role
		e = s.row(ctx, tx, "SELECT owner_id,generation,lease_until FROM workspace_lease WHERE workspace_id=? AND role=?", s.workspace, role).Scan(&current.OwnerID, &current.Generation, &current.Until)
		if e == nil {
			if current.Until > now {
				return contracts.Fail("conflict")
			}
			if current.Generation == math.MaxInt64 {
				return contracts.Fail("conflict")
			}
			out = Lease{role, owner, current.Generation + 1, until}
			_, e = s.exec(ctx, tx, "UPDATE workspace_lease SET owner_id=?,generation=?,lease_until=? WHERE workspace_id=? AND role=?", owner, out.Generation, until, s.workspace, role)
			return e
		}
		if e != sql.ErrNoRows {
			return e
		}
		out = Lease{role, owner, 1, until}
		_, e = s.exec(ctx, tx, "INSERT INTO workspace_lease(workspace_id,role,owner_id,generation,lease_until) VALUES(?,?,?,?,?)", s.workspace, role, owner, 1, until)
		return e
	})
	return out, e
}
func (s *Store) RenewLease(ctx context.Context, claim Lease, ttl time.Duration) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		until, e := expiry(now, ttl)
		if e != nil {
			return e
		}
		r, e := s.exec(ctx, tx, "UPDATE workspace_lease SET lease_until=? WHERE workspace_id=? AND role=? AND owner_id=? AND generation=? AND lease_until>?", until, s.workspace, claim.Role, claim.OwnerID, claim.Generation, now)
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
		return nil
	})
}
