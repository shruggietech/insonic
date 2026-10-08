// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"errors"
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
func waitForWork(ctx context.Context, w *workspace.Workspace, id, requestID string, interval time.Duration, call workCaller) (contracts.Response, error) {
	var last contracts.Response
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return last, &contracts.Error{Code: "unavailable", Message: "The work did not finish within the selected wait timeout."}
			}
			return last, contracts.Fail("cancelled")
		case <-timer.C:
		}
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: requestID, Operation: "work.show", ItemID: id}
		current, e := call(ctx, w, req)
		if e != nil {
			if ctx.Err() != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return last, &contracts.Error{Code: "unavailable", Message: "The work did not finish within the selected wait timeout."}
				}
				return last, contracts.Fail("cancelled")
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
		switch value.State {
		case "succeeded":
			return current, nil
		case "failed":
			if value.Error != "" {
				return current, contracts.Fail(value.Error)
			}
			return current, contracts.Fail("operation_failed")
		case "cancelled":
			return current, contracts.Fail("cancelled")
		case "interrupted":
			return current, contracts.Fail("unavailable")
		case "pending", "running":
		default:
			return current, contracts.Fail("invalid_request")
		}
		timer.Reset(interval)
	}
}

func workWaitCommand(ctx context.Context, w *workspace.Workspace, args []string, requestID string, machine bool) int {
	id, timeout, e := parseWorkWait(args)
	if e != nil {
		return output(nil, e, machine)
	}
	bounded, stop := context.WithTimeout(ctx, timeout)
	defer stop()
	if e = local.Ensure(bounded, w, ""); e != nil {
		return output(nil, e, machine)
	}
	response, e := waitForWork(bounded, w, id, requestID, 200*time.Millisecond, local.Call)
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
	// Preserve the observed terminal/last state together with its error, so wait
	// failures remain inspectable in the machine-readable result.
	output(response, nil, machine)
	switch typed.Code {
	case "invalid_request":
		return 2
	case "incompatible_version":
		return 3
	case "unavailable":
		return 4
	default:
		return 1
	}
}
