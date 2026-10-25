// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

// CaptureResult retains partial reports even when the extractor fails. Callers
// decide capture state and persist diagnostics separately from machine output.
type CaptureResult struct {
	Started     bool // The owned process was created; exit code alone cannot prove launch.
	Stdout      []byte
	Stderr      []byte
	ExitCode    int
	Truncated   bool
	Descendants *DescendantReport
}

// DescendantReport records only this launch's owned Windows job, never ambient processes.
type DescendantReport struct {
	Initial   uint32              `json:"initial"`
	Remaining uint32              `json:"remaining"`
	Released  bool                `json:"released"`
	Processes []DescendantProcess `json:"processes,omitempty"`
	Timing    *DescendantTiming   `json:"timing,omitempty"`
}

// DescendantTiming uses only elapsed monotonic nanoseconds from this launch.
// Root exit is observed when exec.Cmd.Wait returns, including its output joins;
// it does not claim the exact kernel exit instant. Release timing stops before
// any failure cleanup. ReleaseBudgetNS is zero when no caller deadline exists.
type DescendantTiming struct {
	LaunchElapsedNS      int64 `json:"launch_elapsed_ns"`
	RootWaitElapsedNS    int64 `json:"root_wait_elapsed_ns"`
	RootExitElapsedNS    int64 `json:"root_exit_elapsed_ns"`
	ReleaseDeadline      bool  `json:"release_deadline"`
	ReleaseBudgetNS      int64 `json:"release_budget_ns"`
	ReleaseWaitElapsedNS int64 `json:"release_wait_elapsed_ns"`
}

// DescendantProcess exposes only a fixed image class and kernel exit observation.
type DescendantProcess struct {
	PID      uint32 `json:"pid"`
	Image    string `json:"image"`
	Observed bool   `json:"observed"`
	Running  bool   `json:"running"`
}

func Capture(ctx context.Context, spec Spec) (CaptureResult, error) {
	result := CaptureResult{ExitCode: -1}
	if !filepath.IsAbs(spec.Executable) || spec.MaxOutput < 1 || spec.MaxOutput > 16<<20 {
		return result, contracts.Fail("invalid_request")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return result, contracts.Fail("cancelled")
	}
	command := exec.Command(spec.Executable, spec.Args...)
	command.Dir = spec.Directory
	command.Env = childEnvironment(spec)
	command.Stdin = spec.Input
	command.WaitDelay = time.Second
	Hide(command, false)
	stdout := &bounded{limit: spec.MaxOutput, cancel: cancel}
	stderr := &bounded{limit: spec.MaxOutput, cancel: cancel}
	command.Stdout = stdout
	command.Stderr = stderr
	launchStarted := time.Now()
	tree, err := startTree(command)
	if err != nil {
		return result, contracts.Fail("operation_failed")
	}
	result.Started = true
	launched := time.Now()
	defer tree.Close()
	finished, supervised := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(supervised)
		select {
		case <-ctx.Done():
			tree.Kill()
		case <-finished:
		}
	}()
	err = command.Wait()
	rootExited := time.Now()
	close(finished)
	<-supervised
	if err == nil && ctx.Err() == nil && spec.WaitForDescendants {
		releaseStarted := time.Now()
		timing := &DescendantTiming{
			LaunchElapsedNS:   launched.Sub(launchStarted).Nanoseconds(),
			RootWaitElapsedNS: rootExited.Sub(launched).Nanoseconds(),
			RootExitElapsedNS: rootExited.Sub(launchStarted).Nanoseconds(),
		}
		if deadline, ok := ctx.Deadline(); ok {
			timing.ReleaseDeadline = true
			timing.ReleaseBudgetNS = max(int64(0), deadline.Sub(releaseStarted).Nanoseconds())
		}
		observation, waitErr := waitDescendants(ctx, tree)
		timing.ReleaseWaitElapsedNS = time.Since(releaseStarted).Nanoseconds()
		observation.Timing = timing
		result.Descendants = &observation
		err = waitErr
		if waitErr != nil {
			// Failure cleanup cannot replace the natural-release observation.
			tree.Kill()
			cleanup, stop := context.WithTimeout(context.Background(), time.Second)
			_, _ = waitDescendants(cleanup, tree)
			stop()
		}
	}
	result.Stdout = stdout.bytes
	result.Stderr = stderr.bytes
	result.Truncated = stdout.exceeded || stderr.exceeded
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	if result.Truncated {
		return result, contracts.Fail("output_limit")
	}
	if ctx.Err() != nil {
		return result, contracts.Fail("cancelled")
	}
	if err != nil {
		return result, contracts.Fail("operation_failed")
	}
	return result, nil
}
