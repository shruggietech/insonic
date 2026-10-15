// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"net"
	"testing"
	"time"
)

func TestOperationTimeoutPreservesCallerDefaultsAndLongFamilies(t *testing.T) {
	for _, ordinary := range []time.Duration{5 * time.Second, 10 * time.Second} {
		for _, operation := range []string{"media.import", "media.refresh", "models.acquire", "models.materialize", "artifacts.publish", "pipelines.inspect", "speakers.select", "speakers.diagnostics", "terms.compile"} {
			if timeout := OperationTimeout(operation, ordinary); timeout != 10*time.Minute {
				t.Fatal(operation, timeout)
			}
		}
		for _, operation := range []string{"workspace.show", "credentials.status", "work.results", "jobs.cancel", "medial.import", "models", ""} {
			if timeout := OperationTimeout(operation, ordinary); timeout != ordinary {
				t.Fatal("ordinary deadline changed", operation, timeout)
			}
		}
	}
}

func TestNativeQueryAcceptsValidResponsePastOrdinaryDeadline(t *testing.T) {
	for _, example := range []struct {
		name         string
		cancelCaller bool
	}{{"slow valid response", false}, {"caller cancellation after connection", true}} {
		t.Run(example.name, func(t *testing.T) {
			w, err := workspace.Init(t.TempDir(), "slow native query transport")
			if err != nil {
				t.Fatal(err)
			}
			listener, cleanup, err := listen(w)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				data, err := frameLimit(conn, MaxRequestFrame)
				var request contracts.Request
				if err != nil || decode(data, &request) != nil {
					done <- err
					return
				}
				if example.cancelCaller {
					cancel()
					done <- ctx.Err()
					return
				}
				select {
				case <-time.After(5100 * time.Millisecond):
				case <-ctx.Done():
					done <- ctx.Err()
					return
				}
				data, _ = json.Marshal(contracts.Response{Kind: "runtime-response", Version: contracts.Version,
					WorkspaceID: w.Config.WorkspaceID, RequestID: request.RequestID, SessionID: contracts.ID(),
					Result: map[string]any{"items": []any{}}})
				_, err = conn.Write(append(data, '\n'))
				done <- err
			}()
			request := contracts.Request{Kind: "runtime-request", Version: contracts.Version,
				WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "query.run"}
			response, err := Call(ctx, w, request)
			if example.cancelCaller {
				cancel()
				listener.Close()
				<-done
				condition, ok := err.(*contracts.Error)
				if !ok || condition.Code != "cancelled" {
					t.Fatalf("caller deadline was not retained: %v", err)
				}
				return
			}
			if err != nil || response.Error != nil {
				cancel()
				listener.Close()
				<-done
				t.Fatalf("valid delayed query response rejected: %v %+v", err, response.Error)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if OperationTimeout("query.explain", time.Second) != OperationTimeout("query.run", time.Second) {
				t.Fatal("explain and run must share the bounded query budget")
			}
		})
	}
}

func TestFrameDeadlineIsNotMalformedInput(t *testing.T) {
	reader, writer := net.Pipe()
	defer reader.Close()
	defer writer.Close()
	reader.SetReadDeadline(time.Now().Add(15 * time.Millisecond))
	_, err := frame(reader)
	condition, ok := err.(*contracts.Error)
	if !ok || condition.Code != "cancelled" {
		t.Fatalf("deadline reported as malformed input: %v", err)
	}
}
