//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/process"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestExitFixtureRejectsAmbiguousOrUnboundedInput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "fixture.json")
	for _, raw := range []string{
		"{}", "{\"action\":\"kill\"}", "{\"action\":\"menu\",\"force\":true}",
		"{\"action\":\"menu\"} {}", "{\"action\":\"menu\",\"workspace\":\"relative\"}",
		"{\"action\":\"menu\",\"media_id\":\"missing-workspace\"}",
	} {
		os.WriteFile(file, []byte(raw), 0600)
		if _, err := readExitFixture(file); err == nil {
			t.Fatalf("accepted invalid fixture %s", raw)
		}
	}
	for _, action := range []string{"menu", "close"} {
		os.WriteFile(file, []byte("{\"action\":\""+action+"\"}"), 0600)
		fixture, err := readExitFixture(file)
		if err != nil || fixture.Action != action {
			t.Fatal(fixture, err)
		}
	}
}
func TestExitObserverRejectsRuntimeReplacementWithoutRestart(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Exit observer")
	if err != nil {
		t.Fatal(err)
	}
	r := &exitRuntime{workspace: w}
	if _, err = r.call("workspace.show", "", nil); err == nil {
		t.Fatal("unavailable runtime was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready, done := make(chan struct{}), make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready, Idle: 10 * time.Second}) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("runtime did not become ready")
	}
	original, err := r.call("workspace.show", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.session = contracts.ID()
	if _, err = r.call("workspace.show", "", nil); err == nil {
		t.Fatal("replacement session was accepted")
	}
	r.session = original.SessionID
	if _, err = r.call("workspace.show", "", nil); err != nil {
		t.Fatal(err)
	}
}

// This checks the actual media/acquisition/IPC fixture without launching Wails.
// It cannot satisfy the source/package native activation receipt.
func TestExitOwnerFixtureReleasesWorkAfterClientCleanup(t *testing.T) {
	if os.Getenv("CUESON_EXECUTABLE") == "" {
		t.Skip("native qualification companions not prepared")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("INSONIC_DESKTOP_FIXTURE_DIRECTORY", filepath.Join(root, "tests", "fixtures", "media"))
	if os.Getenv("INSONIC_LIBRARY_TOOLS_FILE") == "" {
		t.Setenv("INSONIC_LIBRARY_TOOLS_FILE", filepath.Join(root, "build", "native", "media-tools.json"))
	}
	calls := 0
	profiles := map[string]int{}
	combinations := map[string]int{}
	err = runExitQualificationWith(func(fixture exitFixture, config string) (exitObservation, error) {
		calls++
		profiles[config]++
		combinations[fmt.Sprintf("%s:%t", fixture.Action, fixture.Workspace != "")]++
		observation := exitObservation{Action: fixture.Action, NativeExit: "passed", Playback: "not-selected"}
		if fixture.Workspace == "" {
			return observation, nil
		}
		w, err := workspace.Open(fixture.Workspace)
		if err != nil {
			return observation, err
		}
		bridge := &desktop.Bridge{Workspace: w, CLIExecutable: desktop.SiblingCLI()}
		stop, err := desktop.StartPlaybackTransport(bridge)
		if err != nil {
			return observation, err
		}
		defer stop()
		id, verify, err := desktop.ObserveExitPlaybackQualification(bridge, fixture.MediaID, fixture.Revision)
		if err != nil {
			return observation, err
		}
		desktop.ClosePlaybackHandles(bridge)
		stop()
		if err = verify(); err != nil {
			return observation, err
		}
		observation.Playback = "released"
		observation.PlaybackID = id
		return observation, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if calls != 12 || len(profiles) != 6 || len(combinations) != 6 {
			t.Fatalf("cold/warm native matrix incomplete: %d %v %v", calls, profiles, combinations)
		}
		for _, count := range profiles {
			if count != 2 {
				t.Fatal("profile was not reused exactly once after cold launch")
			}
		}
	} else if calls != 3 {
		t.Fatal("fixture cases were skipped")
	}
}

func TestExitFailureFixtureProcess(t *testing.T) {
	if os.Getenv("INSONIC_EXIT_FAILURE_FIXTURE") != "1" {
		return
	}
	fmt.Fprintln(os.Stdout, "[WebView2 Error] synthetic controller HRESULT 8007139f")
	fmt.Fprintln(os.Stderr, "synthetic controller stack")
	os.Exit(23)
}

func TestExitChildFailureRetainsBothActualStreams(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	report, captureErr := process.Capture(ctx, process.Spec{Executable: executable, Args: []string{"-test.run=^TestExitFailureFixtureProcess$"}, Env: append(os.Environ(), "INSONIC_EXIT_FAILURE_FIXTURE=1"), MaxOutput: 65536})
	var output bytes.Buffer
	_, err = interpretExitChild(exitFixture{Action: "menu"}, report, captureErr, ctx.Err(), &output)
	if err == nil {
		t.Fatal("failed native child accepted")
	}
	var evidence nativeExitFailure
	if err = json.Unmarshal(output.Bytes(), &evidence); err != nil {
		t.Fatal(err, output.String())
	}
	if !strings.Contains(evidence.Stdout, "synthetic controller HRESULT 8007139f") || !strings.Contains(evidence.Stderr, "synthetic controller stack") || evidence.ExitCode != 23 || evidence.Action != "menu" || evidence.Kind != "native-exit-child-failure" {
		t.Fatalf("lost native error: %+v", evidence)
	}
}

func TestExitChildFailureClassificationsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name                string
		context             error
		truncated           bool
		timedOut, cancelled bool
	}{
		{"failed", nil, false, false, false}, {"deadline", context.DeadlineExceeded, false, true, false}, {"cancelled", context.Canceled, false, false, true}, {"truncated", nil, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			report := process.CaptureResult{Stdout: []byte(strings.Repeat("x", 65537)), Stderr: []byte("native-exit-stage=controller-startup\nstack"), ExitCode: 7, Truncated: tc.truncated}
			_, err := interpretExitChild(exitFixture{Action: "close"}, report, contracts.Fail("operation_failed"), tc.context, &output)
			var evidence nativeExitFailure
			if err == nil || json.Unmarshal(output.Bytes(), &evidence) != nil || len(evidence.Stdout) != 65536 || !evidence.Truncated || evidence.TimedOut != tc.timedOut || evidence.Cancelled != tc.cancelled || evidence.Stage != "controller-startup" {
				t.Fatalf("wrong failure classification: %v %s", err, output.String())
			}
		})
	}
}

func TestExitChildSuccessIsUnchangedAndInvalidSuccessFails(t *testing.T) {
	for _, raw := range []string{`{"action":"menu","native_exit":"passed","playback":"not-selected"}`, "not JSON", `{"action":"close","native_exit":"passed"}`} {
		var output bytes.Buffer
		observation, err := interpretExitChild(exitFixture{Action: "menu"}, process.CaptureResult{Stdout: []byte(raw), ExitCode: 0}, nil, nil, &output)
		if strings.HasPrefix(raw, `{"action":"menu"`) {
			if err != nil || output.Len() != 0 || observation.NativeExit != "passed" {
				t.Fatal(err, output.String())
			}
		} else if err == nil || output.Len() == 0 {
			t.Fatal("invalid success receipt accepted")
		}
	}
}

func TestExitFailureArtifactRetainsBothStreamsWithoutSuccessOutput(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("INSONIC_PROCESS_STARTUP_DIAGNOSTICS", "1")
	t.Setenv("GITHUB_WORKSPACE", root)
	var output bytes.Buffer
	_, err := interpretExitChild(exitFixture{Action: "menu"}, process.CaptureResult{Stdout: []byte("native HRESULT"), Stderr: []byte("stack"), ExitCode: 7}, contracts.Fail("operation_failed"), nil, &output)
	if err == nil {
		t.Fatal("failure became success")
	}
	files, err := filepath.Glob(filepath.Join(root, "build", "native", "native-exit-child-*-receipt.json"))
	if err != nil || len(files) != 1 {
		t.Fatal(files, err)
	}
	raw, err := os.ReadFile(files[0])
	if err != nil || !bytes.Equal(raw, output.Bytes()) {
		t.Fatal("artifact differs from emitted evidence", err)
	}
}

func TestExitChildOwnWatchdogIsReportedAsTimeout(t *testing.T) {
	var output bytes.Buffer
	_, err := interpretExitChild(exitFixture{Action: "menu"}, process.CaptureResult{Stderr: []byte("native-exit-stage=controller-startup\nnative-exit-stage=watchdog-timeout\n"), ExitCode: 1}, contracts.Fail("operation_failed"), nil, &output)
	var evidence nativeExitFailure
	if err == nil || json.Unmarshal(output.Bytes(), &evidence) != nil || !evidence.TimedOut || evidence.Cancelled || evidence.Stage != "watchdog-timeout" {
		t.Fatalf("lost fixture timeout: %v %s", err, output.String())
	}
}

func TestExitOwnerFailureRetainsItsProfileForDiagnosis(t *testing.T) {
	if os.Getenv("CUESON_EXECUTABLE") == "" {
		t.Skip("native qualification companions not prepared")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("INSONIC_DESKTOP_FIXTURE_DIRECTORY", filepath.Join(root, "tests", "fixtures", "media"))
	var retained, marker string
	failure := fmt.Errorf("synthetic native child failure")
	err = runExitQualificationWith(func(_ exitFixture, config string) (exitObservation, error) {
		retained = filepath.Dir(filepath.Dir(config))
		t.Cleanup(func() { os.RemoveAll(retained) })
		profile := filepath.Join(filepath.Dir(config), "webview")
		if err := os.Mkdir(profile, 0700); err != nil {
			return exitObservation{}, err
		}
		marker = filepath.Join(profile, "retained.txt")
		if err := os.WriteFile(marker, []byte("owned failed profile"), 0600); err != nil {
			return exitObservation{}, err
		}
		return exitObservation{}, failure
	})
	if err != failure {
		t.Fatal("native failure was replaced", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("failed native profile was removed before observed release", err)
	}
	if _, err := os.Stat(filepath.Join(retained, ".insonic", "workspace.json")); err != nil {
		t.Fatal("failed fixture authority was not retained", err)
	}
}
