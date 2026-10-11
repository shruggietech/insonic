// SPDX-License-Identifier: Apache-2.0
package process

import (
	"bufio"
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// StartDetachedReported owns only startup transport. Wait reaps the child when
// it eventually exits; the detached owner is never tied to caller cancellation.
func StartDetachedReported(ctx context.Context, executable string, args []string, input []byte) ([]byte, error) {
	return startupReported(ctx, executable, args, input, true)
}

func startupReported(ctx context.Context, executable string, args []string, input []byte, detached bool) ([]byte, error) {
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, contracts.Condition("unavailable", "startup_timeout")
		}
		return nil, contracts.Fail("cancelled")
	}
	if !filepath.IsAbs(executable) {
		return nil, contracts.Fail("invalid_request")
	}
	command := exec.Command(executable, args...)
	Hide(command, detached)
	reader, writer, e := os.Pipe()
	if e != nil {
		return nil, contracts.Condition("unavailable", "startup_launch")
	}
	defer reader.Close()
	command.Stdout = writer
	var stdin io.WriteCloser
	if len(input) > 0 {
		stdin, e = command.StdinPipe()
		if e != nil {
			writer.Close()
			return nil, contracts.Condition("unavailable", "startup_launch")
		}
	}
	if e = command.Start(); e != nil {
		writer.Close()
		if stdin != nil {
			stdin.Close()
		}
		return nil, contracts.Condition("unavailable", "startup_launch")
	}
	writer.Close()
	exited := make(chan struct{})
	go func() { _ = command.Wait(); close(exited) }()
	if stdin != nil {
		defer stdin.Close()
		copyInput := append([]byte(nil), input...)
		go func() { defer clear(copyInput); _, _ = stdin.Write(copyInput); _ = stdin.Close() }()
	}
	type report struct {
		raw []byte
		err error
	}
	ready := make(chan report, 1)
	go func() {
		r := bufio.NewReaderSize(reader, 4097)
		raw, err := r.ReadSlice('\n')
		if len(raw) > 4096 || err != nil {
			reason := "startup_report"
			if len(raw) == 0 && err == io.EOF {
				reason = "startup_exit"
			}
			ready <- report{err: contracts.Condition("unavailable", reason)}
			return
		}
		ready <- report{raw: append([]byte(nil), raw...)}
	}()
	select {
	case got := <-ready:
		return got.raw, got.err
	case <-exited:
		// Give the bounded reader time to consume an already-written report.
		// Descendants retaining stdout cannot hide termination of this child.
		grace := time.NewTimer(100 * time.Millisecond)
		defer grace.Stop()
		select {
		case got := <-ready:
			return got.raw, got.err
		case <-grace.C:
			return nil, contracts.Condition("unavailable", "startup_exit")
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil, contracts.Fail("cancelled")
			}
			return nil, contracts.Condition("unavailable", "startup_timeout")
		}
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return nil, contracts.Fail("cancelled")
		}
		return nil, contracts.Condition("unavailable", "startup_timeout")
	}
}
