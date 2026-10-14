// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/firstsetup"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupInvocationAndRuntimePayload(t *testing.T) {
	for _, args := range [][]string{{"setup", "location"}, {"setup", "location", t.TempDir()}, {"setup", "draft", "show"}, {"setup", "provision"}, {"setup", "show"}, {"setup", "readiness"}, {"settings", "setup"}} {
		if _, e := parseInvocation(args); e != nil {
			t.Fatalf("%v: %v", args, e)
		}
	}
	f := filepath.Join(t.TempDir(), "state.json")
	state := firstsetup.DefaultState()
	for i := range state.Stages {
		state.Stages[i].Proof = strings.Repeat("a", 64)
	}
	valid, _ := json.Marshal(firstsetup.SaveInput{State: state})
	os.WriteFile(f, valid, 0600)
	op, item, raw, e := parseDesktopOperation([]string{"setup", "save", "--input", f})
	if e != nil || op != "setup.save" || item != "" || len(raw) == 0 {
		t.Fatalf("%s %v", op, e)
	}
	op, _, _, e = parseDesktopOperation([]string{"settings", "setup"})
	if e != nil || op != "setup.show" {
		t.Fatalf("settings alias: %s %v", op, e)
	}
}
func TestSetupDraftCLIRejectsUnknownAndSecretFields(t *testing.T) {
	file := filepath.Join(t.TempDir(), "draft.json")
	raw, _ := json.Marshal(map[string]any{"expected_revision": 0, "path": t.TempDir(), "plan": map[string]any{"display_name": "Fixture"}, "stage": "workspace", "status": "active", "password": "SECRET_SENTINEL"})
	os.WriteFile(file, raw, 0600)
	if code := setupBootstrapCommand([]string{"setup", "draft", "save", "--input", file}, contracts.ID(), true); code == 0 {
		t.Fatal("secret draft accepted")
	}
}
