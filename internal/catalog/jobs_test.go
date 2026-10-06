package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func TestSQLiteDurableJobs(t *testing.T) { jobSuite(t, localStore(t, contracts.ID())) }
func jobSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	owner := contracts.ID()
	op := contracts.ID()
	first, e := s.StartJob(ctx, op, owner, 1000, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := s.StartJob(ctx, op, owner, 1000, time.Second)
	if e != nil || replay.AttemptID != first.AttemptID {
		t.Fatalf("start replay %+v %v", replay, e)
	}
	changedOwner, e := s.StartJob(ctx, op, contracts.ID(), 1000, time.Second)
	if e != nil || changedOwner.AttemptID != first.AttemptID {
		t.Fatalf("restart reconciliation %+v %v", changedOwner, e)
	}
	if _, e = s.StartJob(ctx, op, owner, 2000, time.Second); e == nil {
		t.Fatal("changed start replay")
	}
	if _, e = s.Recover(ctx, contracts.ID(), time.Second); e != nil {
		t.Fatal(e)
	}
	time.Sleep(1100 * time.Millisecond)
	if e = s.Complete(ctx, first, "succeeded"); e == nil {
		t.Fatal("expired completion")
	}
	recovered, e := s.Recover(ctx, contracts.ID(), time.Second)
	if e != nil || len(recovered) != 1 {
		t.Fatalf("recover %+v %v", recovered, e)
	}
	second := recovered[0]
	if second.Generation != first.Generation+1 {
		t.Fatal("generation")
	}
	if e = s.Renew(ctx, first, time.Second); e == nil {
		t.Fatal("stale renewal")
	}
	if e = s.Complete(ctx, first, "succeeded"); e == nil {
		t.Fatal("stale publication")
	}
	cancelled, e := s.CancelJob(ctx, contracts.ID(), first.JobID)
	if e != nil || cancelled.State != "cancelled" {
		t.Fatalf("cancel %+v %v", cancelled, e)
	}
	if e = s.Complete(ctx, second, "succeeded"); e == nil {
		t.Fatal("cancelled publication")
	}
	retried, e := s.RetryJob(ctx, contracts.ID(), first.JobID, owner, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if retried.Generation != 3 || retried.AttemptID == second.AttemptID {
		t.Fatal("retry identity")
	}
	history, e := s.History(ctx, first.JobID)
	if e != nil || len(history) != 3 || history[0].State != "interrupted" || history[1].State != "cancelled" {
		t.Fatalf("history %+v %v", history, e)
	}
	if e = s.Complete(ctx, retried, "succeeded"); e != nil {
		t.Fatal(e)
	}
	if e = s.Complete(ctx, retried, "succeeded"); e != nil {
		t.Fatal("completion reconciliation", e)
	}
	if _, e = s.RetryJob(ctx, contracts.ID(), first.JobID, owner, time.Second); e == nil {
		t.Fatal("retried successful job")
	}
}
