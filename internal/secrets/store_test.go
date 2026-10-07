// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func fixture(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, e := workspace.Init(t.TempDir(), "Secrets fixture")
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func TestSessionLifecycleIsolationAndStatus(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	id := contracts.ID()
	secret := []byte(`{"password":"fixture-hidden-value"}`)
	m, e := Open(w, Options{Mode: "session"})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Add(ctx, id, secret); e != nil {
		t.Fatal(e)
	}
	secret[0] = '!'
	got, e := m.Resolve(ctx, id)
	if e != nil || got[0] != '{' {
		t.Fatal("input aliased", e)
	}
	got[0] = '!'
	again, _ := m.Resolve(ctx, id)
	if again[0] != '{' {
		t.Fatal("output aliased")
	}
	status, e := m.Inspect(ctx, id)
	encoded, _ := json.Marshal(status)
	if e != nil || status.State != "configured" || strings.Contains(string(encoded), "hidden-value") {
		t.Fatal(status, e)
	}
	if e = m.Add(ctx, id, []byte("new")); e == nil {
		t.Fatal("implicit replacement admitted")
	}
	if e = m.Replace(ctx, id, []byte("replaced")); e != nil {
		t.Fatal(e)
	}
	other, e := Open(fixture(t), Options{Mode: "session"})
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	if _, e = other.Resolve(ctx, id); e == nil {
		t.Fatal("cross workspace resolution")
	}
	if e = m.Delete(ctx, id); e != nil {
		t.Fatal(e)
	}
	if s, _ := m.Status(ctx, id); s != "missing" {
		t.Fatal(s)
	}
	m.Close()
	if _, e = m.Resolve(ctx, id); e == nil {
		t.Fatal("closed session retained")
	}
	restarted, _ := Open(w, Options{Mode: "session"})
	defer restarted.Close()
	if s, _ := restarted.Status(ctx, id); s != "missing" {
		t.Fatal(s)
	}
}
func TestVaultRestartTamperAndIsolation(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	id := contracts.ID()
	value := []byte("fixture-vault-secret")
	opts := Options{Mode: "vault", Passphrase: []byte("fixture-passphrase")}
	m, e := Open(w, opts)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Add(ctx, id, value); e != nil {
		t.Fatal(e)
	}
	m.Close()
	file := filepath.Join(w.Control, "secrets", "vault.json")
	data, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(data), string(value)) || strings.Contains(string(data), string(opts.Passphrase)) {
		t.Fatal("plaintext persistence")
	}
	m, e = Open(w, opts)
	if e != nil {
		t.Fatal(e)
	}
	got, e := m.Resolve(ctx, id)
	if e != nil || string(got) != string(value) {
		t.Fatal("restart", e)
	}
	if e = m.Replace(ctx, id, []byte("replacement")); e != nil {
		t.Fatal(e)
	}
	m.Close()
	wrong, e := Open(w, Options{Mode: "vault", Passphrase: []byte("incorrect")})
	if e == nil {
		wrong.Close()
		t.Fatal("wrong passphrase admitted")
	}
	if strings.Contains(e.Error(), "incorrect") {
		t.Fatal("passphrase disclosed")
	}
	locked, e := Open(w, Options{Mode: "vault"})
	if e != nil {
		t.Fatal(e)
	}
	defer locked.Close()
	if state, _ := locked.Status(ctx, id); state != "rejected" {
		t.Fatal(state)
	}
	data, _ = os.ReadFile(file)
	other := fixture(t)
	if e = os.MkdirAll(filepath.Join(other.Control, "secrets"), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(other.Control, "secrets", "vault.json"), data, 0600); e != nil {
		t.Fatal(e)
	}
	if copied, e := Open(other, opts); e == nil {
		copied.Close()
		t.Fatal("workspace-bound ciphertext accepted elsewhere")
	}
	data[len(data)/2] ^= 1
	if e = os.WriteFile(file, data, 0600); e != nil {
		t.Fatal(e)
	}
	if corrupt, e := Open(w, opts); e == nil {
		corrupt.Close()
		t.Fatal("tamper admitted")
	}
}
func TestVaultConcurrentOwnersAndExplicitReplacement(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	opts := Options{Mode: "vault", Passphrase: []byte("concurrent-passphrase")}
	first, e := Open(w, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer first.Close()
	second, e := Open(w, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer second.Close()
	a, b := contracts.ID(), contracts.ID()
	if e = first.Add(ctx, a, []byte("first")); e != nil {
		t.Fatal(e)
	}
	if e = second.Add(ctx, b, []byte("second")); e != nil {
		t.Fatal(e)
	}
	if got, e := first.Resolve(ctx, b); e != nil || string(got) != "second" {
		t.Fatal("lost update", e)
	}
	if e = second.Replace(ctx, contracts.ID(), []byte("missing")); e == nil {
		t.Fatal("replace created new credential")
	}
	if e = first.Delete(ctx, a); e != nil {
		t.Fatal(e)
	}
	if state, _ := second.Status(ctx, a); state != "missing" {
		t.Fatal(state)
	}
}
func TestInputValidationAndNoFallback(t *testing.T) {
	w := fixture(t)
	if m, e := Open(w, Options{Mode: "plaintext"}); e == nil {
		m.Close()
		t.Fatal("unknown mode accepted")
	}
	m, e := Open(w, Options{Mode: "session"})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	for _, v := range [][]byte{nil, make([]byte, MaxValueBytes+1)} {
		if e = m.Add(context.Background(), contracts.ID(), v); e == nil {
			t.Fatal("value bound")
		}
	}
	if e = m.Add(context.Background(), "arbitrary-secret-id", []byte("x")); e == nil {
		t.Fatal("nonopaque ID admitted")
	}
}

func TestVaultRejectsUntrustedAlgorithmsAndAuthenticatesHeader(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	opts := Options{Mode: "vault", Passphrase: []byte("format-test-passphrase")}
	m, e := Open(w, opts)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Add(ctx, contracts.ID(), []byte("format-test-secret")); e != nil {
		t.Fatal(e)
	}
	m.Close()
	file := filepath.Join(w.Control, "secrets", "vault.json")
	raw, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	var original vaultEnvelope
	if e = json.Unmarshal(raw, &original); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*vaultEnvelope){
		func(v *vaultEnvelope) { v.MemoryKiB = 1 << 30 },
		func(v *vaultEnvelope) { v.Iterations = 1 },
		func(v *vaultEnvelope) { v.KDF = "plain" },
		func(v *vaultEnvelope) { v.IDs = []string{contracts.ID()} },
		func(v *vaultEnvelope) { v.Ciphertext = append([]byte(nil), v.Ciphertext...); v.Ciphertext[0] ^= 1 },
	} {
		v := original
		mutate(&v)
		corrupt, _ := json.Marshal(v)
		if e = os.WriteFile(file, corrupt, 0600); e != nil {
			t.Fatal(e)
		}
		if m, e = Open(w, opts); e == nil {
			m.Close()
			t.Fatal("unsupported or unauthenticated envelope admitted")
		}
	}
}

func TestProtectedFilesRejectDuplicateKeysAndSymlinks(t *testing.T) {
	var value any
	for _, raw := range []string{`{"mode":"native","mode":"session"}`, `{"a":{"b":1,"b":2}}`, `[] []`} {
		if strict([]byte(raw), &value) == nil {
			t.Fatal("ambiguous JSON accepted")
		}
	}
	w := fixture(t)
	root, _, e := privateRoot(w)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	outside := filepath.Join(t.TempDir(), "external.json")
	if e = os.WriteFile(outside, []byte(`{"external":true}`), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink(outside, filepath.Join(w.Control, "secrets", "selection.json")); e != nil {
		t.Skip("symlink creation unavailable")
	}
	if _, e = readFile(root, "selection.json"); e == nil {
		t.Fatal("secret control symlink followed")
	}
}
