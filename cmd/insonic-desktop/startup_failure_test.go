//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/diagnostics"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func TestReturnedStartupFailurePreservesCauseAndClosesOwnedTransport(t *testing.T) {
	cause := errors.New("injected window startup cause")
	closed, launched := 0, 0
	runner := desktopApplicationRunner{
		launch: func(*options.App) error {
			launched++
			if closed != 0 {
				t.Fatal("cleanup preceded launch completion")
			}
			return cause
		},
		playback:   func(*desktop.Bridge) (func(), error) { return func() { closed++ }, nil },
		closeError: func() error { return nil },
	}
	err := runDesktopApplicationWithRunner(&desktop.Bridge{}, nil, nil, true, "", runner)
	var failure *desktopStartupFailure
	if !errors.Is(err, cause) || !errors.As(err, &failure) || failure.Stage != "desktop-window" || launched != 1 || closed != 1 {
		t.Fatalf("original cause, stage or owned cleanup lost: %v %d %d", err, launched, closed)
	}
}

func TestStartupTransportAndCloseFailuresRemainDistinct(t *testing.T) {
	cause := errors.New("injected cause")
	launched := false
	runner := desktopApplicationRunner{
		launch:     func(*options.App) error { launched = true; return nil },
		playback:   func(*desktop.Bridge) (func(), error) { return nil, cause },
		closeError: func() error { return nil },
	}
	err := runDesktopApplicationWithRunner(&desktop.Bridge{}, nil, nil, true, "", runner)
	var failure *desktopStartupFailure
	if launched || !errors.Is(err, cause) || !errors.As(err, &failure) || failure.Stage != "playback-transport" {
		t.Fatal("transport failure changed", err)
	}
	runner.playback = func(*desktop.Bridge) (func(), error) { return func() {}, nil }
	runner.closeError = func() error { return cause }
	err = runDesktopApplicationWithRunner(&desktop.Bridge{}, nil, nil, true, "", runner)
	if !errors.Is(err, cause) || !errors.As(err, &failure) || failure.Stage != "desktop-close" {
		t.Fatal("close cause changed", err)
	}
}

func TestStartupFeedbackSurvivesDeniedDiagnosticsAndUnavailablePresentation(t *testing.T) {
	primary := startupFailure("desktop-window", fmt.Errorf("controller cause: %w", syscall.Errno(5)))
	for _, interactive := range []bool{false, true} {
		var out bytes.Buffer
		calls := 0
		got := reportDesktopStartupFailure(primary, interactive, &out,
			func(event diagnostics.Event) (string, error) {
				if event.FailureStage != "desktop-window" || event.SystemErrorCode == nil || *event.SystemErrorCode != 5 {
					t.Fatal(event)
				}
				return "", os.ErrPermission
			},
			func(message string) error {
				calls++
				if !strings.Contains(message, "controller cause") || !strings.Contains(message, "could not be saved") {
					t.Fatal(message)
				}
				return errors.New("presentation unavailable")
			})
		if got != primary || !errors.Is(got, syscall.Errno(5)) || !strings.Contains(out.String(), "controller cause") || calls != map[bool]int{false: 0, true: 1}[interactive] {
			t.Fatalf("secondary error replaced startup cause: %v %s", got, out.String())
		}
	}
}

func TestStartupPrivateEventOmitsArbitraryErrorTextAndPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup.jsonl")
	primary := startupFailure("desktop-window", fmt.Errorf("private-token C:/private/path provider-body: %w", syscall.Errno(5)))
	var out bytes.Buffer
	shown := ""
	got := reportDesktopStartupFailure(primary, true, &out,
		func(event diagnostics.Event) (string, error) { return saveStartupEventAt(path, event) },
		func(message string) error { shown = message; return nil })
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-token", "C:/private/path", "provider-body"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatal("private error data persisted", string(raw))
		}
	}
	if got != primary || !strings.Contains(string(raw), "desktop-window") || !strings.Contains(string(raw), `"system_error_code":5`) ||
		!strings.Contains(shown, path) || !strings.Contains(shown, "Private diagnostics saved") {
		t.Fatal("useful event or feedback lost", string(raw), shown)
	}
}

func TestSuccessfulStartupDoesNotReportOrWriteDiagnostics(t *testing.T) {
	var out bytes.Buffer
	calls := 0
	err := reportDesktopStartupFailure(nil, true, &out, func(diagnostics.Event) (string, error) { calls++; return "", nil }, func(string) error { calls++; return nil })
	if err != nil || calls != 0 || out.Len() != 0 {
		t.Fatal("success reported a startup failure")
	}
}

func TestStartupFeedbackBoundsTextAndRemovesNativeStringControls(t *testing.T) {
	cause := errors.New("cause\x00\n" + strings.Repeat("界", 2100))
	text := startupErrorText(cause)
	if strings.ContainsAny(text, "\x00\r\n") || len([]rune(text)) != 2003 || !strings.HasSuffix(text, "...") || !strings.HasPrefix(text, "cause  ") {
		t.Fatal("startup feedback lost its native string bound or control sanitization")
	}
}
