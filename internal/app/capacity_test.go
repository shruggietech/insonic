package app

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
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
	// Keep real durable attempts and lease renewal active, while holding the
	// synthetic completion timers. Slow Windows storage can take more than
	// the public 60-second job duration to fill capacity.
	holdCompletion := func(value Attempt) {
		a.mu.Lock()
		defer a.mu.Unlock()
		worker := a.attempts[value.JobID]
		if worker == nil {
			t.Fatal("accepted attempt was not attached")
		}
		worker.cancel()
	}
	op := contracts.ID()
	first, err := a.startRequest(op, 60000)
	if err != nil {
		t.Fatal(err)
	}
	holdCompletion(first)
	for range 1023 {
		job, err := a.Start(60000)
		if err != nil {
			t.Fatal(err)
		}
		holdCompletion(job)
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
