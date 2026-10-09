// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
	"time"
)

func TestAssistanceWireCancellation(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "assistance")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	done := make(chan error, 1)
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	fixture := func(ctx context.Context, _ assistance.Request, _ assistance.Config) (assistance.Proposal, error) {
		started <- struct{}{}
		<-ctx.Done()
		stopped <- struct{}{}
		return assistance.Proposal{}, contracts.Fail("cancelled")
	}
	go func() { done <- Serve(ctx, w, Options{Ready: ready, AssistanceFixture: fixture}) }()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("startup")
	}
	req := func(op string, data any) contracts.Request {
		raw, _ := json.Marshal(data)
		return contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: op, Data: raw}
	}
	c := assistance.DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://127.0.0.1:1/query"
	c.Model = "fixture"
	r, e := Call(ctx, w, req("query.assistance-set", map[string]any{"expected_revision": 0, "configuration": c}))
	if e != nil || r.Error != nil {
		t.Fatalf("configuration: %v %+v", e, r)
	}
	callctx, stop := context.WithCancel(ctx)
	called := make(chan struct{})
	go func() { Call(callctx, w, req("query.assist", map[string]any{"prompt": "List media"})); close(called) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("wire did not invoke assistance")
	}
	stop()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("client cancellation did not close connection")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("disconnect did not cancel provider")
	}
	if OperationTimeout("query.assist", time.Second) != 90*time.Second {
		t.Fatal("assistance deadline")
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown")
	}
}
