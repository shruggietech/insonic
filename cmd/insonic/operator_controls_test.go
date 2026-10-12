// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOperatorControlsAreGlobalAndValidated(t *testing.T) {
	got, err := parseInvocation([]string{"--debug", "work", "wait", "11111111-1111-4111-8111-111111111111", "--log-level=debug", "--log-file", "operator.jsonl"})
	if err != nil || !got.Debug || got.LogLevel != "debug" || got.LogFile != "operator.jsonl" || len(got.Args) != 3 {
		t.Fatal(got, err)
	}
	for _, args := range [][]string{{"version", "--quiet", "--verbose"}, {"version", "--quiet", "--debug"}, {"version", "--log-level=trace"}, {"version", "--log-file="}} {
		if _, err := parseInvocation(args); err == nil {
			t.Fatal("invalid control admitted", args)
		}
	}
}

func TestElectedLogFailurePrecedesInitialization(t *testing.T) {
	root := filepath.Join(t.TempDir(), "must remain absent")
	log := filepath.Join(t.TempDir(), "missing", "operator.jsonl")
	if code := execute([]string{"--json", "--log-file", log, "workspace", "init", root}); code != 4 {
		t.Fatal(code)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("log failure performed workspace initialization", err)
	}
}
