// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func TestSpeakerWorkflowYieldCancelAndRetry(t *testing.T) {
	speakerWorkflowSuite(t, localStore(t, contracts.ID()))
}
func speakerWorkflowSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	parent := speakerWorkPayload(t, s, "speakers.run", json.RawMessage("{}"))
	child, e := s.EnqueueChildWork(ctx, parent, contracts.ID(), "speakers.discover", json.RawMessage("{}"))
	if e != nil {
		t.Fatal(e)
	}
	child, e = s.ClaimWork(ctx, child.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	progress := json.RawMessage(`{"stage":"discovery","awaiting_work_id":"` + child.ID + `"}`)
	if _, e = s.YieldSpeakerWorkflow(ctx, parent, "waiting", progress); e != nil {
		t.Fatal(e)
	}
	if _, e = s.RenewWork(ctx, child, time.Minute); e != nil {
		t.Fatal("pending parent revoked child", e)
	}
	if _, e = s.CancelWork(ctx, contracts.ID(), parent.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CheckpointWork(ctx, child, "complete", "succeeded", json.RawMessage("{}"), time.Minute); e == nil {
		t.Fatal("cancelled parent accepted child")
	}
	got, _ := s.Work(ctx, child.ID)
	if got.State != "cancelled" {
		t.Fatal(got)
	}
	restored, e := s.RetryWork(ctx, contracts.ID(), parent.ID)
	if e != nil || string(restored.Result) != string(progress) {
		t.Fatal(restored, e)
	}
	got, _ = s.Work(ctx, child.ID)
	if got.State != "pending" {
		t.Fatal(got)
	}
	if _, e = s.Export(ctx); e != nil {
		t.Fatal(e)
	}
}
