//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/process"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type exitFixture struct {
	Action    string "json:\"action\""
	Workspace string "json:\"workspace,omitempty\""
	MediaID   string "json:\"media_id,omitempty\""
	Revision  int64  `json:"revision,omitempty"`
}
type exitObservation struct {
	Action     string "json:\"action\""
	NativeExit string "json:\"native_exit\""
	Playback   string "json:\"playback\""
	PlaybackID string "json:\"playback_id,omitempty\""
}

func readExitFixture(path string) (exitFixture, error) {
	var fixture exitFixture
	raw, err := os.ReadFile(path)
	if err != nil {
		return fixture, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if len(raw) > 8192 || decoder.Decode(&fixture) != nil {
		return fixture, fmt.Errorf("invalid Exit fixture")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || fixture.Action != "menu" && fixture.Action != "close" && (fixture.Action != "alt-f4" || runtime.GOOS != "windows") {
		return fixture, fmt.Errorf("invalid Exit action")
	}
	if fixture.Workspace != "" && (!filepath.IsAbs(fixture.Workspace) || !contracts.ValidID(fixture.MediaID) || fixture.Revision < 1) || fixture.Workspace == "" && (fixture.MediaID != "" || fixture.Revision != 0) {
		return fixture, fmt.Errorf("invalid Exit workspace")
	}
	return fixture, nil
}

// Only this child owns Wails. Its parent owns the real shared runtime and
// verifies survival through direct IPC after the GUI process has returned.
func qualifyNativeExitChild(path string) int {
	fixture, err := readExitFixture(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	bridge := &desktop.Bridge{CLIExecutable: desktop.SiblingCLI()}
	if fixture.Workspace != "" {
		bridge.Workspace, err = workspace.Open(fixture.Workspace)
		if err != nil {
			return 1
		}
	}
	ready, finished := make(chan struct{}), make(chan struct{})
	activated := make(chan error, 1)
	var timedOut atomic.Bool
	var prepareErr error
	var verify func() error
	var playbackID string
	startup := func(ctx context.Context) {
		if bridge.Workspace != nil {
			playbackID, verify, prepareErr = desktop.ObserveExitPlaybackQualification(bridge, fixture.MediaID, fixture.Revision)
		}
		close(ready)
		go func() {
			select {
			case <-finished:
			case <-time.After(12 * time.Second):
				timedOut.Store(true)
				wailsruntime.Quit(ctx)
			}
		}()
	}
	domReady := func(ctx context.Context) {
		fmt.Fprintln(os.Stderr, "native-exit-stage=dom-ready")
		go func() {
			select {
			case <-ready:
			case <-time.After(8 * time.Second):
				activated <- fmt.Errorf("startup did not become ready")
				wailsruntime.Quit(ctx)
				return
			}
			if prepareErr != nil {
				activated <- prepareErr
				wailsruntime.Quit(ctx)
				return
			}
			// CREATE_NO_WINDOW remains mandatory for the console launcher. Its
			// STARTUPINFO hide flag can suppress the first native show call, so this
			// explicit disposable-UI probe requests the normal Wails show/activate path.
			if runtime.GOOS == "windows" {
				wailsruntime.WindowShow(ctx)
			}
			fmt.Fprintln(os.Stderr, "native-exit-stage=native-action")
			err := activateNativeExit(fixture.Action)
			activated <- err
			if err != nil {
				wailsruntime.Quit(ctx)
			}
		}()
	}
	fmt.Fprintln(os.Stderr, "native-exit-stage=controller-startup")
	err = runDesktopApplicationWithProfile(bridge, startup, domReady, false, filepath.Join(filepath.Dir(path), "webview"))
	fmt.Fprintln(os.Stderr, "native-exit-stage=shutdown")
	close(finished)
	if timedOut.Load() {
		fmt.Fprintln(os.Stderr, "native-exit-stage=watchdog-timeout")
	}
	if err != nil || timedOut.Load() {
		fmt.Fprintln(os.Stderr, "Native Exit did not finish normally.")
		return 1
	}
	select {
	case err = <-activated:
	case <-time.After(time.Second):
		err = fmt.Errorf("native action was not observed")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	playback := "not-selected"
	if verify != nil {
		if err = verify(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		playback = "released"
	}
	json.NewEncoder(os.Stdout).Encode(exitObservation{Action: fixture.Action, NativeExit: "passed", Playback: playback, PlaybackID: playbackID})
	return 0
}

type exitRuntime struct {
	workspace *workspace.Workspace
	session   string
}

func (r *exitRuntime) call(operation, item string, data any) (contracts.Response, error) {
	var raw json.RawMessage
	var err error
	if data != nil {
		raw, err = json.Marshal(data)
		if err != nil {
			return contracts.Response{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := local.Call(ctx, r.workspace, contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: r.workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: operation, ItemID: item, Data: raw})
	if err == nil && (!contracts.ValidID(response.SessionID) || r.session != "" && r.session != response.SessionID) {
		err = fmt.Errorf("runtime session changed after GUI Exit")
	}
	if err == nil && response.Error != nil {
		err = response.Error
	}
	if err != nil {
		err = fmt.Errorf("%s: %w", operation, err)
	}
	return response, err
}
func (r *exitRuntime) result(operation, item string, data any, target any) error {
	response, err := r.call(operation, item, data)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(response.Result)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
func (r *exitRuntime) waitWork(id string) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		var work struct {
			State string "json:\"state\""
		}
		if err := r.result("work.show", id, nil, &work); err != nil {
			return err
		}
		if work.State == "succeeded" {
			return nil
		}
		if work.State == "failed" || work.State == "cancelled" {
			return fmt.Errorf("qualification work %s", work.State)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("qualification work timed out")
}
func (r *exitRuntime) assertHeld(parent, acquisition string) error {
	for id, state := range map[string]string{parent: "pending", acquisition: "running"} {
		var work struct {
			State string "json:\"state\""
		}
		if err := r.result("work.show", id, nil, &work); err != nil {
			return err
		}
		if work.State != state {
			return fmt.Errorf("held work state differs: %s", work.State)
		}
	}
	return nil
}

// The outer fixture owns a disposable workspace, a loopback byte barrier and
// child GUI processes. It joins its runtime only after the durable job finishes.
func runExitQualification() error { return runExitQualificationWith(runExitGUIChild) }

func runExitQualificationWith(runChild func(exitFixture, string) (exitObservation, error)) (err error) {
	directory, err := os.MkdirTemp("", "insonic-exit-*")
	if err != nil {
		return err
	}
	defer func() {
		if cleanup := os.RemoveAll(directory); cleanup != nil && err == nil {
			err = cleanup
		}
	}()
	w, err := workspace.Init(directory, "Native Exit qualification")
	if err != nil {
		return err
	}
	if err = desktop.ConfigureQualificationTools(w); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready, done := make(chan struct{}), make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready, Idle: 90 * time.Second}) }()
	select {
	case <-ready:
	case err = <-done:
		cancel()
		if err == nil {
			err = fmt.Errorf("runtime exited before readiness")
		}
		return err
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		return fmt.Errorf("runtime readiness timed out")
	}
	defer func() {
		cancel()
		if cleanup := <-done; cleanup != nil && err == nil {
			err = cleanup
		}
	}()
	r := &exitRuntime{workspace: w}
	response, err := r.call("workspace.show", "", nil)
	if err != nil {
		return err
	}
	r.session = response.SessionID
	fixtureRoot := os.Getenv("INSONIC_DESKTOP_FIXTURE_DIRECTORY")
	if fixtureRoot == "" {
		fixtureRoot = filepath.Join("tests", "fixtures", "media")
	}
	audio, err := filepath.Abs(filepath.Join(fixtureRoot, "speech.flac"))
	if err != nil {
		return err
	}
	var admission struct {
		ID string "json:\"work_id\""
	}
	if err = r.result("media.import", "", library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Items: []library.Item{{Source: audio, Title: "Exit playback", NewEntry: true}}}, &admission); err != nil {
		return err
	}
	if err = r.waitWork(admission.ID); err != nil {
		return err
	}
	var imported struct {
		Items []library.ItemResult "json:\"items\""
	}
	if err = r.result("work.results", admission.ID, nil, &imported); err != nil {
		return err
	}
	if len(imported.Items) != 1 || !contracts.ValidID(imported.Items[0].MediaID) {
		return fmt.Errorf("Exit media fixture unavailable")
	}
	var entry library.EntryView
	if err = r.result("media.show", imported.Items[0].MediaID, nil, &entry); err != nil {
		return err
	}
	if entry.Revision < 1 {
		return fmt.Errorf("media revision missing")
	}
	released, entered := make(chan struct{}), make(chan struct{})
	var releaseOnce, enterOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(released) }) }
	payload := []byte("synthetic Exit acquisition bytes\n")
	digest := sha256.Sum256(payload)
	source := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/held" {
			http.NotFound(out, request)
			return
		}
		enterOnce.Do(func() { close(entered) })
		select {
		case <-released:
		case <-request.Context().Done():
			return
		}
		out.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		out.Write(payload)
	}))
	defer source.Close()
	defer release()
	manifest := models.Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: "native-exit-fixture", ModelVersion: "1", Revision: "synthetic-1", License: "fixture", Capabilities: []string{"transcription"}, Files: []models.File{{Role: "weights", SHA256: hex.EncodeToString(digest[:]), Size: int64(len(payload)), URL: source.URL + "/held", LocalHTTP: true}}}
	var acquired struct {
		ID           string   "json:\"work_id\""
		Acquisitions []string "json:\"acquisition_ids\""
	}
	if err = r.result("models.acquire", "", models.Request{Manifest: manifest}, &acquired); err != nil {
		return err
	}
	if len(acquired.Acquisitions) != 1 {
		return fmt.Errorf("acquisition identity missing")
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("acquisition did not reach barrier")
	}
	if err = r.assertHeld(acquired.ID, acquired.Acquisitions[0]); err != nil {
		return err
	}
	cases := []exitFixture{{Action: "menu"}, {Action: "menu", Workspace: directory, MediaID: imported.Items[0].MediaID, Revision: entry.Revision}, {Action: "close", Workspace: directory, MediaID: imported.Items[0].MediaID, Revision: entry.Revision}}
	if runtime.GOOS == "windows" {
		cases = []exitFixture{{Action: "menu"}, {Action: "close"}, {Action: "alt-f4"}, {Action: "menu", Workspace: directory, MediaID: imported.Items[0].MediaID, Revision: entry.Revision}, {Action: "close", Workspace: directory, MediaID: imported.Items[0].MediaID, Revision: entry.Revision}, {Action: "alt-f4", Workspace: directory, MediaID: imported.Items[0].MediaID, Revision: entry.Revision}}
	}
	for caseIndex, fixture := range cases {
		// Each case begins with a cold owned profile. Its second launch reuses that
		// exact profile only after the first launch's descendants exit normally.
		caseDirectory := filepath.Join(directory, fmt.Sprintf("native-case-%d", caseIndex))
		if err = os.Mkdir(caseDirectory, 0700); err != nil {
			return err
		}
		config := filepath.Join(caseDirectory, "native-exit.json")
		raw, _ := json.Marshal(fixture)
		if err = os.WriteFile(config, raw, 0600); err != nil {
			return err
		}
		repeats := 1
		if runtime.GOOS == "windows" {
			repeats = 2
		}
		for launch := 0; launch < repeats; launch++ {
			temperature := "cold"
			if launch > 0 {
				temperature = "warm"
			}
			fmt.Fprintf(os.Stderr, "native-exit-case action=%s workspace=%t profile=%s\n", fixture.Action, fixture.Workspace != "", temperature)
			observation, childErr := runChild(fixture, config)
			if childErr != nil {
				return childErr
			}
			if observation.NativeExit != "passed" || observation.Action != fixture.Action {
				return fmt.Errorf("native action receipt invalid")
			}
			if fixture.Workspace != "" {
				if observation.Playback != "released" || !contracts.ValidID(observation.PlaybackID) {
					return fmt.Errorf("playback cleanup receipt invalid")
				}
				checked, checkErr := r.call("media.playback-check", fixture.MediaID, map[string]any{"playback_id": observation.PlaybackID})
				if checkErr == nil || checked.Error == nil || checked.Error.Code != "not_found" || checked.SessionID != r.session {
					return fmt.Errorf("runtime playback handle survived GUI shutdown")
				}
			}
			if err = r.assertHeld(acquired.ID, acquired.Acquisitions[0]); err != nil {
				return err
			}
		}
	}
	release()
	if err = r.waitWork(acquired.Acquisitions[0]); err != nil {
		return err
	}
	if err = r.waitWork(acquired.ID); err != nil {
		return err
	}
	return nil
}
func qualifyNativeExit() int {
	if err := runExitQualification(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	proof := map[string]any{"schema_version": contracts.Version, "native_exit": "passed", "no_workspace_exit": "passed", "workspace_exit": "passed", "ordinary_close": "passed", "playback_cleanup": "passed", "runtime_work_survival": "passed"}
	if runtime.GOOS == "windows" {
		proof["owned_browser_release"] = "passed"
	}
	json.NewEncoder(os.Stdout).Encode(proof)
	return 0
}

type nativeExitFailure struct {
	Kind        string                    `json:"kind"`
	Stage       string                    `json:"stage"`
	Action      string                    `json:"action"`
	Stdout      string                    `json:"stdout"`
	Stderr      string                    `json:"stderr"`
	ExitCode    int                       `json:"exit_code"`
	Truncated   bool                      `json:"truncated"`
	TimedOut    bool                      `json:"timed_out"`
	Cancelled   bool                      `json:"cancelled"`
	Descendants *process.DescendantReport `json:"owned_descendants,omitempty"`
}

var nativeExitFailureSequence atomic.Uint32

func retainNativeExitEvidence(raw []byte) {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("INSONIC_PROCESS_STARTUP_DIAGNOSTICS") != "1" {
		return
	}
	root := os.Getenv("GITHUB_WORKSPACE")
	if !filepath.IsAbs(root) {
		return
	}
	sequence := nativeExitFailureSequence.Add(1)
	if sequence > 16 {
		return
	}
	directory := filepath.Join(root, "build", "native")
	if os.MkdirAll(directory, 0700) != nil {
		return
	}
	name := filepath.Join(directory, fmt.Sprintf("native-exit-child-%d-%d-receipt.json", os.Getpid(), sequence))
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	file.Write(append(raw, '\n'))
}

func interpretExitChild(fixture exitFixture, report process.CaptureResult, captureErr, contextErr error, diagnostic io.Writer) (exitObservation, error) {
	var observation exitObservation
	if captureErr == nil && contextErr == nil && report.ExitCode == 0 && !report.Truncated && json.Unmarshal(report.Stdout, &observation) == nil && observation.NativeExit == "passed" && observation.Action == fixture.Action {
		return observation, nil
	}
	stage, last := "native-exit", -1
	for _, candidate := range []string{"controller-startup", "dom-ready", "native-action", "menu-open", "menu-selected", "menu-request", "shutdown", "watchdog-timeout"} {
		if index := strings.LastIndex(string(report.Stderr), "native-exit-stage="+candidate); index > last {
			stage, last = candidate, index
		}
	}
	const limit = 65536
	truncated := report.Truncated || len(report.Stdout) > limit || len(report.Stderr) > limit
	evidence := nativeExitFailure{Kind: "native-exit-child-failure", Stage: stage, Action: fixture.Action, Stdout: string(report.Stdout[:min(len(report.Stdout), limit)]), Stderr: string(report.Stderr[:min(len(report.Stderr), limit)]), ExitCode: report.ExitCode, Truncated: truncated, TimedOut: contextErr == context.DeadlineExceeded || strings.Contains(string(report.Stderr), "native-exit-stage=watchdog-timeout"), Cancelled: contextErr == context.Canceled, Descendants: report.Descendants}
	raw, _ := json.Marshal(evidence)
	retainNativeExitEvidence(raw)
	fmt.Fprintln(diagnostic, string(raw))
	if captureErr == nil {
		captureErr = fmt.Errorf("native action receipt invalid")
	}
	return exitObservation{}, fmt.Errorf("native %s child failed: %w", fixture.Action, captureErr)
}

func runExitGUIChild(fixture exitFixture, config string) (exitObservation, error) {
	executable, err := os.Executable()
	if err != nil {
		return exitObservation{}, err
	}
	ctx, stop := context.WithTimeout(context.Background(), 18*time.Second)
	defer stop()
	report, err := process.Capture(ctx, process.Spec{Executable: executable, Args: []string{"--exit-qualification-child", config}, MaxOutput: 64 << 10, WaitForDescendants: runtime.GOOS == "windows"})
	if report.Descendants != nil {
		lifecycle, _ := json.Marshal(map[string]any{"kind": "native-exit-owned-lifecycle", "action": fixture.Action, "descendants": report.Descendants})
		retainNativeExitEvidence(lifecycle)
		fmt.Fprintln(os.Stderr, string(lifecycle))
	}
	return interpretExitChild(fixture, report, err, ctx.Err(), os.Stderr)
}
