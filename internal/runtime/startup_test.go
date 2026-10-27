// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	qt "github.com/shruggietech/insonic/internal/qualificationtiming"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartupJoinsHeldOwnerAfterReadinessProbeExpires(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Busy owner")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := w.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	listener, cleanup, err := listen(w)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	ownerDone := make(chan error, 1)
	session := contracts.ID()
	go func() {
		for i := 0; i < 2; i++ {
			conn, e := listener.Accept()
			if e != nil {
				ownerDone <- e
				return
			}
			conn.SetDeadline(time.Now().Add(qt.Duration("readiness_seconds")))
			raw, e := frame(conn)
			var request contracts.Request
			if e == nil {
				e = decode(raw, &request)
			}
			if e != nil {
				conn.Close()
				ownerDone <- e
				return
			}
			if i == 0 {
				// Observe the actual first probe's deadline closing its connection.
				// The owner remains alive and holds authority throughout that miss.
				_, e = io.Copy(io.Discard, conn)
				conn.Close()
				if e != nil {
					ownerDone <- e
					return
				}
				continue
			}
			response := contracts.Response{Kind: "runtime-response", Version: contracts.Version,
				WorkspaceID: w.Config.WorkspaceID, SessionID: session, RequestID: request.RequestID}
			raw, e = json.Marshal(response)
			if e == nil {
				_, e = conn.Write(append(raw, '\n'))
			}
			conn.Close()
			ownerDone <- e
		}
	}()
	var launches atomic.Int32
	err = ensureInput(context.Background(), w, "fixture", nil,
		func(context.Context, string, []string, []byte, func([]byte) error) ([]byte, error) {
			launches.Add(1)
			return nil, contracts.Condition("unavailable", "startup_launch")
		})
	cleanup() // Also releases the listener on the intentionally failing old path.
	select {
	case ownerErr := <-ownerDone:
		if err == nil && ownerErr != nil {
			t.Fatal(ownerErr)
		}
	case <-time.After(qt.Duration("cleanup_seconds")):
		t.Fatal("owned probe listener did not release")
	}
	if err != nil || launches.Load() != 0 {
		t.Fatalf("live owner was unnecessarily relaunched: launches=%d error=%v", launches.Load(), err)
	}
}

func TestStartupHeldOwnerHonorsCallerDeadlineAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			w, err := workspace.Init(t.TempDir(), "Unavailable held owner")
			if err != nil {
				t.Fatal(err)
			}
			lock, err := w.Lock()
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Unlock()
			ctx, finish := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer finish()
			if cancelled {
				finish()
			}
			launches := 0
			err = ensureInput(ctx, w, "fixture", nil,
				func(context.Context, string, []string, []byte, func([]byte) error) ([]byte, error) {
					launches++
					return nil, contracts.Condition("unavailable", "startup_launch")
				})
			condition, ok := err.(*contracts.Error)
			if !ok || launches != 0 || (!cancelled && condition.Reason != "startup_timeout") || (cancelled && condition.Code != "cancelled") {
				t.Fatalf("held authority lost caller bound: launches=%d error=%v", launches, err)
			}
		})
	}
}

func TestStartupLaunchesOnlyAfterUnresponsiveOwnerReleasesAuthority(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Releasing owner")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := w.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	listener, cleanup, err := listen(w)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	done := make(chan error, 1)
	go func() {
		conn, e := listener.Accept()
		if e == nil {
			conn.SetDeadline(time.Now().Add(qt.Duration("readiness_seconds")))
			_, e = frame(conn)
			if e == nil {
				_, e = io.Copy(io.Discard, conn) // First readiness deadline expires.
			}
			conn.Close()
		}
		cleanup()
		lock.Unlock()
		done <- e
	}()
	launches := 0
	err = ensureInput(context.Background(), w, "fixture", nil,
		func(context.Context, string, []string, []byte, func([]byte) error) ([]byte, error) {
			launches++
			// A genuine replacement bootstrap failure must retain its cause.
			return nil, contracts.Condition("unavailable", "runtime_catalog")
		})
	cleanup()
	select {
	case ownerErr := <-done:
		if ownerErr != nil {
			t.Fatal(ownerErr)
		}
	case <-time.After(qt.Duration("cleanup_seconds")):
		t.Fatal("releasing owner fixture did not stop")
	}
	condition, ok := err.(*contracts.Error)
	if !ok || condition.Reason != "runtime_catalog" || launches != 1 {
		t.Fatalf("released owner did not preserve bootstrap outcome: launches=%d error=%v", launches, err)
	}
}

func startupID(args []string) string {
	for i, arg := range args {
		if arg == "--startup-id" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
func TestStartupReportValidationAndImmediateFailure(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Startup reports")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"failed", "wrong-workspace", "wrong-id", "wrong-version", "wrong-stage", "unknown", "trailing", "duplicate"} {
		start := func(_ context.Context, _ string, args []string, input []byte, validate func([]byte) error) ([]byte, error) {
			report := contracts.StartupReport{Kind: "runtime-startup", Version: contracts.Version, StartupID: startupID(args), WorkspaceID: w.Config.WorkspaceID, State: "failed", Stage: "startup", Code: "unavailable", Reason: "runtime_catalog"}
			switch mode {
			case "wrong-workspace":
				report.WorkspaceID = contracts.ID()
			case "wrong-id":
				report.StartupID = contracts.ID()
			case "wrong-version":
				report.Version = "99"
			case "wrong-stage":
				report.Stage = "ready"
			}
			raw, _ := json.Marshal(report)
			switch mode {
			case "unknown":
				raw = append(raw[:len(raw)-1], []byte(",\"private_input\":\"fixture-secret\"}")...)
			case "trailing":
				raw = append(raw, []byte(" {}")...)
			case "duplicate":
				raw = append(raw[:len(raw)-1], []byte(",\"state\":\"ready\"}")...)
			}
			return raw, validate(raw)
		}
		begin := time.Now()
		err = ensureInput(context.Background(), w, "fixture", nil, start)
		typed, ok := err.(*contracts.Error)
		want := "startup_report"
		if mode == "failed" {
			want = "runtime_catalog"
		}
		if !ok || typed.Reason != want || time.Since(begin) > qt.Duration("readiness_seconds") || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal(mode, err)
		}
	}
}

// Prepare fixture schema outside the fixed product startup clock. The runtime
// owner is still absent, and startup must load the elected protected input.
func prepareOwnerStartupCatalog(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	ctx, cancel := qt.Context(context.Background(), "subprocess_seconds")
	defer cancel()
	store, err := catalog.OpenWorkspace(ctx, w, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err := w.Lock()
	if err != nil {
		t.Fatal("schema preparation started a runtime owner", err)
	}
	lock.Unlock()
}

func TestConcurrentColdStartupLoadsProtectedInputIntoOneOwner(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Concurrent startup")
	if err != nil {
		t.Fatal(err)
	}
	prepareOwnerStartupCatalog(t, w)
	if err = secrets.Select(w, "session"); err != nil {
		t.Fatal(err)
	}
	ownerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var owners sync.WaitGroup
	launch := func(ctx context.Context, _ string, args []string, raw []byte, validate func([]byte) error) ([]byte, error) {
		for _, arg := range args {
			if strings.Contains(arg, "session-secret") {
				t.Fatal("secret in child arguments")
			}
		}
		input, e := credentialcmd.ReadInput(bytes.NewReader(raw))
		if e != nil {
			return nil, e
		}
		reports := make(chan contracts.StartupReport, 1)
		report := contracts.StartupReport{Kind: "runtime-startup", Version: contracts.Version, StartupID: startupID(args), WorkspaceID: w.Config.WorkspaceID, State: "ready", Stage: "ready"}
		owners.Add(1)
		go func() {
			defer owners.Done()
			defer input.Close()
			err := Serve(ownerCtx, w, Options{Input: input, Idle: qt.Duration("fixture_idle_seconds"), ReportReady: func() { reports <- report }})
			if err != nil {
				safe := contracts.SafeError(err)
				report.State, report.Stage, report.Code, report.Reason = "failed", "startup", safe.Code, safe.Reason
				reports <- report
			}
		}()
		select {
		case report = <-reports:
			raw, _ := json.Marshal(report)
			return raw, validate(raw)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	id := contracts.ID()
	raw, _ := json.Marshal(map[string]any{"session": map[string]string{id: "session-secret"}})
	var calls sync.WaitGroup
	failures := make(chan error, 4)
	for i := 0; i < 4; i++ {
		calls.Add(1)
		go func() { defer calls.Done(); failures <- ensureInput(context.Background(), w, "fixture", raw, launch) }()
	}
	calls.Wait()
	for i := 0; i < 4; i++ {
		if e := <-failures; e != nil {
			cancel()
			owners.Wait()
			t.Fatal(e)
		}
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "credentials.status"}
	req.Data, _ = json.Marshal(map[string]any{"arguments": []string{"status", id}, "input": map[string]any{}})
	out, err := Call(context.Background(), w, req)
	encoded, _ := json.Marshal(out)
	if err != nil || out.Error != nil || !bytes.Contains(encoded, []byte("configured")) || bytes.Contains(encoded, []byte("session-secret")) {
		t.Fatal(out, err)
	}
	first, err := Call(context.Background(), w, contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "workspace.show"})
	if err != nil || first.Error != nil {
		t.Fatal(first, err)
	}
	if err = ensureInput(context.Background(), w, "fixture", nil, func(context.Context, string, []string, []byte, func([]byte) error) ([]byte, error) {
		t.Fatal("live owner started again")
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	cancel()
	owners.Wait()
}
func TestStartupInputDeadlineIsBounded(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Bounded startup")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	begin := time.Now()
	err = ensureInput(ctx, w, "fixture", nil, func(ctx context.Context, _ string, _ []string, _ []byte, validate func([]byte) error) ([]byte, error) {
		<-ctx.Done()
		return nil, contracts.Condition("unavailable", "startup_timeout")
	})
	if time.Since(begin) > qt.Duration("readiness_seconds") || err == nil || err.(*contracts.Error).Reason != "startup_timeout" {
		t.Fatal(err)
	}
	if err = ensureInput(context.Background(), w, "fixture", make([]byte, credentialcmd.MaxInputBytes+1), nil); err == nil || err.(*contracts.Error).Reason != "input_invalid" {
		t.Fatal("unbounded protected input admitted", err)
	}
}

func TestStartupReconcilesOwnerWinningAfterAuthorityCheck(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Closing owner")
	if err != nil {
		t.Fatal(err)
	}
	prepareOwnerStartupCatalog(t, w)
	lock, err := w.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	ownerCtx, stop := context.WithCancel(context.Background())
	defer stop()
	done := make(chan error, 1)
	launches := 0
	start := func(ctx context.Context, _ string, args []string, _ []byte, validate func([]byte) error) ([]byte, error) {
		launches++
		report := contracts.StartupReport{Kind: "runtime-startup", Version: contracts.Version, StartupID: startupID(args), WorkspaceID: w.Config.WorkspaceID, State: "failed", Stage: "startup", Code: "conflict", Reason: "runtime_owner"}
		if launches == 1 {
			// A competing owner wins after the pre-launch authority observation.
			lock, err = w.Lock()
			if err != nil {
				return nil, err
			}
			// Release it only after its conflict was observed.
			raw, _ := json.Marshal(report)
			err := validate(raw)
			lock.Unlock()
			return raw, err
		} else {
			report.State, report.Stage, report.Code, report.Reason = "ready", "ready", "", ""
			ready := make(chan struct{})
			go func() { done <- Serve(ownerCtx, w, Options{Ready: ready, Idle: qt.Duration("fixture_idle_seconds")}) }()
			select {
			case <-ready:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		raw, _ := json.Marshal(report)
		return raw, validate(raw)
	}
	ctx, cancel := qt.Context(context.Background(), "readiness_seconds")
	defer cancel()
	lock.Unlock()
	err = ensureInput(ctx, w, "fixture", nil, start)
	stop()
	if launches > 1 {
		select {
		case <-done:
		case <-time.After(qt.Duration("cleanup_seconds")):
			t.Fatal("replacement owner did not stop")
		}
	}
	if err != nil || launches != 2 {
		t.Fatalf("closing owner prevented restart: launches=%d error=%v", launches, err)
	}
}
