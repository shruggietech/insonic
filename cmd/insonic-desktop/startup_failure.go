//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shruggietech/insonic/internal/diagnostics"
)

type desktopStartupFailure struct {
	Stage string
	Err   error
}

func (failure *desktopStartupFailure) Error() string {
	labels := map[string]string{
		"workspace":          "Opening the selected workspace",
		"playback-transport": "Starting desktop playback transport",
		"desktop-window":     "Starting the desktop window",
		"desktop-close":      "Closing the desktop window",
	}
	return fmt.Sprintf("%s: %v", labels[failure.Stage], failure.Err)
}
func (failure *desktopStartupFailure) Unwrap() error { return failure.Err }

func startupFailure(stage string, cause error) error {
	if cause == nil {
		return nil
	}
	return &desktopStartupFailure{Stage: stage, Err: cause}
}

func startupErrorText(err error) string {
	text := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(err.Error(), "\uFFFD"))
	runes := []rune(text)
	if len(runes) > 2000 {
		text = string(runes[:2000]) + "..."
	}
	return text
}

func saveStartupEventAt(path string, event diagnostics.Event) (string, error) {
	log, err := diagnostics.Open(path, "error")
	if err != nil {
		return "", err
	}
	writeErr := log.Write("error", event)
	closeErr := log.Close()
	if err = errors.Join(writeErr, closeErr); err != nil {
		return "", err
	}
	return path, nil
}

func saveDesktopStartupEvent(event diagnostics.Event) (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	directory := filepath.Join(root, "insonic", "diagnostics")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	return saveStartupEventAt(filepath.Join(directory, "desktop-startup.jsonl"), event)
}

// Reporting may fail independently; its return remains the original cause.
// Tests inject both sinks, so no test opens a dialog or uses a real user profile.
func reportDesktopStartupFailure(primary error, interactive bool, out io.Writer,
	record func(diagnostics.Event) (string, error), present func(string) error) error {
	if primary == nil {
		return nil
	}
	stage := "desktop-window"
	var failure *desktopStartupFailure
	if errors.As(primary, &failure) {
		stage = failure.Stage
	}
	event := diagnostics.StartupEvent(stage, primary)
	feedback := "insonic could not start.\n\n" + startupErrorText(primary) +
		"\n\nCheck the indicated dependency or directory access, then launch insonic again."
	if record != nil {
		path, err := record(event)
		if err == nil && path != "" {
			feedback += "\n\nPrivate diagnostics saved: " + path
		} else {
			feedback += "\n\nPrivate diagnostics could not be saved. Check the diagnostic directory and its access permissions."
		}
	}
	fmt.Fprintln(out, feedback)
	if interactive {
		if present == nil || present(feedback) != nil {
			fmt.Fprintln(out, "Native failure feedback was unavailable; the original startup failure is retained above.")
		}
	}
	return primary
}
