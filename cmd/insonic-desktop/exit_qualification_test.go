//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"os"
	"path/filepath"
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
	err = runExitQualificationWith(func(fixture exitFixture, _ string) (exitObservation, error) {
		calls++
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
	if calls < 3 {
		t.Fatal("fixture cases were skipped")
	}
}
