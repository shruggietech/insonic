// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workers"
	"github.com/shruggietech/insonic/internal/workspace"
)

func workerToolsFixture(t *testing.T, a *App) ProcessingTools {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fixture-tool")
	os.WriteFile(p, []byte("fixture"), 0700)
	pin, _ := workers.Hash(context.Background(), p)
	c := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: p, ExecutableSHA256: pin.SHA256}, Processing: processing.Config{Worker: pin, FFmpeg: library.Tool{Path: p, SHA256: pin.SHA256, Version: "fixture"}}}
	b, _ := json.Marshal(c)
	os.WriteFile(filepath.Join(a.Workspace.Control, "processing-tools.json"), b, 0600)
	return c
}
func TestWorkerReadinessDoesNotPromoteMediaToolsToInference(t *testing.T) {
	a := configuredApp(t)
	workerToolsFixture(t, a)
	got, e := a.processingReadiness(context.Background(), ReadinessInput{})
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Tasks) != 5 || got.Tasks["transcription"].Usable || got.Tasks["diarization"].Usable || got.Tasks["training"].Usable || !got.Tasks["existing_subtitles"].Usable {
		t.Fatal(got)
	}
}

func TestReusePlanUsesReceiptBudgetInsteadOfManagedInstallationMinimum(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	raw, _ := json.Marshal(workers.Election{Role: "recognition", Mode: "reuse", PythonPath: tools.Processing.Worker.Path})
	out, e := a.workerDispatch(a.ctx, contracts.Request{Operation: "workers.plan", Data: raw})
	if e != nil {
		t.Fatal(e)
	}
	plan := out.(map[string]any)
	if plan["required_space_bytes"] != workers.ReuseSpaceBytes || plan["download_bytes"] != 0 {
		t.Fatal("managed capacity applied to reuse", plan)
	}
}
func TestWorkerSetupRevisionConflictAndCancellationPreserveSettings(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	path := filepath.Join(a.Workspace.Control, "processing-tools.json")
	before, _ := os.ReadFile(path)
	p, _ := workers.Selected("recognition")
	payload := workerPayload{Election: workers.Election{Role: "recognition", Mode: "reuse", PythonPath: tools.Processing.Worker.Path, ExpectedRevision: strings.Repeat("0", 64), ProfileDigest: p.Digest()}, Tools: tools}
	raw, _ := json.Marshal(payload)
	w, e := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "workers.setup", raw)
	if e != nil {
		t.Fatal(e)
	}
	w, e = a.Catalog.ClaimWork(a.ctx, w.ID, a.Session, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	receipt := workers.Receipt{Kind: "worker-environment", Version: contracts.Version, ProfileDigest: p.Digest(), Inspection: workers.Inspection{Role: "recognition", Python: tools.Processing.Worker, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{tools.Processing.Worker}}}
	body, _ := json.Marshal(receipt)
	rp := filepath.Join(t.TempDir(), "receipt.json")
	os.WriteFile(rp, body, 0600)
	receiptPin, _ := workers.Hash(context.Background(), rp)
	prepared := workers.Prepared{Python: tools.Processing.Worker, Receipt: receiptPin}
	if _, e = a.publishWorkerSetup(context.Background(), w, payload, prepared); e == nil {
		t.Fatal("stale settings replaced")
	}
	payload.Election.ExpectedRevision = settingRevision(before)
	if _, e = a.Catalog.CancelWork(a.ctx, contracts.ID(), w.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = a.publishWorkerSetup(context.Background(), w, payload, prepared); e == nil {
		t.Fatal("cancelled claim published")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("accepted settings changed")
	}
}
func TestDefaultModelPreviewIsReadOnlyAndNamesStorage(t *testing.T) {
	a := configuredApp(t)
	raw, _ := json.Marshal(defaultModelInput{Choice: "whisper-tiny"})
	out, e := a.defaultModelPrepare(context.Background(), contracts.Request{Data: raw})
	if e != nil {
		t.Fatal(e)
	}
	view := out.(map[string]any)
	if view["total_bytes"] != int64(78203619) || view["destination"] == nil {
		t.Fatal(view)
	}
	all, e := a.Catalog.Works(a.ctx)
	if e != nil || len(all) != 0 {
		t.Fatal("preview acquired work", all, e)
	}
}

func TestModelCapacityCombinesSharedVolumePeak(t *testing.T) {
	for _, v := range []struct {
		scratch, download, storage int64
		shared                     bool
		control, destination       int64
	}{
		{100, 100, 100, true, 300, 300},
		{100, 100, 100, false, 200, 100},
		{80, 0, 100, true, 180, 180},
		{0, 0, 0, true, 0, 0},
	} {
		control, destination := modelCapacityBudget(v.scratch, v.download, v.storage, v.shared)
		if control != v.control || destination != v.destination {
			t.Fatal(v, control, destination)
		}
	}
}

func TestGatedModelWithoutReferenceRejectsBeforeEnqueueButAllowsPreview(t *testing.T) {
	a := configuredApp(t)
	preview, _ := json.Marshal(defaultModelInput{Choice: "community-1"})
	view, e := a.defaultModelPrepare(a.ctx, contracts.Request{Data: preview})
	if e != nil || view.(map[string]any)["access_ready"] != false {
		t.Fatal(view, e)
	}
	input, _ := json.Marshal(defaultModelInput{Choice: "community-1", Acquire: true})
	_, e = a.defaultModelPrepare(a.ctx, contracts.Request{Data: input})
	if typed, ok := e.(*contracts.Error); !ok || typed.Reason != "model_access_required" {
		t.Fatal("missing reference was enqueued", e)
	}
	all, e := a.Catalog.Works(a.ctx)
	if e != nil || len(all) != 0 {
		t.Fatal("failed prerequisite created work", all, e)
	}
}

func TestManagedSetupReusesVerifiedEnvironmentAndReplaysAfterPublication(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	p, _ := workers.Selected("recognition")
	r := workers.Receipt{Kind: "worker-environment", Version: contracts.Version, Mode: "managed", ProfileDigest: p.Digest(), Inspection: workers.Inspection{Role: "recognition", Python: tools.Processing.Worker, PythonVersion: p.Python, Packages: p.Engines, Compatible: true, EngineAvailable: true, Files: []library.PinnedFile{tools.Processing.Worker}}}
	b, _ := json.Marshal(r)
	receiptPath := filepath.Join(t.TempDir(), "environment.json")
	os.WriteFile(receiptPath, b, 0600)
	pin, _ := workers.Hash(a.ctx, receiptPath)
	tools.Processing.RecognitionPython = tools.Processing.Worker
	tools.Processing.RecognitionEnvironment = pin
	b, _ = json.Marshal(tools)
	settingsPath := filepath.Join(a.Workspace.Control, "processing-tools.json")
	os.WriteFile(settingsPath, b, 0600)
	election := workers.Election{Role: "recognition", Mode: "managed", ProfileDigest: p.Digest(), ExpectedRevision: settingRevision(b)}
	payload := workerPayload{Election: election, Tools: tools}
	raw, _ := json.Marshal(payload)
	a.workMu.Lock()
	w, e := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "workers.setup", raw)
	if e == nil {
		w, e = a.Catalog.ClaimWork(a.ctx, w.ID, a.Session, time.Minute)
	}
	a.workMu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.executeWorkerSetup(a.ctx, w); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(a.Workspace.Control, "workers")); !os.IsNotExist(e) {
		t.Fatal("unchanged accepted environment was reinstalled")
	}
	input, _ := json.Marshal(election)
	view, e := a.workerDispatch(a.ctx, contracts.Request{Operation: "workers.setup", RequestID: w.ID, Data: input})
	if e != nil || view.(map[string]any)["state"] != "succeeded" {
		t.Fatal("completed setup replay lost its original election", view, e)
	}
}

func TestDefaultPreviewRetainsProtectedReferenceAndFirstDestination(t *testing.T) {
	a := configuredApp(t)
	choice, _ := models.Default("community-1", contracts.ID())
	destination := a.Workspace.Config.Profiles.Storage
	raw, _ := json.Marshal(models.Request{Manifest: choice.Manifest, Destination: &destination})
	a.workMu.Lock()
	defer a.workMu.Unlock()
	_, e := a.Catalog.EnqueueWork(a.ctx, models.AcquisitionID(a.Workspace.Config.WorkspaceID, choice.Manifest), "models.acquire", raw)
	if e != nil {
		t.Fatal(e)
	}
	a.Workspace.Config.Profiles.Storage.ID = contracts.ID()
	input, _ := json.Marshal(defaultModelInput{Choice: "community-1"})
	out, e := a.defaultModelPrepare(a.ctx, contracts.Request{Data: input})
	if e != nil {
		t.Fatal(e)
	}
	view := out.(map[string]any)
	got := view["choice"].(models.Choice)
	if got.Manifest.Digest() != choice.Manifest.Digest() || view["destination"].(workspace.Profile).ID != destination.ID || view["destination_frozen"] != true {
		t.Fatal(view)
	}
}
func TestDefaultPreviewReusesRegisteredCredentialIdentity(t *testing.T) {
	a := configuredApp(t)
	c, _ := models.Default("community-1", contracts.ID())
	raw, _ := json.Marshal(c.Manifest)
	a.workMu.Lock()
	defer a.workMu.Unlock()
	w, e := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "models.register", []byte("{}"))
	if e == nil {
		w, e = a.Catalog.ClaimWork(a.ctx, w.ID, a.Session, time.Minute)
	}
	if e != nil {
		t.Fatal(e)
	}
	_, e = a.Catalog.CommitBaseModel(a.ctx, w, catalog.BaseModelInstall{ID: models.InstallationID(c.Manifest), Name: c.Manifest.Name, Version: c.Manifest.ModelVersion, Digest: c.Manifest.Digest(), Manifest: raw, PublicationIDs: []byte("[]"), State: "registered"})
	if e != nil {
		t.Fatal(e)
	}
	input, _ := json.Marshal(defaultModelInput{Choice: "community-1"})
	out, e := a.defaultModelPrepare(a.ctx, contracts.Request{Data: input})
	if e != nil || out.(map[string]any)["manifest_digest"] != c.Manifest.Digest() {
		t.Fatal(out, e)
	}
}
