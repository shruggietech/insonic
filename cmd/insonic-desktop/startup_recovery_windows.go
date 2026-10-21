//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/diagnostics"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/wailsapp/go-webview2/pkg/edge"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const startupProfileBlocker = "insonic owned startup recovery fixture\n"

type startupFailureObservation struct {
	SchemaVersion      string `json:"schema_version"`
	ReturnedFailure    string `json:"returned_failure"`
	Stage              string `json:"stage"`
	PrivateStage       string `json:"private_stage"`
	PrivateDiagnostics string `json:"private_diagnostics"`
	OwnedCleanup       string `json:"owned_cleanup"`
}

func controlledStartupStage(stage string) bool {
	switch stage {
	case "profile-directory", "environment-create", "environment-completion", "controller-request", "controller-completion", "browser-process":
		return true
	}
	return false
}

// Only the fixture-owned first profile is obstructed. The next ordinary matrix
// launch uses the same path after verified normal failed-child termination.
func startupRecoveryRunner(runChild func(exitFixture, string) (exitObservation, error)) (func(exitFixture, string) (exitObservation, error), func() bool) {
	return startupRecoveryRunnerWith(runChild, probeStartupFailure, func(raw []byte) {
		retainNativeExitEvidence(raw)
		fmt.Fprintln(os.Stderr, string(raw))
	})
}

func startupRecoveryRunnerWith(runChild func(exitFixture, string) (exitObservation, error), probe func(string) (startupFailureObservation, error), emit func([]byte)) (func(exitFixture, string) (exitObservation, error), func() bool) {
	var attempted, completed bool
	var first startupFailureObservation
	var failure error
	runner := func(fixture exitFixture, config string) (exitObservation, error) {
		if failure != nil {
			return exitObservation{}, failure
		}
		firstLaunch := !attempted
		if firstLaunch {
			attempted = true
			first, failure = probe(config)
			if failure != nil {
				return exitObservation{}, failure
			}
		}
		observation, err := runChild(fixture, config)
		if err != nil {
			failure = err
			return exitObservation{}, err
		}
		if firstLaunch {
			if observation.NativeExit != "passed" || observation.Action != fixture.Action {
				failure = fmt.Errorf("startup recovery relaunch receipt invalid")
				return exitObservation{}, failure
			}
			completed = true
			raw, _ := json.Marshal(map[string]any{"kind": "native-startup-recovery", "schema_version": contracts.Version, "failure_stage": first.Stage, "returned_failure": "passed", "private_diagnostics": "saved", "owned_cleanup": "passed", "same_profile_relaunch": "passed", "directory_access": "current-user-modify"})
			emit(raw)
		}
		return observation, nil
	}
	return runner, func() bool { return completed && failure == nil }
}

func startupFixtureModifyDirectory(directory string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;OICI;0x1301bf;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

func probeStartupFailure(config string) (startupFailureObservation, error) {
	executable, err := os.Executable()
	if err != nil {
		return startupFailureObservation{}, err
	}
	return probeStartupFailureWith(config, executable, process.Capture, func(raw []byte) {
		retainNativeExitEvidence(raw)
		fmt.Fprintln(os.Stderr, string(raw))
	})
}

func probeStartupFailureWith(config, executable string, capture func(context.Context, process.Spec) (process.CaptureResult, error), emit func([]byte)) (startupFailureObservation, error) {
	var observation startupFailureObservation
	directory := filepath.Dir(config)
	if err := startupFixtureModifyDirectory(directory); err != nil {
		return observation, err
	}
	profile := filepath.Join(directory, "webview")
	file, err := os.OpenFile(profile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return observation, err
	}
	_, writeErr := io.WriteString(file, startupProfileBlocker)
	original, statErr := file.Stat()
	closeErr := file.Close()
	if err = errors.Join(writeErr, statErr, closeErr); err != nil {
		return observation, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	report, captureErr := capture(ctx, process.Spec{Executable: executable, Args: []string{"--startup-failure-qualification-child", config}, MaxOutput: 64 << 10, WaitForDescendants: true})
	raw, _ := json.Marshal(map[string]any{"kind": "native-startup-failure-owned-lifecycle", "descendants": report.Descendants, "stdout": string(report.Stdout[:min(len(report.Stdout), 65536)]), "stderr": string(report.Stderr[:min(len(report.Stderr), 65536)]), "exit_code": report.ExitCode, "truncated": report.Truncated, "timed_out": ctx.Err() == context.DeadlineExceeded})
	emit(raw)
	if captureErr != nil || ctx.Err() != nil || report.ExitCode != 0 || report.Truncated || len(report.Stdout) > 65536 || len(report.Stderr) > 65536 {
		return observation, fmt.Errorf("controlled startup failure child did not return normally: %w", errors.Join(captureErr, ctx.Err(), fmt.Errorf("exit=%d truncated=%t", report.ExitCode, report.Truncated)))
	}
	if report.Descendants == nil || !report.Descendants.Released || report.Descendants.Remaining != 0 {
		return observation, fmt.Errorf("controlled startup failure descendants not released")
	}
	for _, child := range report.Descendants.Processes {
		if !child.Observed || child.Running {
			return observation, fmt.Errorf("controlled startup failure descendant still unobserved or running")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(report.Stdout))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&observation); err != nil {
		return observation, fmt.Errorf("controlled startup failure receipt: %w", err)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || observation.SchemaVersion != contracts.Version || observation.ReturnedFailure != "passed" || !controlledStartupStage(observation.Stage) || observation.PrivateStage != observation.Stage || observation.PrivateDiagnostics != "saved" || observation.OwnedCleanup != "passed" {
		return observation, fmt.Errorf("controlled startup failure receipt invalid")
	}
	current, err := os.Lstat(profile)
	if err != nil {
		return observation, err
	}
	contents, err := os.ReadFile(profile)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(original, current) || string(contents) != startupProfileBlocker {
		return observation, fmt.Errorf("controlled startup profile blocker changed; retained")
	}
	// This is the exact regular file created above, after child/job release. No
	// existing browser profile, arbitrary directory or user data is removed.
	if err = os.Remove(profile); err != nil {
		return observation, err
	}
	return observation, nil
}

func qualifyStartupFailureChild(config string) int {
	fixture, err := readExitFixture(config)
	if err != nil || fixture.Workspace != "" {
		fmt.Fprintln(os.Stderr, "invalid startup recovery fixture")
		return 1
	}
	directory := filepath.Dir(config)
	profile := filepath.Join(directory, "webview")
	st, err := os.Lstat(profile)
	raw, readErr := os.ReadFile(profile)
	if err != nil || readErr != nil || !st.Mode().IsRegular() || string(raw) != startupProfileBlocker {
		fmt.Fprintln(os.Stderr, "startup recovery blocker unavailable")
		return 1
	}
	var started atomic.Bool
	unexpected := func(ctx context.Context) { started.Store(true); wailsruntime.Quit(ctx) }
	fmt.Fprintln(os.Stderr, "native-startup-recovery-stage=before-application")
	finished := make(chan struct{})
	go func() {
		timer := time.NewTimer(12 * time.Second)
		defer timer.Stop()
		select {
		case <-finished:
		case <-timer.C:
			fmt.Fprintln(os.Stderr, "native-startup-recovery-stage=watchdog-observation")
			logNativeUIFunctionStacks()
			logNativeStartupProgress()
		}
	}()
	primary := runDesktopApplicationWithProfile(&desktop.Bridge{CLIExecutable: desktop.SiblingCLI()}, unexpected, unexpected, true, profile)
	close(finished)
	fmt.Fprintln(os.Stderr, "native-startup-recovery-stage=application-returned")
	var native *edge.StartupError
	if primary == nil || !errors.As(primary, &native) || !controlledStartupStage(native.Stage) || started.Load() {
		fmt.Fprintln(os.Stderr, "controlled startup failure did not return from the expected native boundary", primary)
		return 1
	}
	path := filepath.Join(directory, "startup.jsonl")
	var diagnosticErr error
	reportDesktopStartupFailure(primary, false, os.Stderr, func(event diagnostics.Event) (string, error) {
		var saved string
		saved, diagnosticErr = saveStartupEventAt(path, event)
		return saved, diagnosticErr
	}, nil)
	if diagnosticErr != nil {
		return 1
	}
	raw, err = os.ReadFile(path)
	var event diagnostics.Event
	if err != nil || json.Unmarshal(bytes.TrimSpace(raw), &event) != nil || event.Operation != "desktop.startup" || event.FailureStage != native.Stage || event.State != "failed" {
		fmt.Fprintln(os.Stderr, "startup private diagnostic receipt invalid")
		return 1
	}
	if err = nativeWebviewCloseError(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "native-startup-recovery-stage=before-apartment-cleanup")
	if err = nativeCOMCleanup(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err = json.NewEncoder(os.Stdout).Encode(startupFailureObservation{SchemaVersion: contracts.Version, ReturnedFailure: "passed", Stage: native.Stage, PrivateStage: event.FailureStage, PrivateDiagnostics: "saved", OwnedCleanup: "passed"}); err != nil {
		return 1
	}
	return 0
}
