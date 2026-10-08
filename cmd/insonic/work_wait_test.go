// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestWorkWaitUsesExistingReadContractAndPreservesTerminalFailures(t *testing.T) {
	id := contracts.ID()
	w := &workspace.Workspace{}
	w.Config.WorkspaceID = contracts.ID()
	for _, terminal := range []string{"succeeded", "failed", "cancelled", "interrupted"} {
		t.Run(terminal, func(t *testing.T) {
			states := []string{"pending", "running", terminal}
			calls := 0
			call := func(_ context.Context, _ *workspace.Workspace, req contracts.Request) (contracts.Response, error) {
				if req.Operation != "work.show" || req.ItemID != id || req.WorkspaceID != w.Config.WorkspaceID {
					t.Fatal("wait changed shared route")
				}
				state := states[calls]
				calls++
				return contracts.Response{Result: catalog.Work{ID: id, State: state}}, nil
			}
			ctx, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			response, e := waitForWork(ctx, w, id, contracts.ID(), time.Millisecond, call)
			if calls != 3 || response.Result.(catalog.Work).State != terminal || (e == nil) != (terminal == "succeeded") {
				t.Fatal("terminal state or failure lost", calls, response, e)
			}
		})
	}
}

func TestWorkWaitTimeoutAndCancellationKeepLastObservedState(t *testing.T) {
	w := &workspace.Workspace{}
	w.Config.WorkspaceID = contracts.ID()
	id := contracts.ID()
	call := func(_ context.Context, _ *workspace.Workspace, _ contracts.Request) (contracts.Response, error) {
		return contracts.Response{Result: catalog.Work{ID: id, State: "running"}}, nil
	}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stop()
	response, e := waitForWork(ctx, w, id, contracts.ID(), time.Millisecond, call)
	var typed *contracts.Error
	if !errors.As(e, &typed) || typed.Code != "unavailable" || response.Result.(catalog.Work).State != "running" {
		t.Fatal("timeout hid last state", response, e)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = waitForWork(cancelled, w, id, contracts.ID(), time.Millisecond, call); !errors.As(e, &typed) || typed.Code != "cancelled" {
		t.Fatal("cancellation ignored", e)
	}
}

func TestWorkWaitReadsPendingAndTerminalThroughActualRuntime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	w, e := workspace.Init(t.TempDir(), "Work wait runtime fixture")
	if e != nil {
		t.Fatal(e)
	}
	store, e := catalog.OpenWorkspace(ctx, w, nil, true)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	id := contracts.ID()
	// Assembly retains pending work until its ephemeral input is resubmitted.
	// The test can therefore drive checkpoints without racing recovery workers.
	if _, e = store.EnqueueWork(ctx, id, "recordings.assemble", json.RawMessage(`{}`)); e != nil {
		t.Fatal(e)
	}
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready}) }()
	select {
	case <-ready:
	case e = <-done:
		t.Fatal(e)
	case <-ctx.Done():
		t.Fatal("runtime startup timeout")
	}
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	states := []string{}
	call := func(ctx context.Context, w *workspace.Workspace, req contracts.Request) (contracts.Response, error) {
		out, e := local.Call(ctx, w, req)
		if e != nil || out.Error != nil {
			return out, e
		}
		raw, e := json.Marshal(out.Result)
		if e != nil {
			t.Fatal(e)
		}
		var observed catalog.Work
		if e = json.Unmarshal(raw, &observed); e != nil || observed.ID != id {
			t.Fatal("runtime returned wrong durable work", string(raw), e)
		}
		states = append(states, observed.State)
		if observed.State == "pending" {
			claim, e := store.ClaimWork(ctx, id, contracts.ID(), time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = store.CheckpointWork(ctx, claim, "complete", "succeeded", json.RawMessage(`{"current":true}`), time.Minute); e != nil {
				t.Fatal(e)
			}
		}
		return out, nil
	}
	out, e := waitForWork(ctx, w, id, contracts.ID(), time.Millisecond, call)
	if e != nil || len(states) != 2 || states[0] != "pending" || states[1] != "succeeded" {
		t.Fatal("wait did not follow actual durable states", states, out, e)
	}
}

func TestWorkWaitArgumentsBoundTimeout(t *testing.T) {
	id := contracts.ID()
	for _, args := range [][]string{{"work", "wait", id}, {"work", "wait", id, "--timeout-ms", "60000"}} {
		if parsed, timeout, e := parseWorkWait(args); e != nil || parsed != id || timeout <= 0 {
			t.Fatal(args, parsed, timeout, e)
		}
	}
	for _, timeout := range []string{"0", "-1", "600001", "NaN"} {
		if _, _, e := parseWorkWait([]string{"work", "wait", id, "--timeout-ms", timeout}); e == nil {
			t.Fatal("unbounded timeout accepted", timeout)
		}
	}
}
