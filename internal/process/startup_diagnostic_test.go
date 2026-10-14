// SPDX-License-Identifier: Apache-2.0
package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func diagnosticFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("INSONIC_PROCESS_STARTUP_DIAGNOSTICS", "1")
	t.Setenv("GITHUB_WORKSPACE", root)
	previous := startupDiagnosticSequence.Swap(0)
	t.Cleanup(func() { startupDiagnosticSequence.Store(previous) })
	return root
}

func TestStartupDiagnosticRequiresExplicitCIAndAbsoluteOwnedWorkspace(t *testing.T) {
	for _, invalid := range []string{"ci", "election", "workspace"} {
		t.Run(invalid, func(t *testing.T) {
			root := diagnosticFixture(t)
			switch invalid {
			case "ci":
				t.Setenv("GITHUB_ACTIONS", "false")
			case "election":
				t.Setenv("INSONIC_PROCESS_STARTUP_DIAGNOSTICS", "")
			case "workspace":
				t.Setenv("GITHUB_WORKSPACE", "relative-fixture")
			}
			writeStartupDiagnostic("command_start", syscall.Errno(5))
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatal("diagnostic wrote outside the elected CI boundary", entries, err)
			}
		})
	}
}

func TestStartupDiagnosticRecordsOnlyStageAndNumericErrorWithoutChangingAPI(t *testing.T) {
	root := diagnosticFixture(t)
	secret := "private-provider-token-and-command-arguments"
	t.Setenv("INSONIC_PROVIDER_TOKEN", secret)
	cause := &os.PathError{Op: secret, Path: filepath.Join(root, secret), Err: fmt.Errorf("%s: %w", secret, syscall.Errno(5))}
	for _, stage := range []string{"stdout_pipe", "stdin_pipe", "command_start"} {
		err := startupLaunchFailure(stage, cause)
		typed, ok := err.(*contracts.Error)
		if !ok || typed.Code != "unavailable" || typed.Reason != "startup_launch" || strings.Contains(err.Error(), secret) {
			t.Fatal("public startup failure contract changed", err)
		}
	}
	files, err := filepath.Glob(filepath.Join(root, "build", "native", "process-startup-*-receipt.json"))
	if err != nil || len(files) != 3 {
		t.Fatal("launch stages were not recorded", files, err)
	}
	stages := map[string]bool{}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil || len(raw) > 1024 || strings.Contains(string(raw), secret) || strings.Contains(string(raw), root) {
			t.Fatal("private values leaked or diagnostic exceeded its bound", err)
		}
		var record map[string]any
		if json.Unmarshal(raw, &record) != nil || len(record) != 6 || record["kind"] != "process-startup-diagnostic" || record["has_errno"] != true || record["error_number"] != float64(5) {
			t.Fatal("missing sanitized native error", record)
		}
		stages[record["stage"].(string)] = true
	}
	if len(stages) != 3 {
		t.Fatal("stage identities were lost", stages)
	}
}

func TestStartupDiagnosticIsBoundedAndDoesNotSerializeUnknownErrors(t *testing.T) {
	root := diagnosticFixture(t)
	writeStartupDiagnostic("private-stage-token", fmt.Errorf("private cause"))
	for i := 0; i < 20; i++ {
		writeStartupDiagnostic("command_start", fmt.Errorf("private cause without native errno"))
	}
	files, err := filepath.Glob(filepath.Join(root, "build", "native", "process-startup-*-receipt.json"))
	if err != nil || len(files) != 16 {
		t.Fatal("diagnostic event bound changed", len(files), err)
	}
	raw, err := os.ReadFile(files[0])
	var record map[string]any
	if err != nil || json.Unmarshal(raw, &record) != nil || record["has_errno"] != false || record["error_number"] != float64(0) || strings.Contains(string(raw), "private") {
		t.Fatal("unknown error contents leaked", record, err)
	}
}

func TestStartupDiagnosticWriteFailureDoesNotChangeStartupError(t *testing.T) {
	root := diagnosticFixture(t)
	if err := os.WriteFile(filepath.Join(root, "build"), []byte("fixture blocks diagnostic output"), 0600); err != nil {
		t.Fatal(err)
	}
	err := startupLaunchFailure("command_start", syscall.Errno(5))
	if typed, ok := err.(*contracts.Error); !ok || typed.Code != "unavailable" || typed.Reason != "startup_launch" {
		t.Fatal("optional diagnostic failure replaced product error", err)
	}
}
