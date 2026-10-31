// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

// Stream sends stdout directly to an owned writer without retaining decoded
// media in the process supervisor. The writer must honor the caller context
// when applying backpressure. Stderr remains bounded by Spec.MaxOutput.
func Stream(ctx context.Context, spec Spec, output io.Writer) (CaptureResult, error) {
	result := CaptureResult{ExitCode: -1}
	if !filepath.IsAbs(spec.Executable) || spec.MaxOutput < 1 || spec.MaxOutput > 16<<20 || output == nil {
		return result, contracts.Fail("invalid_request")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return result, contracts.Fail("cancelled")
	}
	command := exec.Command(spec.Executable, spec.Args...)
	command.Dir, command.Env, command.Stdin = spec.Directory, childEnvironment(spec), spec.Input
	command.WaitDelay = time.Second
	Hide(command, false)
	stdout := &streamWriter{writer: output, cancel: cancel}
	stderr := &bounded{limit: spec.MaxOutput, cancel: cancel}
	command.Stdout, command.Stderr = stdout, stderr
	tree, err := startTree(command)
	if err != nil {
		return result, contracts.Fail("operation_failed")
	}
	result.Started = true
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
	result.Stderr, result.Truncated = stderr.bytes, stderr.exceeded
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	if stdout.failure() != nil {
		return result, contracts.Fail("operation_failed")
	}
	if stderr.exceeded {
		return result, contracts.Fail("output_limit")
	}
	if ctx.Err() != nil {
		return result, contracts.Fail("cancelled")
	}
	if err == nil && spec.WaitForDescendants {
		observation, waitErr := waitDescendants(ctx, tree)
		result.Descendants = &observation
		err = waitErr
		if waitErr != nil {
			tree.Kill()
		}
	}
	if err != nil {
		return result, contracts.Fail("operation_failed")
	}
	return result, nil
}

type streamWriter struct {
	writer io.Writer
	cancel context.CancelFunc
	mu     sync.Mutex
	err    error
}

func (w *streamWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n < len(p) && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.mu.Lock()
		if w.err == nil {
			w.err = err
		}
		w.mu.Unlock()
		w.cancel()
	}
	return n, err
}
func (w *streamWriter) failure() error { w.mu.Lock(); defer w.mu.Unlock(); return w.err }
