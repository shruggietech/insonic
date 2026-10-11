// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"strings"
	"sync"
	"testing"
	"time"
)

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
		start := func(_ context.Context, _ string, args []string, input []byte) ([]byte, error) {
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
			return raw, nil
		}
		begin := time.Now()
		err = ensureInput(context.Background(), w, "fixture", nil, start)
		typed, ok := err.(*contracts.Error)
		want := "startup_report"
		if mode == "failed" {
			want = "runtime_catalog"
		}
		if !ok || typed.Reason != want || time.Since(begin) > time.Second || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal(mode, err)
		}
	}
}
func TestConcurrentColdStartupLoadsProtectedInputIntoOneOwner(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Concurrent startup")
	if err != nil {
		t.Fatal(err)
	}
	if err = secrets.Select(w, "session"); err != nil {
		t.Fatal(err)
	}
	ownerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var owners sync.WaitGroup
	launch := func(ctx context.Context, _ string, args []string, raw []byte) ([]byte, error) {
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
			err := Serve(ownerCtx, w, Options{Input: input, ReportReady: func() { reports <- report }})
			if err != nil {
				safe := contracts.SafeError(err)
				report.State, report.Stage, report.Code, report.Reason = "failed", "startup", safe.Code, safe.Reason
				reports <- report
			}
		}()
		select {
		case report = <-reports:
			raw, _ := json.Marshal(report)
			return raw, nil
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
	if err = ensureInput(context.Background(), w, "fixture", nil, func(context.Context, string, []string, []byte) ([]byte, error) {
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
	err = ensureInput(ctx, w, "fixture", nil, func(ctx context.Context, _ string, _ []string, _ []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, contracts.Condition("unavailable", "startup_timeout")
	})
	if time.Since(begin) > time.Second || err == nil || err.(*contracts.Error).Reason != "startup_timeout" {
		t.Fatal(err)
	}
	if err = ensureInput(context.Background(), w, "fixture", make([]byte, credentialcmd.MaxInputBytes+1), nil); err == nil || err.(*contracts.Error).Reason != "input_invalid" {
		t.Fatal("unbounded protected input admitted", err)
	}
}
