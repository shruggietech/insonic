// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
	"time"
)

func TestConfigurationBridgeUsesLiveSharedOwner(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Configured bridge fixture")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready}) }()
	select {
	case <-ready:
	case err := <-done:
		cancel()
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("owner startup timeout")
	}
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	b := &Bridge{Workspace: w}
	if response := b.Credential([]string{"select", "session"}, ""); response.Error != nil {
		t.Fatal(response.Error)
	}
	call := func(operation, item string, data any) contracts.Response {
		t.Helper()
		var raw json.RawMessage
		if data != nil {
			raw, _ = json.Marshal(data)
		}
		return b.Operate(contracts.Request{Operation: operation, ItemID: item, Data: raw})
	}
	speakerID := contracts.ID()
	input := map[string]any{"expected_revision": 0, "speaker": map[string]any{"id": speakerID, "name": "Exact Fixture Name"}, "aliases": []any{}}
	created := call("speakers.set", speakerID, input)
	if created.Error != nil {
		t.Fatal(created.Error)
	}
	stale := call("speakers.set", speakerID, input)
	if stale.Error == nil || stale.Error.Code != "conflict" {
		t.Fatal("stale revision replaced identity", stale)
	}
	termID := contracts.ID()
	term := call("terms.set", termID, map[string]any{"expected_revision": 0, "term": map[string]any{"id": termID, "canonical": "Fixture Vocabulary"}})
	if term.Error != nil {
		t.Fatal(term.Error)
	}
	compiled := call("terms.compile", "", map[string]any{"speaker_ids": []string{speakerID}, "max_hint_bytes": 200})
	if compiled.Error != nil {
		t.Fatal(compiled.Error)
	}
	raw, _ := json.Marshal(compiled.Result)
	var hints struct {
		Hints []string `json:"hints"`
	}
	if json.Unmarshal(raw, &hints) != nil {
		t.Fatal("invalid compilation result")
	}
	seen := map[string]bool{}
	for _, hint := range hints.Hints {
		seen[hint] = true
	}
	if !seen["Exact Fixture Name"] || !seen["Fixture Vocabulary"] {
		t.Fatal("bridge and runtime context differ", string(raw))
	}
	for _, operation := range []string{"pipelines.list", "speakers.list", "terms.list", "terms.compile"} {
		if response := call(operation, "", nil); response.Error != nil {
			t.Fatal(operation, response.Error)
		}
	}
	if response := call("speakers.aliases", speakerID, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	if response := call("speakers.select", speakerID, map[string]any{"limit": 101}); response.Error == nil || response.Error.Code != "invalid_request" {
		t.Fatal("bridge admitted invalid selection bound", response)
	}
}
