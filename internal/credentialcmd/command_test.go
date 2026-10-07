// SPDX-License-Identifier: Apache-2.0
package credentialcmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

func fixture(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, e := workspace.Init(t.TempDir(), "Credential command fixture")
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func TestVaultCatalogIndependentCommands(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	id := contracts.ID()
	// Deliberately poison catalog routing: credential bootstrap must not open it.
	w.Config.Profiles.Catalog.Adapter = "unreachable-catalog"
	if _, e := Execute(ctx, w, []string{"select", "vault"}, nil); e != nil {
		t.Fatal(e)
	}
	payload := `{"value":{"username":"fixture","password":"hidden-fixture"},"passphrase":"fixture-passphrase"}`
	result, e := Execute(ctx, w, []string{"add", id}, strings.NewReader(payload))
	if e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "hidden-fixture") || strings.Contains(string(data), "fixture-passphrase") {
		t.Fatal("disclosed input")
	}
	if _, e = Execute(ctx, w, []string{"add", id}, strings.NewReader(payload)); e == nil {
		t.Fatal("implicit replacement")
	}
	input, e := ReadInput(strings.NewReader(`{"passphrase":"fixture-passphrase"}`))
	if e != nil {
		t.Fatal(e)
	}
	defer input.Close()
	manager, e := Provider(w, input)
	if e != nil {
		t.Fatal(e)
	}
	value, e := manager.Resolve(ctx, id)
	manager.Close()
	if e != nil || !strings.Contains(string(value), "hidden-fixture") {
		t.Fatal("configured credential unavailable", e)
	}
	if _, e = Execute(ctx, w, []string{"replace", id}, strings.NewReader(`{"value":"replacement","passphrase":"fixture-passphrase"}`)); e != nil {
		t.Fatal(e)
	}
	if _, e = Execute(ctx, w, []string{"delete", id}, strings.NewReader(`{"passphrase":"fixture-passphrase"}`)); e != nil {
		t.Fatal(e)
	}
}
func TestInputBoundsShapeAndDisclosure(t *testing.T) {
	for _, payload := range []string{`{"value":"private","value":"different"}`, `{"value":1}`, `{"passphrase":"secret","unexpected":true}`, `{"value":"` + strings.Repeat("x", secrets.MaxValueBytes+1) + `"}`, `[]`, `{} {}`} {
		input, e := ReadInput(strings.NewReader(payload))
		if e == nil {
			input.Close()
			t.Fatal("bad input admitted")
		}
		if strings.Contains(e.Error(), "private") || strings.Contains(e.Error(), "secret") {
			t.Fatal("input disclosure")
		}
	}
	input, e := ReadInput(strings.NewReader(`{"value":"bearer-value","passphrase":"passphrase"}`))
	if e != nil {
		t.Fatal(e)
	}
	data, e := json.Marshal(input)
	if e != nil || string(data) != "{}" {
		t.Fatal("transient input serialized")
	}
	value, pass := input.Value, input.Passphrase
	input.Close()
	if strings.Contains(string(value), "bearer-value") || strings.Contains(string(pass), "passphrase") {
		t.Fatal("input not cleared")
	}
}
func TestSessionProviderExplicitSelectionAndProcessLifetime(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	id := contracts.ID()
	t.Setenv("INSONIC_SESSION_CREDENTIALS", `{"`+id+`":{"password":"session-value"}}`)
	native, e := Provider(w, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = native.Resolve(ctx, id); e == nil {
		t.Fatal("ambient session fallback")
	}
	native.Close()
	if e = secrets.Select(w, "session"); e != nil {
		t.Fatal(e)
	}
	manager, e := Provider(w, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer manager.Close()
	if value, e := manager.Resolve(ctx, id); e != nil || !strings.Contains(string(value), "session-value") {
		t.Fatal(e)
	}
	if _, e = Execute(ctx, w, []string{"add", contracts.ID()}, strings.NewReader(`{"value":"lost-on-exit"}`)); e == nil {
		t.Fatal("ephemeral command falsely succeeded")
	}
	input, _ := ReadInput(strings.NewReader(`{"value":"live-value"}`))
	defer input.Close()
	newID := contracts.ID()
	if _, e = ExecuteWithManager(ctx, manager, []string{"add", newID}, input); e != nil {
		t.Fatal(e)
	}
	if value, e := manager.Resolve(ctx, newID); e != nil || string(value) != "live-value" {
		t.Fatal(e)
	}
}

func TestProposedBackendFailureDoesNotPersistSelection(t *testing.T) {
	w := fixture(t)
	t.Setenv("INSONIC_SESSION_CREDENTIALS", `{"not-a-uuid":"sensitive-input"}`)
	if manager, e := ProviderMode(w, "session", nil); e == nil {
		manager.Close()
		t.Fatal("malformed session mapping admitted")
	}
	if mode, e := secrets.ReadSelection(w); e != nil || mode != "native" {
		t.Fatal("failed proposal changed selection", mode, e)
	}
	if _, e := Execute(context.Background(), w, []string{"select", "session"}, nil); e == nil {
		t.Fatal("command selected malformed session mapping")
	}
	if mode, e := secrets.ReadSelection(w); e != nil || mode != "native" {
		t.Fatal("failed command changed selection", mode, e)
	}
	if manager, e := ProviderMode(w, "plaintext", nil); e == nil {
		manager.Close()
		t.Fatal("unsupported proposal admitted")
	}
}
