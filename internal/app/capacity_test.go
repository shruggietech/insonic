package app

import (
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTerminalHistoryCannotExhaustActiveCapacity(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "capacity")
	if err != nil {
		t.Fatal(err)
	}
	a := mustNew(t, w)
	defer a.Close()
	oldest, err := a.Start(60000)
	if err != nil {
		t.Fatal(err)
	}
	a.Cancel(oldest.JobID)
	for range 1100 {
		job, err := a.Start(60000)
		if err != nil {
			t.Fatal("terminal history blocked new work", err)
		}
		a.Cancel(job.JobID)
	}
	if _, err := a.Show(oldest.JobID); err != nil {
		t.Fatal("durable history was evicted")
	}
	if len(a.attempts) > 1024 {
		t.Fatal("unbounded history")
	}
}

func TestActiveAttemptsAreNeverEvicted(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "active")
	if err != nil {
		t.Fatal(err)
	}
	a := mustNew(t, w)
	defer a.Close()
	op := contracts.ID()
	first, err := a.startRequest(op, 60000)
	if err != nil {
		t.Fatal(err)
	}
	// Capacity is independent of admission throughput. Seed the remaining real
	// durable claims in one transaction instead of racing 1024 FULL fsyncs
	// against five-second lease expiry. Retain the real first acceptance receipt,
	// normal claim TTL, production attachment and replay/capacity paths.
	func() {
		a.mu.Lock() // Renewal cannot consume a partially constructed fixture.
		defer a.mu.Unlock()
		a.attempts[first.JobID].cancel()
		path := filepath.ToSlash(filepath.Join(w.Control, "catalog.sqlite"))
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		selected := url.URL{Scheme: "file", Path: path, RawQuery: "_foreign_keys=on&_busy_timeout=5000&_txlock=immediate"}
		db, err := sql.Open("sqlite3", selected.String())
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		tx, err := db.BeginTx(a.ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		var revision int64
		var digest string
		if err = tx.QueryRowContext(a.ctx, "SELECT revision FROM workspace WHERE id=?", w.Config.WorkspaceID).Scan(&revision); err != nil {
			t.Fatal(err)
		}
		if err = tx.QueryRowContext(a.ctx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", w.Config.WorkspaceID, op).Scan(&digest); err != nil {
			t.Fatal(err)
		}
		attempts := make([]Attempt, 0, 1023)
		for range 1023 {
			job := first
			job.JobID, job.AttemptID = contracts.ID(), contracts.ID()
			job.StartedNS = time.Now().UnixNano()
			job.LeaseUntil = job.StartedNS + int64(leaseTTL)
			request := contracts.ID()
			if _, err = tx.ExecContext(a.ctx, "INSERT INTO job(workspace_id,id,operation_id,duration_ms,generation,state) VALUES(?,?,?,?,?,'running')", job.WorkspaceID, job.JobID, request, job.DurationMS, job.Generation); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.ExecContext(a.ctx, "INSERT INTO job_attempt(workspace_id,id,job_id,owner_id,generation,state,lease_until,started_ns,ended_ns,reason) VALUES(?,?,?,?,?,'running',?,?,NULL,?)", job.WorkspaceID, job.AttemptID, job.JobID, job.SessionID, job.Generation, job.LeaseUntil, job.StartedNS, job.Reason); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(job)
			if err != nil {
				t.Fatal(err)
			}
			revision++
			if _, err = tx.ExecContext(a.ctx, "INSERT INTO operation_receipt(workspace_id,id,digest,revision,result) VALUES(?,?,?,?,?)", job.WorkspaceID, request, digest, revision, string(raw)); err != nil {
				t.Fatal(err)
			}
			attempts = append(attempts, job)
		}
		if _, err = tx.ExecContext(a.ctx, "UPDATE workspace SET revision=? WHERE id=?", revision, w.Config.WorkspaceID); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.ExecContext(a.ctx, "UPDATE job_attempt SET lease_until=? WHERE workspace_id=? AND owner_id=? AND state='running'", time.Now().Add(leaseTTL).UnixNano(), w.Config.WorkspaceID, a.Session); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
		for _, job := range attempts {
			a.attach(job)
			worker := a.attempts[job.JobID]
			if worker == nil {
				t.Fatal("durable running fixture was not attached")
			}
			worker.cancel() // Hold only synthetic completion, not owner renewal.
		}
		var active int
		if err = db.QueryRowContext(a.ctx, "SELECT count(*) FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id AND j.generation=a.generation JOIN operation_receipt r ON r.workspace_id=j.workspace_id AND r.id=j.operation_id WHERE a.workspace_id=? AND a.owner_id=? AND a.state='running' AND j.state='running' AND a.lease_until>?", w.Config.WorkspaceID, a.Session, time.Now().UnixNano()).Scan(&active); err != nil || active != 1024 {
			t.Fatalf("active durable fixture has %d current claims: %v", active, err)
		}
	}()
	if a.Active() != 1024 {
		t.Fatal("active capacity fixture was not full before rejection")
	}
	if _, err := a.Start(60000); err == nil {
		t.Fatal("active capacity exceeded")
	}
	if replay, err := a.startRequest(op, 60000); err != nil || replay.AttemptID != first.AttemptID {
		t.Fatal("accepted start could not reconcile at capacity", err)
	}
	if value, err := a.Show(first.JobID); err != nil || value.State != "running" {
		t.Fatal("active attempt evicted or completed", err)
	}
	if a.Active() != 1024 {
		t.Fatal("active capacity fixture lost an attempt")
	}
}
