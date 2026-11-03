// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
	"time"
)

func TestSpeakerDiscoveryBridgeUsesSharedValidation(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "Discovery bridge fixture")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready}) }()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(qualificationtiming.Duration("readiness_seconds")):
		t.Fatal("runtime readiness")
	}
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	b := &Bridge{Workspace: w}
	config := assistance.DefaultConfig()
	config.Enabled = true
	config.Endpoint = "http://127.0.0.1:1"
	config.Model = "fixture"
	for _, op := range []string{"speakers.discover", "speakers.hypotheses", "speakers.resolve", "speakers.apply", "speakers.retract", "speakers.run"} {
		var data json.RawMessage
		if op == "speakers.discover" {
			data, _ = json.Marshal(map[string]any{"configuration": config})
		}
		if op == "speakers.resolve" || op == "speakers.apply" {
			data = json.RawMessage(`{"expected_revision":1}`)
		}
		if op == "speakers.retract" {
			data, _ = json.Marshal(map[string]any{"expected_recording_revision": 1, "mappings": []any{map[string]any{"id": contracts.ID(), "revision": 1}}})
		}
		if op == "speakers.run" {
			data, _ = json.Marshal(map[string]any{"configuration": map[string]any{"provider": config, "apply": true, "enroll": false, "select_profile": false}})
		}
		response := b.Operate(contracts.Request{Operation: op, ItemID: contracts.ID(), Data: data})
		if response.Error == nil || response.Error.Code != "not_found" {
			t.Fatal(op, response)
		}
	}
	for _, op := range []string{"speakers.discover", "speakers.resolve", "speakers.apply", "speakers.retract", "speakers.run"} {
		response := b.Operate(contracts.Request{Operation: op, ItemID: contracts.ID(), Data: json.RawMessage(`{"unknown":true}`)})
		if response.Error == nil || response.Error.Code != "invalid_request" {
			t.Fatal(op, response)
		}
	}
}
