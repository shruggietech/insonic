// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestBackupCLIIntentAndModeValidation(t *testing.T) {
	for _, action := range []string{"create", "show", "verify", "restore", "release"} {
		got, err := parseBackup([]string{action, "portable backup"})
		if err != nil || got.action != action || !filepath.IsAbs(got.directory) || got.mode != "self-contained" {
			t.Fatalf("%s: %#v %v", action, got, err)
		}
	}
	got, err := parseBackup([]string{"create", "bundle", "--mode", "reference-only"})
	if err != nil || got.mode != "reference-only" {
		t.Fatal(got, err)
	}
	id := contracts.ID()
	got, err = parseBackup([]string{"release", id})
	if err != nil || got.directory != id {
		t.Fatal("backup recovery identity became a path", got, err)
	}
	for _, args := range [][]string{{}, {"create"}, {"create", ""}, {"create", "--mode"}, {"create", "bundle", "--mode", "unknown"}, {"restore", "bundle", "--mode", "reference-only"}, {"delete", "bundle"}, {"release", "bundle", "extra"}} {
		if _, err := parseBackup(args); err == nil {
			t.Fatalf("invalid backup intent accepted: %v", args)
		}
	}
}

func TestBackupCLIMachineRoundTripAndNoOverwrite(t *testing.T) {
	ctx := context.Background()
	initialize := func(name string) *workspace.Workspace {
		t.Helper()
		w, err := workspace.Init(filepath.Join(t.TempDir(), name), name)
		if err != nil {
			t.Fatal(err)
		}
		if err = secrets.Select(w, "session"); err != nil {
			t.Fatal(err)
		}
		store, err := catalog.OpenWorkspace(ctx, w, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Close(); err != nil {
			t.Fatal(err)
		}
		return w
	}
	source, destination := initialize("source"), initialize("destination")
	invoke := func(w *workspace.Workspace, args ...string) (int, []byte) {
		t.Helper()
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		previous := os.Stdout
		os.Stdout = writer
		code := execute(append([]string{"--workspace", w.Root, "--json", "backup"}, args...))
		os.Stdout = previous
		writer.Close()
		raw, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		return code, raw
	}
	bundle := filepath.Join(t.TempDir(), "portable bundle")
	code, raw := invoke(source, "create", bundle)
	if code != 0 {
		t.Fatalf("create exit %d: %s", code, raw)
	}
	var envelope contracts.Response
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Kind != "runtime-response" || envelope.WorkspaceID != source.Config.WorkspaceID || envelope.Error != nil {
		t.Fatalf("machine response: %s %v", raw, err)
	}
	code, raw = invoke(destination, "restore", bundle)
	if code != 0 {
		t.Fatalf("restore exit %d: %s", code, raw)
	}
	restored, err := workspace.Open(destination.Root)
	if err != nil || restored.Config.WorkspaceID != source.Config.WorkspaceID {
		t.Fatal("restored identity", err)
	}
	if _, err := secrets.ReadSelection(restored); err != nil {
		t.Fatal("restored credential selection cannot reopen", err)
	}
	code, _ = invoke(source, "create", bundle)
	if code == 0 {
		t.Fatal("new backup identity overwrote existing bundle")
	}
}
