// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/app"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const MaxFrame = 1 << 20
const MaxRequestFrame = contracts.MaxWorkPayload + (64 << 10)

type Options struct {
	AssistanceFixture assistance.Fixture // Explicit deterministic qualification only.
	Idle              time.Duration
	Ready             chan<- struct{}
	Input             *credentialcmd.Input
}

func frame(reader io.Reader) ([]byte, error) {
	return frameLimit(reader, MaxFrame)
}
func frameLimit(reader io.Reader, limit int) ([]byte, error) {
	r := bufio.NewReaderSize(reader, limit+1)
	data, err := r.ReadSlice('\n')
	if err != nil || len(data) > limit {
		return nil, contracts.Fail("invalid_request")
	}
	return data, nil
}
func decode(data []byte, target any) error {
	if catalog.ValidateJSON(data) != nil {
		return contracts.Fail("invalid_request")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return contracts.Fail("invalid_request")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}

func Serve(ctx context.Context, w *workspace.Workspace, options Options) error {
	lock, err := w.Lock()
	if err != nil {
		return err
	}
	defer lock.Unlock()
	if err := app.RecoverPlaybackScratch(w); err != nil {
		return err
	}
	listener, cleanup, err := listen(w)
	if err != nil {
		return err
	}
	defer cleanup()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	provider, err := app.Bootstrap(w, options.Input)
	if err != nil {
		return err
	}
	application, err := app.NewContextWithSecrets(ctx, w, provider)
	if err != nil {
		if closer, ok := provider.(interface{ Close() }); ok {
			closer.Close()
		}
		return err
	}
	application.AssistanceFixture = options.AssistanceFixture
	defer application.Close()
	if options.Idle <= 0 {
		options.Idle = 30 * time.Second
	}
	var active atomic.Int64
	var last atomic.Int64
	last.Store(time.Now().UnixNano())
	var handlers sync.WaitGroup
	var lifecycle sync.Mutex
	go func() {
		interval := options.Idle / 2
		if interval > time.Second {
			interval = time.Second
		}
		if interval < 10*time.Millisecond {
			interval = 10 * time.Millisecond
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				listener.Close()
				return
			case <-ticker.C:
				lifecycle.Lock()
				idle := active.Load() == 0 && application.Active() == 0 && time.Since(time.Unix(0, last.Load())) >= options.Idle
				if idle {
					cancel()
				}
				lifecycle.Unlock()
				if idle {
					listener.Close()
					return
				}
			}
		}
	}()
	if options.Ready != nil {
		close(options.Ready)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			return contracts.Fail("unavailable")
		}
		lifecycle.Lock()
		if ctx.Err() != nil || active.Load() >= 64 {
			lifecycle.Unlock()
			conn.Close()
			continue
		}
		active.Add(1)
		handlers.Add(1)
		last.Store(time.Now().UnixNano())
		lifecycle.Unlock()
		go func(conn net.Conn) {
			defer handlers.Done()
			defer conn.Close()
			defer active.Add(-1)
			defer func() { last.Store(time.Now().UnixNano()) }()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			data, readErr := frameLimit(conn, MaxRequestFrame)
			var request contracts.Request
			var response contracts.Response
			if readErr != nil || decode(data, &request) != nil {
				response = contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, SessionID: application.Session, Error: contracts.Fail("invalid_request")}
			} else if request.Version == contracts.Version && schemas.ValidateRequest(data) != nil {
				response = contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, SessionID: application.Session, Error: contracts.Fail("invalid_request")}
				if contracts.ValidID(request.RequestID) {
					response.RequestID = request.RequestID
				}
			} else {
				conn.SetDeadline(time.Now().Add(OperationTimeout(request.Operation, 5*time.Second)))
				operationCtx, finish := context.WithTimeout(ctx, OperationTimeout(request.Operation, 5*time.Second))
				if request.Operation == "query.assist" {
					// One request per connection. Any further input or disconnect cancels this ephemeral operation.
					go func() { var extra [1]byte; conn.Read(extra[:]); finish() }()
				}
				response = application.DispatchContext(operationCtx, request)
				finish()
			}
			encoded, e := json.Marshal(response)
			if e != nil || len(encoded)+1 > MaxFrame {
				response.Result = nil
				response.Error = contracts.Fail("output_limit")
				encoded, _ = json.Marshal(response)
			}
			conn.Write(append(encoded, '\n'))
		}(conn)
	}
	handlers.Wait()
	return nil
}

func Call(ctx context.Context, w *workspace.Workspace, request contracts.Request) (contracts.Response, error) {
	conn, err := dial(ctx, w)
	if err != nil {
		return contracts.Response{Kind: "runtime-response"}, contracts.Fail("unavailable")
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	deadline := time.Now().Add(OperationTimeout(request.Operation, 5*time.Second))
	if value, ok := ctx.Deadline(); ok && value.Before(deadline) {
		deadline = value
	}
	conn.SetDeadline(deadline)
	data, err := json.Marshal(request)
	if err != nil || len(data)+1 > MaxRequestFrame || len(request.Data) > contracts.MaxWorkPayload {
		return contracts.Response{Kind: "runtime-response"}, contracts.Fail("invalid_request")
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return contracts.Response{Kind: "runtime-response"}, contracts.Fail("unavailable")
	}
	data, err = frame(conn)
	if err != nil {
		return contracts.Response{Kind: "runtime-response"}, err
	}
	var response contracts.Response
	if err := decode(data, &response); err != nil {
		return response, err
	}
	if response.Kind != "runtime-response" || response.Version != contracts.Version || response.WorkspaceID != w.Config.WorkspaceID || response.RequestID != request.RequestID || !contracts.ValidID(response.SessionID) {
		return contracts.Response{Kind: "runtime-response"}, contracts.Fail("invalid_request")
	}
	return response, nil
}

func Ensure(ctx context.Context, w *workspace.Workspace, executable string) error {
	return EnsureInput(ctx, w, executable, nil)
}
func EnsureInput(ctx context.Context, w *workspace.Workspace, executable string, input []byte) error {
	probe := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "workspace.show"}
	if _, err := Call(ctx, w, probe); err == nil {
		if len(input) > 0 {
			data, _ := json.Marshal(map[string]any{"arguments": []string{"load"}, "input": json.RawMessage(input)})
			probe.Operation = "credentials.load"
			probe.Data = data
			out, e := Call(ctx, w, probe)
			clear(data)
			if e != nil {
				return e
			}
			if out.Error != nil {
				return out.Error
			}
		}
		return nil
	}
	if executable == "" {
		executable, _ = os.Executable()
	}
	args := []string{"--workspace", w.Root, "runtime", "serve"}
	if len(input) > 0 {
		args = append(args, "--secret-input")
	}
	if err := process.StartDetachedInput(executable, args, input); err != nil {
		return err
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return contracts.Fail("cancelled")
		case <-timer.C:
			return contracts.Fail("unavailable")
		case <-ticker.C:
			if _, err := Call(ctx, w, probe); err == nil {
				return nil
			}
		}
	}
}
