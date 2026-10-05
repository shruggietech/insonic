package app

import (
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
)

func TestTerminalHistoryCannotExhaustActiveCapacity(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "capacity")
	if err != nil {
		t.Fatal(err)
	}
	a := New(w)
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
	if _, err := a.Show(oldest.JobID); err == nil {
		t.Fatal("oldest terminal record retained")
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
	a := New(w)
	defer a.Close()
	first, err := a.Start(60000)
	if err != nil {
		t.Fatal(err)
	}
	for range 1023 {
		if _, err := a.Start(60000); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.Start(60000); err == nil {
		t.Fatal("active capacity exceeded")
	}
	if _, err := a.Show(first.JobID); err != nil {
		t.Fatal("active attempt evicted")
	}
}
