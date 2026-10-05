// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Spec struct {
	Executable string
	Args       []string
	Directory  string
	Env        []string
	Input      io.Reader
	MaxOutput  int
}
type Result struct{ Output []byte }
type bounded struct {
	mu       sync.Mutex
	bytes    []byte
	limit    int
	exceeded bool
	cancel   context.CancelFunc
}

func (b *bounded) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	size := len(p)
	if len(b.bytes)+size > b.limit {
		left := b.limit - len(b.bytes)
		b.bytes = append(b.bytes, p[:left]...)
		b.exceeded = true
		b.cancel()
		return size, nil
	}
	b.bytes = append(b.bytes, p...)
	return size, nil
}
func Run(ctx context.Context, spec Spec) (Result, error) {
	if !filepath.IsAbs(spec.Executable) || spec.MaxOutput < 1 || spec.MaxOutput > 16<<20 {
		return Result{}, contracts.Fail("invalid_request")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	command := exec.CommandContext(ctx, spec.Executable, spec.Args...)
	command.Dir = spec.Directory
	command.Env = append(os.Environ(), spec.Env...)
	command.Stdin = spec.Input
	Hide(command, false)
	command.WaitDelay = time.Second
	output := &bounded{limit: spec.MaxOutput, cancel: cancel}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	if output.exceeded {
		return Result{}, contracts.Fail("output_limit")
	}
	if ctx.Err() != nil {
		return Result{}, contracts.Fail("cancelled")
	}
	if err != nil {
		return Result{}, contracts.Fail("operation_failed")
	}
	return Result{output.bytes}, nil
}
func StartDetached(executable string, args []string) error {
	if !filepath.IsAbs(executable) {
		return contracts.Fail("invalid_request")
	}
	command := exec.Command(executable, args...)
	Hide(command, true)
	// Nil standard handles attach to the null device, never an interactive console.
	if err := command.Start(); err != nil {
		return contracts.Fail("unavailable")
	}
	return command.Process.Release()
}
