// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/mattn/go-isatty"
	"os"
	"strconv"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
)

func parseWorkWait(args []string) (string, time.Duration, error) {
	if (len(args) != 3 && len(args) != 5) || args[0] != "work" || args[1] != "wait" || !contracts.ValidID(args[2]) {
		return "", 0, contracts.Fail("invalid_request")
	}
	timeout := 30 * time.Second
	if len(args) == 5 {
		if args[3] != "--timeout-ms" {
			return "", 0, contracts.Fail("invalid_request")
		}
		value, e := strconv.ParseInt(args[4], 10, 64)
		if e != nil || value < 1 || value > 600000 {
			return "", 0, contracts.Fail("invalid_request")
		}
		timeout = time.Duration(value) * time.Millisecond
	}
	return args[2], timeout, nil
}

type workCaller func(context.Context, *workspace.Workspace, contracts.Request) (contracts.Response, error)

// waitForWork reuses the current process and the existing shared work.show
// contract. It neither starts polling subprocesses nor changes durable work.
func waitForWork(ctx context.Context, w *workspace.Workspace, id, requestID string, interval time.Duration, call workCaller, observers ...func(catalog.Work) error) (contracts.Response, error) {
	var last contracts.Response
	var observedFingerprint string
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return last, contracts.Condition("unavailable", "wait_timeout")
			}
			return last, contracts.Condition("cancelled", "wait_interrupted")
		case <-timer.C:
		}
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: requestID, Operation: "work.show", ItemID: id}
		current, e := call(ctx, w, req)
		if e != nil {
			if ctx.Err() != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return last, contracts.Condition("unavailable", "wait_timeout")
				}
				return last, contracts.Condition("cancelled", "wait_interrupted")
			}
			return last, e
		}
		last = current
		if current.Error != nil {
			return current, current.Error
		}
		raw, e := json.Marshal(current.Result)
		if e != nil {
			return current, contracts.Fail("invalid_request")
		}
		var value catalog.Work
		if json.Unmarshal(raw, &value) != nil || value.ID != id {
			return current, contracts.Fail("invalid_request")
		}
		// State, phase and result represent real persisted observations. Duplicate
		// snapshots produce no extra terminal/log events. No denominator is invented.
		fingerprint := value.State + "\x00" + value.Phase + "\x00" + string(value.Result)
		if fingerprint != observedFingerprint {
			for _, observer := range observers {
				if observer != nil {
					if e := observer(value); e != nil {
						return current, e
					}
				}
			}
			observedFingerprint = fingerprint
		}
		switch value.State {
		case "succeeded":
			return current, nil
		case "failed":
			if value.Error != "" {
				return current, generatedWorkCause(value)
			}
			return current, contracts.Fail("operation_failed")
		case "cancelled":
			return current, contracts.Fail("cancelled")
		case "interrupted":
			return current, contracts.Condition("unavailable", "wait_interrupted")
		case "pending", "running":
		default:
			return current, contracts.Fail("invalid_request")
		}
		timer.Reset(interval)
	}
}

// Only the caller's expired wait budget can relabel a readiness timeout. A
// concrete startup report remains a startup failure even if cancellation races.
func ensureWorkWait(ctx context.Context, w *workspace.Workspace, ensure func(context.Context, *workspace.Workspace) error) error {
	err := ensure(ctx, w)
	if err == nil || ctx.Err() == nil {
		return err
	}
	var typed *contracts.Error
	budgetFailure := errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
	if errors.As(err, &typed) {
		budgetFailure = typed.Reason == "startup_timeout" || typed.Code == "cancelled" && typed.Reason == ""
	}
	if !budgetFailure {
		return err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return contracts.Condition("unavailable", "wait_timeout")
	}
	return contracts.Condition("cancelled", "wait_interrupted")
}

func workWaitCommand(ctx context.Context, w *workspace.Workspace, args []string, requestID string, machine bool) int {
	id, timeout, e := parseWorkWait(args)
	if e != nil {
		return output(nil, e, machine)
	}
	bounded, stop := context.WithTimeout(ctx, timeout)
	defer stop()
	if e = ensureWorkWait(bounded, w, func(ctx context.Context, w *workspace.Workspace) error { return local.Ensure(ctx, w, "") }); e != nil {
		typed := contracts.SafeError(e)
		return output(contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: requestID, Error: typed}, typed, machine)
	}
	observer := workProgressObserver(os.Stderr, isatty.IsTerminal(os.Stderr.Fd()), machine, operatorContext.Quiet)
	logObserver := func(item catalog.Work) error {
		event := diagnosticEvent(item.State)
		return activeLog.Write("debug", event)
	}
	response, e := waitForWork(bounded, w, id, requestID, 200*time.Millisecond, local.Call, observer, logObserver)
	if e == nil {
		return output(response, nil, machine)
	}
	typed, ok := e.(*contracts.Error)
	if !ok {
		typed = contracts.Fail("operation_failed")
	}
	response.Kind = "runtime-response"
	response.Version = contracts.Version
	response.WorkspaceID = w.Config.WorkspaceID
	response.RequestID = requestID
	response.Error = typed
	return output(response, typed, machine)
}

func generatedWorkCause(value catalog.Work) *contracts.Error {
	var result struct {
		Cause *contracts.Error `json:"cause"`
	}
	if json.Unmarshal(value.Result, &result) == nil && result.Cause != nil && result.Cause.Code == value.Error {
		return contracts.SafeError(result.Cause)
	}
	return contracts.Fail(value.Error)
}
