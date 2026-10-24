// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	qt "github.com/shruggietech/insonic/internal/qualificationtiming"
	"os"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

func credentialInput(t *testing.T, raw string) {
	t.Helper()
	read, write, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = write.Write([]byte(raw)); e != nil {
		t.Fatal(e)
	}
	write.Close()
	previous := os.Stdin
	os.Stdin = read
	t.Cleanup(func() { os.Stdin = previous; read.Close() })
}

func TestCredentialColdBootstrapRejectsWrongModeOrMissingInput(t *testing.T) {
	for _, fixture := range []struct{ mode, action, input string }{
		{"native", "unlock", `{"passphrase":"fixture-passphrase"}`},
		{"native", "load", `{}`},
		{"session", "unlock", `{"session":{}}`},
		{"session", "load", `{}`},
		{"vault", "unlock", `{}`},
		{"vault", "load", `{}`},
	} {
		t.Run(fixture.mode+"-"+fixture.action, func(t *testing.T) {
			w, e := workspace.Init(t.TempDir(), "Cold credential fixture")
			if e != nil {
				t.Fatal(e)
			}
			if e = secrets.Select(w, fixture.mode); e != nil {
				t.Fatal(e)
			}
			credentialInput(t, fixture.input)
			if code := credentialCommand(context.Background(), w, []string{fixture.action}, true); code != 2 {
				t.Fatal("invalid bootstrap was not rejected", code)
			}
			ctx, cancel := qt.Context(context.Background(), "readiness_seconds")
			defer cancel()
			_, e = local.Call(ctx, w, contracts.Request{Operation: "workspace.show"})
			if e == nil {
				t.Fatal("invalid input started a runtime owner")
			}
		})
	}
}

func TestCredentialVaultUnlockRetainsUnlockedRuntimeManager(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "Vault runtime unlock fixture")
	if e != nil {
		t.Fatal(e)
	}
	if e = secrets.Select(w, "vault"); e != nil {
		t.Fatal(e)
	}
	id := contracts.ID()
	manager, e := secrets.Open(w, secrets.Options{Mode: "vault", Passphrase: []byte("fixture-passphrase")})
	if e != nil {
		t.Fatal(e)
	}
	if e = manager.Add(context.Background(), id, []byte("fixture-value")); e != nil {
		t.Fatal(e)
	}
	manager.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- local.Serve(ctx, w, local.Options{Ready: ready, Idle: qt.Duration("fixture_idle_seconds")})
	}()
	select {
	case <-ready:
	case e = <-done:
		cancel()
		t.Fatal(e)
	case <-time.After(qt.Duration("readiness_seconds")):
		cancel()
		t.Fatal("runtime startup timeout")
	}
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	credentialInput(t, `{"passphrase":"fixture-passphrase"}`)
	if code := credentialCommand(context.Background(), w, []string{"unlock"}, true); code != 0 {
		t.Fatal("valid vault unlock failed", code)
	}
	data, _ := json.Marshal(map[string]any{"arguments": []string{"status", id}})
	out, e := local.Call(context.Background(), w, contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "credentials.status", Data: data})
	if e != nil || out.Error != nil {
		t.Fatal(out.Error, e)
	}
	raw, _ := json.Marshal(out.Result)
	var status secrets.CredentialStatus
	if json.Unmarshal(raw, &status) != nil || status.State != "configured" {
		t.Fatal("unlock was not retained", string(raw))
	}
	// Re-selecting the live vault replaces its manager in the locked state.
	if code := credentialCommand(context.Background(), w, []string{"select", "vault"}, true); code != 0 {
		t.Fatal(code)
	}
	credentialInput(t, `{"passphrase":"fixture-passphrase"}`)
	if code := credentialCommand(context.Background(), w, []string{"delete", id}, true); code != 0 {
		t.Fatal("live locked vault delete failed", code)
	}
	check, e := secrets.Open(w, secrets.Options{Mode: "vault", Passphrase: []byte("fixture-passphrase")})
	if e != nil {
		t.Fatal(e)
	}
	defer check.Close()
	if state, _ := check.Status(context.Background(), id); state == "configured" {
		t.Fatal("deleted entry retained")
	}
}
