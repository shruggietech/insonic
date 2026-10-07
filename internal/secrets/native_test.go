// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

type fixtureNative struct {
	values      map[string][]byte
	gets        int
	unavailable bool
}

func (n *fixtureNative) available(context.Context) error {
	if n.unavailable {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n *fixtureNative) get(_ context.Context, id string) ([]byte, error) {
	n.gets++
	if value, ok := n.values[id]; ok {
		return append([]byte(nil), value...), nil
	}
	return nil, contracts.Fail("not_found")
}
func (n *fixtureNative) put(_ context.Context, id string, value []byte) error {
	n.values[id] = append([]byte(nil), value...)
	return nil
}
func (n *fixtureNative) remove(_ context.Context, id string) error { delete(n.values, id); return nil }
func TestNativeStatusNeverFetchesValuesAndScope(t *testing.T) {
	ctx := context.Background()
	w := fixture(t)
	id := contracts.ID()
	backend := &fixtureNative{values: map[string][]byte{}}
	m, e := Open(w, Options{Mode: "native"})
	if e != nil {
		t.Fatal(e)
	}
	m.native = backend
	if e = m.Add(ctx, id, []byte("private-value")); e != nil {
		t.Fatal(e)
	}
	backend.gets = 0
	for range 3 {
		if state, e := m.Status(ctx, id); e != nil || state != "configured" {
			t.Fatal(state, e)
		}
	}
	if backend.gets != 0 {
		t.Fatal("status retrieved saved value")
	}
	m.Close()
	restarted, e := Open(w, Options{Mode: "native"})
	if e != nil {
		t.Fatal(e)
	}
	defer restarted.Close()
	restarted.native = backend
	if state, _ := restarted.Status(ctx, id); state != "configured" {
		t.Fatal(state)
	}
	other, e := Open(fixture(t), Options{Mode: "native"})
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	other.native = backend
	if _, e = other.Resolve(ctx, id); e == nil {
		t.Fatal("index scope escaped")
	}
	if backend.gets != 0 {
		t.Fatal("other workspace accessed saved value")
	}
	if e = restarted.Replace(ctx, id, []byte("replaced")); e != nil {
		t.Fatal(e)
	}
	value, e := restarted.Resolve(ctx, id)
	if e != nil || string(value) != "replaced" {
		t.Fatal(e)
	}
	backend.unavailable = true
	if state, _ := restarted.Status(ctx, id); state != "rejected" {
		t.Fatal(state)
	}
	if e = restarted.Replace(ctx, id, []byte("must-not-fallback")); e == nil {
		t.Fatal("unavailable native accepted write")
	}
	backend.unavailable = false
	if e = restarted.Delete(ctx, id); e != nil {
		t.Fatal(e)
	}
	if state, _ := restarted.Status(ctx, id); state != "missing" {
		t.Fatal(state)
	}
}
func TestNativeInterruptedRegistrationRequiresExplicitRecovery(t *testing.T) {
	ctx := context.Background()
	id := contracts.ID()
	backend := &fixtureNative{values: map[string][]byte{id: []byte("orphaned-value")}}
	m, e := Open(fixture(t), Options{Mode: "native"})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	m.native = backend
	if e = m.Add(ctx, id, []byte("silent-overwrite")); e == nil || string(backend.values[id]) != "orphaned-value" {
		t.Fatal("orphan was implicitly replaced", e)
	}
	if e = m.Replace(ctx, id, []byte("explicit-recovery")); e != nil {
		t.Fatal(e)
	}
	if state, e := m.Status(ctx, id); e != nil || state != "configured" {
		t.Fatal(state, e)
	}
	if value, e := m.Resolve(ctx, id); e != nil || string(value) != "explicit-recovery" {
		t.Fatal("recovery failed", e)
	}
	orphan := contracts.ID()
	backend.values[orphan] = []byte("delete-orphan")
	if e = m.Delete(ctx, orphan); e != nil || backend.values[orphan] != nil {
		t.Fatal("explicit orphan deletion failed", e)
	}
	if e = m.Replace(ctx, contracts.ID(), []byte("new")); e == nil {
		t.Fatal("replace created missing native value")
	}
}
func TestSelectedBackendPersistsAndRejectsForeignSelection(t *testing.T) {
	w := fixture(t)
	if mode, e := ReadSelection(w); e != nil || mode != "native" {
		t.Fatal(mode, e)
	}
	for _, mode := range []string{"session", "vault", "native"} {
		if e := Select(w, mode); e != nil {
			t.Fatal(e)
		}
		got, e := ReadSelection(w)
		if e != nil || got != mode {
			t.Fatal(got, e)
		}
		m, e := Open(w, Options{})
		if e != nil {
			t.Fatal(e)
		}
		if m.mode != mode {
			t.Fatal(m.mode)
		}
		m.Close()
	}
	if e := Select(w, "plaintext"); e == nil {
		t.Fatal("plaintext selection")
	}
	foreign := fixture(t)
	raw, _ := json.Marshal(map[string]any{"version": 1, "workspace_id": foreign.Config.WorkspaceID, "mode": "session"})
	if e := os.WriteFile(w.Control+"/secrets/selection.json", raw, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := ReadSelection(w); e == nil {
		t.Fatal("foreign selection admitted")
	}
}
func TestNativePlatformLifecycle(t *testing.T) {
	if os.Getenv("INSONIC_NATIVE_SECRET_TEST") != "1" {
		t.Skip("native lifecycle executes under isolated platform qualification")
	}
	ctx := context.Background()
	w := fixture(t)
	id := contracts.ID()
	m, e := Open(w, Options{Mode: "native"})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Add(ctx, id, []byte("native-fixture-value")); e != nil {
		m.Close()
		t.Fatal(e)
	}
	defer func() {
		cleanup, e := Open(w, Options{Mode: "native"})
		if e == nil {
			cleanup.Delete(ctx, id)
			cleanup.Close()
		}
	}()
	if state, e := m.Status(ctx, id); e != nil || state != "configured" {
		t.Fatal(state, e)
	}
	m.Close()
	m, e = Open(w, Options{Mode: "native"})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	if value, e := m.Resolve(ctx, id); e != nil || string(value) != "native-fixture-value" {
		t.Fatal("native restart", e)
	}
	if e = m.Replace(ctx, id, []byte("native-replacement")); e != nil {
		t.Fatal(e)
	}
	if value, e := m.Resolve(ctx, id); e != nil || string(value) != "native-replacement" {
		t.Fatal("native replacement", e)
	}
	if e = m.Delete(ctx, id); e != nil {
		t.Fatal(e)
	}
	if _, e = m.Resolve(ctx, id); e == nil {
		t.Fatal("deleted native resolved")
	}
}
