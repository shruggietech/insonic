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
	ctx, cancel := context.WithCancel(context.Background())
	store, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		cancel()
		t.Fatal(e)
	}
	entered := make(chan struct{})
	// Inject the blocking adapter before exposing the App to concurrent calls.
	a := &App{Workspace: w, Session: contracts.ID(), Catalog: blockedCatalog{store, entered}, ctx: ctx, cancel: cancel, attempts: map[string]*worker{}, workers: map[string]*realWorker{}}
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
	// Completion includes a durable catalog write, whose latency is independent
	// of the worker's 20 ms timer. Wait for authority rather than scheduler speed.
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	for {
		got, err := a.Show(job.JobID)
		if err != nil || got.State != "running" && got.State != "succeeded" {
			t.Fatalf("runtime work: %+v %v", got, err)
		}
		if got.State == "succeeded" {
			break
		}
		select {
		case <-deadline.C:
			t.Fatalf("runtime work did not persist completion: %+v", got)
		case <-poll.C:
		}
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
