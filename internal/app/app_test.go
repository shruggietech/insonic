package app

import (
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
	"time"
)

func TestAttemptsFenceRetryAndCancellation(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	a := New(w)
	first, err := a.Start(1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Retry(first.JobID); err == nil {
		t.Fatal("retried active attempt")
	}
	if _, err = a.Cancel(first.JobID); err != nil {
		t.Fatal(err)
	}
	next, err := a.Retry(first.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if next.Generation != first.Generation+1 || next.AttemptID == first.AttemptID {
		t.Fatal("retry identity")
	}
	if a.Complete(first.JobID, first.Generation, "succeeded") {
		t.Fatal("stale result published")
	}
	a.Cancel(next.JobID)
	if a.Active() != 0 {
		t.Fatal("cancelled work remains active")
	}
}

func TestDisconnectedClientDoesNotOwnWork(t *testing.T) {
	w, _ := workspace.Init(t.TempDir(), "test")
	a := New(w)
	job, err := a.Start(20)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	got, err := a.Show(job.JobID)
	if err != nil || got.State != "succeeded" {
		t.Fatalf("runtime work: %+v %v", got, err)
	}
	other := New(w)
	if other.Session == a.Session {
		t.Fatal("restart session reused")
	}
	if _, err := other.Show(job.JobID); err == nil {
		t.Fatal("old session job manufactured")
	}
}
