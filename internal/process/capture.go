// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

// CaptureResult retains partial reports even when the extractor fails. Callers
// decide capture state and persist diagnostics separately from machine output.
type CaptureResult struct {
	Stdout    []byte
	Stderr    []byte
	ExitCode  int
	Truncated bool
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
	command.Env = append(os.Environ(), spec.Env...)
	command.Stdin = spec.Input
	command.WaitDelay = time.Second
	Hide(command, false)
	stdout := &bounded{limit: spec.MaxOutput, cancel: cancel}
	stderr := &bounded{limit: spec.MaxOutput, cancel: cancel}
	command.Stdout = stdout
	command.Stderr = stderr
	tree, err := startTree(command)
	if err != nil {
		return result, contracts.Fail("operation_failed")
	}
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
	close(finished)
	<-supervised
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
