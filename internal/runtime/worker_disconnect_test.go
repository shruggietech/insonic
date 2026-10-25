// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	qt "github.com/shruggietech/insonic/internal/qualificationtiming"
	"github.com/shruggietech/insonic/internal/workspace"
)

// The actual owned test executable acts as a blocked interpreter. No shell or
// inherited test-only environment variable is needed by the clean probe child.
func init() {
	if len(os.Args) != 8 || os.Args[1] != "-I" || os.Args[2] != "-B" || os.Args[3] != "-c" || !strings.HasPrefix(os.Args[4], "# SPDX-License-Identifier: Apache-2.0") || os.Args[5] != "recognition" || os.Args[6] != "inspect" {
		return
	}
	point := func(stage string) map[string]any {
		return map[string]any{"stage": stage, "elapsed_ms": 0, "files": 0, "hashed_files": 0, "hashed_bytes": 0}
	}
	raw, _ := json.Marshal(map[string]any{"mode": "inspect", "stage": "library_import", "reason": "running", "elapsed_ms": 0, "files": 0, "hashed_files": 0, "hashed_bytes": 0, "checkpoints": []any{point("package_versions"), point("library_import")}})
	if os.WriteFile(os.Args[7], raw, 0600) != nil {
		os.Exit(97)
	}
	time.Sleep(time.Minute)
	os.Exit(98)
}
func TestDisconnectedWorkerInspectionStopsOwnedProbe(t *testing.T) {
	profile, err := os.MkdirTemp("", "insonic-probe-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if filepath.Dir(profile) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(profile), "insonic-probe-") {
			t.Fatal("unexpected owned profile")
		}
		if err := os.RemoveAll(profile); err != nil {
			t.Error(err)
		}
	})
	for _, key := range []string{"APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_RUNTIME_DIR"} {
		t.Setenv(key, profile)
	}
	w, err := workspace.Init(t.TempDir(), "probe disconnect")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, w, Options{Idle: qt.Duration("fixture_idle_seconds"), Ready: ready}) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(qt.Duration("cleanup_seconds")):
			t.Error("owned runtime did not stop")
		}
	}()
	select {
	case <-ready:
	case err := <-done:
		done <- err
		t.Fatal("runtime exited before readiness")
	case <-time.After(qt.Duration("readiness_seconds")):
		t.Fatal("runtime startup")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]any{"role": "recognition", "python_path": executable})
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "workers.inspect", Data: data}
	conn, err := dial(ctx, w)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	raw, _ := json.Marshal(request)
	if _, err = conn.Write(append(raw, '\n')); err != nil {
		t.Fatal(err)
	}
	var home string
	deadline := time.Now().Add(qt.Duration("readiness_seconds"))
	for home == "" && time.Now().Before(deadline) {
		entries, _ := os.ReadDir(w.Control)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "worker-probe-") {
				candidate := filepath.Join(w.Control, entry.Name())
				if _, err := os.Stat(filepath.Join(candidate, "probe-progress.json")); err == nil {
					home = candidate
					break
				}
			}
		}
		if home == "" {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if home == "" {
		t.Fatal("actual owned interpreter did not reach its blocked import")
	}
	conn.Close()
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(home); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("disconnected ephemeral request left its owned probe running")
}
