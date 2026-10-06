package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
	"time"
)

type blockedCatalog struct {
	catalog.Catalog
	entered chan struct{}
}

func (b blockedCatalog) StartJob(ctx context.Context, _ string, _ string, _ int, _ time.Duration) (catalog.Attempt, error) {
	close(b.entered)
	<-ctx.Done()
	return catalog.Attempt{}, contracts.Fail("cancelled")
}
func TestCloseCancelsInFlightCatalogCall(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "close blocked call")
	if e != nil {
		t.Fatal(e)
	}
	a := mustNew(t, w)
	entered := make(chan struct{})
	a.Catalog = blockedCatalog{a.Catalog, entered}
	returned := make(chan struct{})
	go func() { a.Start(60000); close(returned) }()
	<-entered
	closed := make(chan struct{})
	go func() { a.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		a.cancel()
		<-returned
		<-closed
		t.Fatal("Close waited for its mutex before cancelling SQL")
	}
	<-returned
}

func TestCancelReplayDoesNotDetachRetriedWork(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "cancel replay")
	if e != nil {
		t.Fatal(e)
	}
	a := mustNew(t, w)
	first, e := a.Start(60000)
	if e != nil {
		t.Fatal(e)
	}
	op := contracts.ID()
	if _, e = a.cancelRequest(op, first.JobID); e != nil {
		t.Fatal(e)
	}
	next, e := a.Retry(first.JobID)
	if e != nil {
		t.Fatal(e)
	}
	if old, e := a.cancelRequest(op, first.JobID); e != nil || old.AttemptID != first.AttemptID {
		t.Fatal("cancel reconciliation", e)
	}
	if a.Active() != 1 || !a.Complete(next.JobID, next.Generation, "succeeded") {
		t.Fatal("old cancellation detached the new attempt")
	}
}

func TestAttemptsFenceRetryAndCancellation(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	a := mustNew(t, w)
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
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	a := mustNew(t, w)
	job, err := a.Start(20)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	got, err := a.Show(job.JobID)
	if err != nil || got.State != "succeeded" {
		t.Fatalf("runtime work: %+v %v", got, err)
	}
	a.Close()
	other := mustNew(t, w)
	if other.Session == a.Session {
		t.Fatal("restart session reused")
	}
	if got, err := other.Show(job.JobID); err != nil || got.State != "succeeded" {
		t.Fatal("durable job lost after restart")
	}
}

func mustNew(t *testing.T, w *workspace.Workspace) *App {
	t.Helper()
	a, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	return a
}
func TestRestartRecoversInterruptedWork(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "recovery")
	if err != nil {
		t.Fatal(err)
	}
	a := mustNew(t, w)
	first, err := a.Start(1000)
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	next := mustNew(t, w)
	got, err := next.Show(first.JobID)
	if err != nil || got.AttemptID == first.AttemptID || got.Generation != first.Generation+1 || got.Reason != "recovery" {
		t.Fatalf("recovery %+v %v", got, err)
	}
	history, err := next.Catalog.History(next.ctx, got.JobID)
	if err != nil || len(history) != 2 || history[0].State != "interrupted" {
		t.Fatalf("history %+v %v", history, err)
	}
	next.Cancel(got.JobID)
	next.Close()
	last := mustNew(t, w)
	got, err = last.Show(first.JobID)
	if err != nil || got.State != "cancelled" {
		t.Fatalf("cancelled recovery %+v %v", got, err)
	}
}
