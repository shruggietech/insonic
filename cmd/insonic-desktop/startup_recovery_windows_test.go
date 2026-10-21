//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/diagnostics"
	"github.com/shruggietech/insonic/internal/process"
)

func goodStartupFailure() startupFailureObservation {
	return startupFailureObservation{SchemaVersion: contracts.Version, ReturnedFailure: "passed", Stage: "profile-directory", PrivateStage: "profile-directory", PrivateDiagnostics: "saved", OwnedCleanup: "passed"}
}

func TestStartupRecoveryWrapsFirstLaunchAndRetainsAllTwelveCases(t *testing.T) {
	probes, launches, receipts := 0, 0, 0
	run, completed := startupRecoveryRunnerWith(func(f exitFixture, config string) (exitObservation, error) {
		if probes != 1 {
			t.Fatal("valid launch preceded failed launch")
		}
		launches++
		return exitObservation{Action: f.Action, NativeExit: "passed"}, nil
	}, func(string) (startupFailureObservation, error) { probes++; return goodStartupFailure(), nil }, func(raw []byte) {
		receipts++
		var proof map[string]any
		if json.Unmarshal(raw, &proof) != nil || proof["same_profile_relaunch"] != "passed" {
			t.Fatal(string(raw))
		}
	})
	if completed() {
		t.Fatal("recovery accepted before launch")
	}
	for i := 0; i < 12; i++ {
		if _, err := run(exitFixture{Action: "menu"}, "fixture.json"); err != nil {
			t.Fatal(err)
		}
	}
	if !completed() || probes != 1 || launches != 12 || receipts != 1 {
		t.Fatal(completed(), probes, launches, receipts)
	}
}

func TestStartupRecoveryFailureIsStickyWithoutRetryOrSuccessReceipt(t *testing.T) {
	for _, failProbe := range []bool{true, false} {
		t.Run(map[bool]string{true: "failure-probe", false: "valid-relaunch"}[failProbe], func(t *testing.T) {
			original := errors.New("original failure")
			probes, launches, receipts := 0, 0, 0
			run, completed := startupRecoveryRunnerWith(func(f exitFixture, _ string) (exitObservation, error) { launches++; return exitObservation{}, original }, func(string) (startupFailureObservation, error) {
				probes++
				if failProbe {
					return startupFailureObservation{}, original
				}
				return goodStartupFailure(), nil
			}, func([]byte) { receipts++ })
			for i := 0; i < 2; i++ {
				if _, err := run(exitFixture{Action: "menu"}, "fixture.json"); !errors.Is(err, original) {
					t.Fatal(err)
				}
			}
			expected := 1
			if failProbe {
				expected = 0
			}
			if completed() || probes != 1 || launches != expected || receipts != 0 {
				t.Fatal(completed(), probes, launches, receipts)
			}
		})
	}
}

func TestStartupRecoveryRejectsInvalidValidLaunchReceipt(t *testing.T) {
	run, completed := startupRecoveryRunnerWith(func(exitFixture, string) (exitObservation, error) { return exitObservation{}, nil }, func(string) (startupFailureObservation, error) { return goodStartupFailure(), nil }, func([]byte) { t.Fatal("invalid success emitted") })
	if _, err := run(exitFixture{Action: "menu"}, "fixture.json"); err == nil || completed() {
		t.Fatal("invalid relaunch accepted")
	}
}

func TestControlledStartupFailureRequiresNaturalReleaseBeforeOwnedBlockerRemoval(t *testing.T) {
	cases := []string{"success", "capture-error", "nonzero-exit", "truncated", "no-descendants", "unreleased", "remaining", "unobserved", "running", "wrong-stage", "private-stage-mismatch", "unknown-field", "trailing-json", "changed-blocker", "replaced-blocker"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			config := filepath.Join(directory, "native-exit.json")
			profile := filepath.Join(directory, "webview")
			if err := os.WriteFile(config, []byte("fixture bytes retained"), 0600); err != nil {
				t.Fatal(err)
			}
			emitted := 0
			observed, err := probeStartupFailureWith(config, filepath.Join(directory, "desktop.exe"), func(ctx context.Context, spec process.Spec) (process.CaptureResult, error) {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 18*time.Second || time.Until(deadline) < 17*time.Second || !spec.WaitForDescendants || spec.MaxOutput != 64<<10 || len(spec.Args) != 2 || spec.Args[0] != "--startup-failure-qualification-child" || spec.Args[1] != config {
					t.Fatal("child contract changed", spec)
				}
				contents, e := os.ReadFile(profile)
				if e != nil || string(contents) != startupProfileBlocker {
					t.Fatal("missing controlled profile blocker", e)
				}
				observation := goodStartupFailure()
				report := process.CaptureResult{Descendants: &process.DescendantReport{Released: true, Processes: []process.DescendantProcess{{PID: 42, Observed: true}}}}
				switch name {
				case "capture-error":
					return report, errors.New("capture failed")
				case "nonzero-exit":
					report.ExitCode = 1
				case "truncated":
					report.Truncated = true
				case "no-descendants":
					report.Descendants = nil
				case "unreleased":
					report.Descendants.Released = false
				case "remaining":
					report.Descendants.Remaining = 1
				case "unobserved":
					report.Descendants.Processes[0].Observed = false
				case "running":
					report.Descendants.Processes[0].Running = true
				case "wrong-stage":
					observation.Stage = "private-input"
				case "private-stage-mismatch":
					observation.PrivateStage = "desktop-window"
				case "changed-blocker":
					if e := os.WriteFile(profile, []byte("changed"), 0600); e != nil {
						t.Fatal(e)
					}
				case "replaced-blocker":
					if e := os.Rename(profile, profile+".original"); e != nil {
						t.Fatal(e)
					}
					if e := os.WriteFile(profile, []byte(startupProfileBlocker), 0600); e != nil {
						t.Fatal(e)
					}
				}
				report.Stdout, _ = json.Marshal(observation)
				if name == "unknown-field" {
					report.Stdout = []byte(strings.TrimSuffix(string(report.Stdout), "}") + ",\"private_payload\":\"untrusted\"}")
				}
				if name == "trailing-json" {
					report.Stdout = append(report.Stdout, []byte(" {}")...)
				}
				return report, nil
			}, func(raw []byte) {
				emitted++
				var proof map[string]any
				if json.Unmarshal(raw, &proof) != nil || proof["kind"] != "native-startup-failure-owned-lifecycle" {
					t.Fatal(string(raw))
				}
			})
			if emitted != 1 {
				t.Fatal("lifecycle evidence omitted", emitted)
			}
			_, stErr := os.Lstat(profile)
			if name == "success" {
				if err != nil || observed.Stage != "profile-directory" || !os.IsNotExist(stErr) {
					t.Fatal(observed, err, stErr)
				}
			} else if err == nil || stErr != nil {
				t.Fatal("failed fixture was accepted or blocker removed", err, stErr)
			}
			if raw, e := os.ReadFile(config); e != nil || string(raw) != "fixture bytes retained" {
				t.Fatal("unrelated fixture changed", e)
			}
		})
	}
}

func TestControlledStartupFailureNeverOverwritesExistingProfile(t *testing.T) {
	directory := t.TempDir()
	profile := filepath.Join(directory, "webview")
	if err := os.Mkdir(profile, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := probeStartupFailureWith(filepath.Join(directory, "native-exit.json"), filepath.Join(directory, "desktop.exe"), func(context.Context, process.Spec) (process.CaptureResult, error) {
		t.Fatal("child launched for an existing profile")
		return process.CaptureResult{}, nil
	}, func([]byte) {}); err == nil {
		t.Fatal("existing profile admitted")
	}
	if st, err := os.Stat(profile); err != nil || !st.IsDir() {
		t.Fatal("existing profile removed", err)
	}
}

func TestControlledStartupPrivateDiagnosticChild(t *testing.T) {
	if os.Getenv("INSONIC_TEST_STARTUP_DIAGNOSTIC_CHILD") != "1" {
		return
	}
	directory := os.Args[len(os.Args)-1]
	if _, err := saveStartupEventAt(filepath.Join(directory, "startup.jsonl"), diagnostics.Event{Operation: "desktop.startup", FailureStage: "profile-directory", State: "failed"}); err != nil {
		fmt.Fprintln(os.Stderr, "controlled diagnostic child:", err)
		logStartupDiagnosticAccess(directory)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestControlledStartupPrivateDiagnosticsWorkInModifyDirectoryAcrossOwnedChild(t *testing.T) {
	directory := t.TempDir()
	if err := startupFixtureModifyDirectory(directory); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := process.Capture(ctx, process.Spec{Executable: executable, Args: []string{"-test.run=^TestControlledStartupPrivateDiagnosticChild$", "--", directory}, Env: []string{"INSONIC_TEST_STARTUP_DIAGNOSTIC_CHILD=1"}, MaxOutput: 64 << 10, WaitForDescendants: true})
	if err != nil || result.ExitCode != 0 || result.Truncated || result.Descendants == nil || !result.Descendants.Released {
		t.Fatalf("owned child private diagnostics failed: %v exit=%d stderr=%s", err, result.ExitCode, result.Stderr)
	}
	raw, err := os.ReadFile(filepath.Join(directory, "startup.jsonl"))
	var event diagnostics.Event
	if err != nil || json.Unmarshal(raw, &event) != nil || event.FailureStage != "profile-directory" {
		t.Fatal("private child event missing", err)
	}
}

func TestControlledPreControllerFailureRequiresNoControllerAndBalancedCleanup(t *testing.T) {
	called := 0
	if err := finishControlledStartupFailure(nil, 0, func() error { called++; return nil }); err != nil || called != 1 {
		t.Fatal("pre-controller failure must not require a successful controller close", err, called)
	}
	for _, count := range []uint32{1, 2} {
		if err := finishControlledStartupFailure(nil, count, func() error { t.Fatal("unexpected controller admitted"); return nil }); err == nil {
			t.Fatal("unexpected controller creation was not rejected")
		}
	}
	original := errors.New("original close failure")
	if err := finishControlledStartupFailure(original, 0, func() error { t.Fatal("close failure replaced"); return nil }); !errors.Is(err, original) {
		t.Fatal(err)
	}
	if err := finishControlledStartupFailure(nil, 0, func() error { return original }); !errors.Is(err, original) {
		t.Fatal("cleanup failure lost", err)
	}
}
