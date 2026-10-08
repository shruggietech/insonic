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

func TestDesktopOperationDeadlineAllowsRuntimeLongWork(t *testing.T) {
	for _, fixture := range []struct {
		operation string
		timeout   time.Duration
	}{
		{"media.refresh", 10 * time.Minute},
		{"models.materialize", 10 * time.Minute},
		{"artifacts.publish", 10 * time.Minute},
		{"pipelines.inspect", 10 * time.Minute},
		{"speakers.select", 10 * time.Minute},
		{"terms.compile", 10 * time.Minute},
		{"workspace.show", 10 * time.Second},
		{"work.results", 10 * time.Second},
	} {
		started := time.Now()
		ctx, cancel := operationContext(fixture.operation)
		deadline, ok := ctx.Deadline()
		remaining := deadline.Sub(started)
		if !ok || remaining < fixture.timeout-time.Second || remaining > fixture.timeout+time.Second {
			cancel()
			t.Fatal("desktop context caps runtime operation", fixture.operation, remaining)
		}
		cancel()
		select {
		case <-ctx.Done():
		default:
			t.Fatal("operation cancellation lost")
		}
	}
}

func TestGeneralOperationRejectsWrongAuthorityBeforeRuntime(t *testing.T) {
	if result := (&Bridge{}).Operate(contracts.Request{Operation: "media.list"}); result.Error == nil || result.Error.Code != "not_found" {
		t.Fatal(result)
	}
	w, e := workspace.Init(t.TempDir(), "Bridge domain fixture")
	if e != nil {
		t.Fatal(e)
	}
	result := (&Bridge{Workspace: w}).Operate(contracts.Request{Operation: "media.list", WorkspaceID: contracts.ID()})
	if result.Error == nil || result.Error.Code != "workspace_mismatch" {
		t.Fatal(result)
	}
	result = (&Bridge{Workspace: w}).Operate(contracts.Request{Operation: "workspace.show", Version: "unsupported"})
	if result.Error == nil || result.Error.Code != "incompatible_version" {
		t.Fatal(result)
	}
}

func TestCredentialBridgeReusesLiveOwnerAndIgnoresStatusInput(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "Live secret bridge fixture")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- local.Serve(ctx, w, local.Options{Ready: ready}) }()
	select {
	case <-ready:
	case e := <-done:
		cancel()
		t.Fatal("owner startup", e)
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("owner timeout")
	}
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	bridge := &Bridge{Workspace: w}
	if result := bridge.Credential([]string{"select", "session"}, "ignored malformed input"); result.Error != nil {
		t.Fatal(result)
	}
	id := contracts.ID()
	if result := bridge.Credential([]string{"add", id}, `{"value":"session-fixture"}`); result.Error != nil {
		t.Fatal(result)
	}
	if result := bridge.Credential([]string{"unlock"}, `{"session":{}}`); result.Error == nil || result.Error.Code != "invalid_request" {
		t.Fatal("vault unlock accepted session input", result)
	}
	result := bridge.Credential([]string{"status", id}, "ignored malformed input")
	if result.Error != nil {
		t.Fatal(result)
	}
	// The same live owner retains the session registration; a standalone manager
	// would lose it when the preceding command returned.
	raw, _ := json.Marshal(result.Result)
	var status struct {
		State string `json:"state"`
	}
	json.Unmarshal(raw, &status)
	if status.State != "configured" {
		t.Fatal(string(raw))
	}
	if result := bridge.Credential([]string{"replace", contracts.ID()}, `{"value":"new-value"}`); result.Error == nil || result.Error.Code != "not_found" {
		t.Fatal("live application error fell back", result)
	}
}
func TestCredentialBridgeBootstrapWithoutCatalog(t *testing.T) {
	if result := (&Bridge{}).Credential([]string{"select", "session"}, ""); result.Error == nil || result.Error.Code != "not_found" {
		t.Fatal(result)
	}
	w, e := workspace.Init(t.TempDir(), "Bridge secret fixture")
	if e != nil {
		t.Fatal(e)
	}
	w.Config.Profiles.Catalog.Adapter = "unavailable-catalog"
	result := (&Bridge{Workspace: w}).Credential([]string{"select", "vault"}, "")
	if result.Error != nil {
		t.Fatal(result)
	}
	result = (&Bridge{Workspace: w}).Credential([]string{"add", contracts.ID()}, `{"value":"fixture-only-secret","passphrase":"fixture-passphrase"}`)
	if result.Error != nil {
		t.Fatal(result)
	}
}
