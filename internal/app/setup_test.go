// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
)

func setupTestCall(t *testing.T, a *App, op, id string, data any) (map[string]any, error) {
	t.Helper()
	var raw json.RawMessage
	if data != nil {
		raw, _ = json.Marshal(data)
	}
	value, err := a.setupDispatch(context.Background(), contracts.Request{Operation: op, RequestID: id, Data: raw})
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(value)
	var out map[string]any
	if json.Unmarshal(encoded, &out) != nil {
		t.Fatal("setup result is not an object")
	}
	return out, nil
}

func setupTestState(t *testing.T, a *App) (map[string]any, int64) {
	t.Helper()
	out, err := setupTestCall(t, a, "setup.show", contracts.ID(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return out["state"].(map[string]any), int64(out["revision"].(float64))
}

func TestSetupCatalogReplayRestartConflictAndIsolation(t *testing.T) {
	a := configuredApp(t)
	state, revision := setupTestState(t, a)
	if revision != 0 || state["status"] != "active" {
		t.Fatal("missing journey did not offer an optional initial state", state, revision)
	}
	state["status"] = "deferred"
	id := contracts.ID()
	input := map[string]any{"expected_revision": revision, "state": state}
	first, err := setupTestCall(t, a, "setup.save", id, input)
	if err != nil {
		t.Fatal(err)
	}
	accepted := first["accepted_revision"].(float64)
	_, acceptedSettingRevision, err := a.Catalog.NamedSetting(context.Background(), setupSetting)
	if err != nil || accepted != float64(acceptedSettingRevision) {
		t.Fatal("accepted receipt did not identify the persisted setting revision", first, acceptedSettingRevision, err)
	}
	a.Close()
	a, err = New(a.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	replay, err := setupTestCall(t, a, "setup.save", id, input)
	if err != nil || replay["accepted_revision"] != accepted {
		t.Fatal("unknown save outcome did not reconcile after restart", replay, err)
	}
	newState, newerRevision := setupTestState(t, a)
	// Unrelated operational catalog writes advance the global revision, but
	// the journey still compares its own observed setting revision.
	if _, err = a.Catalog.PutNamedSetting(context.Background(), contracts.ID(), "unrelated-setting", 0, json.RawMessage(`{"enabled":true}`)); err != nil {
		t.Fatal(err)
	}
	newState["status"] = "active"
	if _, err = setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": newerRevision, "state": newState}); err != nil {
		t.Fatal(err)
	}
	replay, err = setupTestCall(t, a, "setup.save", id, input)
	if err != nil || replay["accepted_revision"] != accepted || replay["revision"].(float64) <= accepted || replay["state"].(map[string]any)["status"] != "active" {
		t.Fatal("accepted revision was confused with newer current state", replay, err)
	}
	if _, err = setupTestCall(t, a, "setup.save", contracts.ID(), input); err == nil {
		t.Fatal("stale journey overwrote newer accepted state")
	}
	input["state"] = newState
	if _, err = setupTestCall(t, a, "setup.save", id, input); err == nil {
		t.Fatal("changed intent reused an accepted request identity")
	}
	b := configuredApp(t)
	_, revision = setupTestState(t, b)
	if revision != 0 {
		t.Fatal("setup state leaked across workspaces", revision)
	}
}

func TestSetupConcurrentWritersRetainOneAcceptedChoice(t *testing.T) {
	a := configuredApp(t)
	state, revision := setupTestState(t, a)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, status := range []string{"active", "deferred"} {
		encoded, _ := json.Marshal(state)
		var draft map[string]any
		json.Unmarshal(encoded, &draft)
		draft["status"] = status
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": revision, "state": draft})
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	accepted, conflicts := 0, 0
	for err := range errors {
		if err == nil {
			accepted++
		} else if contracts.SafeError(err).Code == "conflict" {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatal("concurrent drafts did not preserve a single winner", accepted, conflicts)
	}
}

func TestSetupProofChangesAreObservationsNotConfigurationWrites(t *testing.T) {
	a := configuredApp(t)
	state, revision := setupTestState(t, a)
	stages := state["stages"].([]any)
	for _, item := range stages {
		item.(map[string]any)["state"] = "completed"
	}
	if _, err := setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": revision, "state": state}); err != nil {
		t.Fatal(err)
	}
	settings := realRequest(a, "settings.show", "", nil).Result.(map[string]any)
	section := settings["sections"].(map[string]any)["media_tools"].(map[string]any)
	pin := map[string]any{"path": filepath.Join(t.TempDir(), "ffprobe"), "sha256": strings.Repeat("a", 64), "version": "test"}
	saved := realRequest(a, "settings.set", "", map[string]any{"section": "media_tools", "revision": section["revision"], "value": map[string]any{"kind": "media-tools", "schema_version": contracts.Version, "exiftool": pin, "ffprobe": pin}})
	if saved.Error != nil {
		t.Fatal(saved.Error)
	}
	out, err := setupTestCall(t, a, "setup.show", contracts.ID(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range out["stages"].([]any) {
		stage := item.(map[string]any)
		if stage["id"] == "import" {
			found = stage["state"] == "needs_attention" && stage["correction"] != ""
		}
	}
	if !found || out["state"].(map[string]any)["stages"].([]any)[4].(map[string]any)["state"] != "completed" {
		t.Fatal("configuration change was not an observational overlay", out)
	}
	current := realRequest(a, "settings.show", "", nil).Result.(map[string]any)["sections"].(map[string]any)["media_tools"].(map[string]any)
	if current["revision"] != saved.Result.(map[string]any)["revision"] {
		t.Fatal("journey inspection changed accepted settings")
	}
}

func TestSetupRejectsSecretsUnknownFieldsAndUnrelatedEnvelope(t *testing.T) {
	a := configuredApp(t)
	state, revision := setupTestState(t, a)
	for _, key := range []string{"password", "api_token", "url", "arbitrary"} {
		input := map[string]any{"expected_revision": revision, "state": state, key: "SETUP_SENTINEL_SECRET"}
		if _, err := setupTestCall(t, a, "setup.save", contracts.ID(), input); err == nil {
			t.Fatal("unknown top-level setup field accepted", key)
		}
	}
	state["choices"].(map[string]any)["token"] = "SETUP_SENTINEL_SECRET"
	if _, err := setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": revision, "state": state}); err == nil {
		t.Fatal("nested credential entered persisted journey")
	}
	raw, _, err := a.Catalog.NamedSetting(context.Background(), "first-run-setup")
	if err != nil || len(raw) != 0 {
		t.Fatal("rejected setup input persisted", string(raw), err)
	}
	for _, operation := range []string{"setup.show", "setup.save", "setup.readiness"} {
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: operation, SourcePath: "unrelated"}
		if operation == "setup.save" {
			req.Data = json.RawMessage(`{}`)
		}
		if contracts.DesktopRequestValid(req) {
			t.Fatal("setup accepted unrelated operation envelope", operation)
		}
	}
}

func TestSetupRequiresExplicitRevisionAndExactStageContract(t *testing.T) {
	a := configuredApp(t)
	state, revision := setupTestState(t, a)
	for _, input := range []map[string]any{
		{"state": state},
		{"expected_revision": nil, "state": state},
		{"expected_revision": -1, "state": state},
	} {
		if _, err := setupTestCall(t, a, "setup.save", contracts.ID(), input); err == nil {
			t.Fatal("missing/null/negative revision became an unconditional initial write", input)
		}
	}
	for _, change := range []func(map[string]any){
		func(s map[string]any) { s["stages"] = s["stages"].([]any)[:1] },
		func(s map[string]any) { s["stages"].([]any)[1].(map[string]any)["id"] = "workspace" },
		func(s map[string]any) { delete(s["stages"].([]any)[0].(map[string]any), "proof") },
		func(s map[string]any) { s["stages"].([]any)[0].(map[string]any)["proof"] = strings.Repeat("A", 64) },
		func(s map[string]any) { s["choices"].(map[string]any)["pipeline_id"] = contracts.ID() },
		func(s map[string]any) { s["status"] = "completed" },
	} {
		encoded, _ := json.Marshal(state)
		var invalid map[string]any
		json.Unmarshal(encoded, &invalid)
		change(invalid)
		input := map[string]any{"expected_revision": revision, "state": invalid}
		if _, err := setupTestCall(t, a, "setup.save", contracts.ID(), input); err == nil {
			t.Fatal("malformed stage/choice contract accepted", input)
		}
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "setup.save"}
		req.Data, _ = json.Marshal(input)
		encoded, _ = json.Marshal(req)
		if schemas.ValidateRequest(encoded) == nil {
			t.Fatal("malformed stage/choice contract accepted by schema", input)
		}
	}
	for _, item := range state["stages"].([]any) {
		stage := item.(map[string]any)
		stage["state"] = "skipped"
		delete(stage, "proof")
	}
	state["status"] = "completed"
	if _, err := setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": revision, "state": state}); err != nil {
		t.Fatal("explicitly skipped stages incorrectly require configuration proofs", err)
	}
}

func TestSetupPortableDestinationProofsAndDeferredWork(t *testing.T) {
	a := configuredApp(t)
	state, revision := setupTestState(t, a)
	state["status"] = "deferred"
	work, err := a.Catalog.EnqueueWork(context.Background(), contracts.ID(), "workers.setup", json.RawMessage(`{"role":"recognition"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": revision, "state": state}); err != nil {
		t.Fatal(err)
	}
	after, err := a.Catalog.Work(context.Background(), work.ID)
	if err != nil || after.State != work.State {
		t.Fatal("deferral cancelled durable accepted work", after, err)
	}
	snapshot, err := a.Catalog.Export(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "restored.sqlite"), a.Workspace.Config.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err = store.Restore(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	destination := *a.Workspace
	destination.Root = t.TempDir()
	destination.Config.Profiles.Graph = workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "ladybugdb", Version: contracts.Version, Configuration: map[string]any{"path": "graph/destination.lbug"}}
	b := &App{Workspace: &destination, Catalog: store, ctx: context.Background()}
	out, err := setupTestCall(t, b, "setup.show", contracts.ID(), nil)
	if err != nil || out["state"].(map[string]any)["status"] != "deferred" {
		t.Fatal("portable journey did not retain logical state", out, err)
	}
	for _, item := range out["stages"].([]any) {
		stage := item.(map[string]any)
		if stage["id"] == "backends" && stage["state"] != "needs_attention" {
			t.Fatal("source backend proof asserted destination configuration", stage)
		}
	}
}

func TestSetupReadinessUsesSavedElectionAndSchema(t *testing.T) {
	a := configuredApp(t)
	value, err := a.setupDispatch(context.Background(), contracts.Request{Operation: "setup.readiness"})
	if err != nil || len(value.(Readiness).Tasks) != 5 {
		t.Fatal("generic shared readiness unavailable", value, err)
	}
	state, revision := setupTestState(t, a)
	state["choices"] = map[string]any{"processing_mode": "local", "pipeline_id": contracts.ID(), "pipeline_revision": 1}
	input := map[string]any{"expected_revision": revision, "state": state}
	out, err := setupTestCall(t, a, "setup.save", contracts.ID(), input)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range out["stages"].([]any) {
		stage := item.(map[string]any)
		if stage["id"] == "processing" && (stage["state"] != "needs_attention" || stage["correction"] == nil) {
			t.Fatal("missing selected pipeline had no corrective stage state", stage)
		}
	}
	if _, err = a.setupDispatch(context.Background(), contracts.Request{Operation: "setup.readiness"}); err == nil || contracts.SafeError(err).Code != "not_found" {
		t.Fatal("saved missing pipeline silently fell back to generic readiness", err)
	}
	for _, op := range []string{"setup.show", "setup.save", "setup.readiness"} {
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: op}
		if op == "setup.save" {
			req.Data, _ = json.Marshal(input)
		}
		encoded, _ := json.Marshal(req)
		if err = schemas.ValidateRequest(encoded); err != nil {
			t.Fatal("documented setup envelope rejected by schema", op, err)
		}
	}
}

func TestSetupPublicRuntimeEnvelopes(t *testing.T) {
	a := configuredApp(t)
	shown := realRequest(a, "setup.show", "", nil)
	if shown.Error != nil {
		t.Fatal("shared runtime did not dispatch optional setup", shown.Error)
	}
	state, revision := setupTestState(t, a)
	saved := realRequest(a, "setup.save", "", map[string]any{"expected_revision": revision, "state": state})
	if saved.Error != nil || saved.Result.(map[string]any)["accepted_revision"] == nil {
		t.Fatal("shared runtime did not return setup acceptance", saved)
	}
	ready := realRequest(a, "setup.readiness", "", nil)
	if ready.Error != nil || len(ready.Result.(Readiness).Tasks) != 5 {
		t.Fatal("shared runtime did not expose existing task readiness", ready)
	}
	for _, operation := range []string{"setup.show", "setup.readiness"} {
		if got := realRequest(a, operation, "", map[string]any{}); got.Error == nil {
			t.Fatal("read-only operation accepted an unrelated payload", operation)
		}
		if got := realRequest(a, operation, contracts.ID(), nil); got.Error == nil {
			t.Fatal("workspace setup accepted an item authority", operation)
		}
	}
}

func TestSetupCompletingProcessingChoicesDoesNotInvalidateItsProof(t *testing.T) {
	a := configuredApp(t)
	id := contracts.ID()
	definition := localDefinition()
	pipeline := map[string]any{"id": id, "name": "Setup election", "preset": "local", "configuration": definition}
	saved := configuredResult(t, realRequest(a, "pipelines.set", id, map[string]any{"expected_revision": 0, "pipeline": pipeline}))
	state, revision := setupTestState(t, a)
	state["choices"] = map[string]any{"processing_mode": "local", "pipeline_id": id, "pipeline_revision": saved["revision"]}
	for _, item := range state["stages"].([]any) {
		stage := item.(map[string]any)
		if stage["id"] == "processing" {
			stage["state"] = "completed"
		}
	}
	out, err := setupTestCall(t, a, "setup.save", contracts.ID(), map[string]any{"expected_revision": revision, "state": state})
	if err != nil {
		t.Fatal(err)
	}
	stageState := func(out map[string]any) string {
		for _, item := range out["stages"].([]any) {
			stage := item.(map[string]any)
			if stage["id"] == "processing" {
				return stage["state"].(string)
			}
		}
		return "missing"
	}
	if stageState(out) != "completed" {
		t.Fatal("completing a valid processing choice invalidated its displayed proof", out)
	}
	pipeline["name"] = "Changed concurrently"
	configuredResult(t, realRequest(a, "pipelines.set", id, map[string]any{"expected_revision": saved["revision"], "pipeline": pipeline}))
	out, err = setupTestCall(t, a, "setup.show", contracts.ID(), nil)
	if err != nil || stageState(out) != "needs_attention" {
		t.Fatal("changed selected pipeline revision was not reconciled", out, err)
	}
	if _, err = a.setupDispatch(context.Background(), contracts.Request{Operation: "setup.readiness"}); err == nil || contracts.SafeError(err).Code != "conflict" {
		t.Fatal("stale pipeline silently selected the newer revision", err)
	}
}
