// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/workers"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkerExpiredReadinessDoesNotClaimChangedTools(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	got := checkedPin(ctx, "processing worker", tools.Processing.Worker)
	if got.State != "unknown" || got.Available {
		t.Fatalf("unverified tool was labeled %s", got.State)
	}
}
func TestWorkerExpiredReadinessLabelsUnattemptedRoles(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	tools.Processing.RecognitionPython, tools.Processing.DiarizationPython = tools.Processing.Worker, tools.Processing.Worker
	raw, _ := json.Marshal(tools)
	if err := os.WriteFile(filepath.Join(a.Workspace.Control, "processing-tools.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	got, err := a.processingReadiness(ctx, ReadinessInput{})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"recognition", "diarization"} {
		inspection := got.Environments[role]
		if inspection.Probe == nil || inspection.Probe.Reason != "deadline" || inspection.Probe.Stage != "not_started" {
			t.Fatalf("%s unattempted inspection: %+v", role, inspection.Probe)
		}
	}
	if len(got.Tasks["transcription"].Prerequisites) < 8 {
		t.Fatal("partial readiness discarded other prerequisite facts")
	}
	if got.Tasks["transcription"].State != "unknown" || got.Tasks["diarization"].State != "unknown" || got.Tasks["training"].State != "unknown" {
		t.Fatalf("expired readiness was promoted or mislabeled: %+v", got.Tasks)
	}
}

func TestFailedWorkerSetupRetainsValidatedProbeWithoutPublishing(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	before, err := os.ReadFile(filepath.Join(a.Workspace.Control, "processing-tools.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := workers.Selected("recognition")
	payload := workerPayload{Election: workers.Election{Role: "recognition", Mode: "reuse", PythonPath: tools.Processing.Worker.Path, ExpectedRevision: settingRevision(before), ProfileDigest: p.Digest()}, Tools: tools}
	raw, _ := json.Marshal(payload)
	pending, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "workers.setup", raw)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, pending.ID, a.Session, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	worker := &realWorker{claim: claim, cancel: cancel}
	a.workMu.Lock()
	a.workers[claim.ID] = worker
	a.workMu.Unlock()
	a.wg.Add(1)
	a.executeWork(ctx, worker, claim)
	got, err := a.Catalog.Work(a.ctx, claim.ID)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Cause       *contracts.Error `json:"cause"`
		WorkerProbe struct {
			Role  string         `json:"role"`
			Probe *workers.Probe `json:"probe"`
		} `json:"worker_probe"`
	}
	if json.Unmarshal(got.Result, &result) != nil || got.State != "failed" || result.Cause == nil || result.Cause.Reason != "worker_probe_launch_failed" || result.WorkerProbe.Probe == nil || !result.WorkerProbe.Probe.Valid() || result.WorkerProbe.Probe.Stage != "process_start" {
		t.Fatalf("lost failed setup diagnosis: %s %s", got.State, got.Result)
	}
	after, _ := os.ReadFile(filepath.Join(a.Workspace.Control, "processing-tools.json"))
	if string(before) != string(after) {
		t.Fatal("failed setup published tools")
	}
	entries, _ := os.ReadDir(filepath.Join(a.Workspace.Control, "workers"))
	if len(entries) != 0 {
		t.Fatal("failed setup attempt was retained")
	}
	if strings.Contains(string(got.Result), tools.Processing.Worker.Path) {
		t.Fatal("diagnostic copied private interpreter path")
	}
}

func TestExpiredModelReadinessDoesNotClaimMissingOrChangedBytes(t *testing.T) {
	a := configuredApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := a.modelPrerequisite(ctx, "not-observed", "transcription")
	if got.State != "unknown" || got.Available {
		t.Fatalf("interrupted model verification became %s", got.State)
	}
}

func TestExpiredReadinessRetainsKnownMissingConfiguration(t *testing.T) {
	a := configuredApp(t)
	workerToolsFixture(t, a)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	got, err := a.processingReadiness(ctx, ReadinessInput{})
	if err != nil || got.Tasks["transcription"].State != "needs_attention" {
		t.Fatal("unknown probe discarded known missing interpreter configuration", err)
	}
}

func TestHealthyWorkerProbeSurvivesRealSetupCheckpointAndPublication(t *testing.T) {
	a := configuredApp(t)
	tools := workerToolsFixture(t, a)
	before, err := os.ReadFile(filepath.Join(a.Workspace.Control, "processing-tools.json"))
	if err != nil {
		t.Fatal(err)
	}
	child := &workers.ProbeProgress{Mode: "snapshot", Stage: "complete", Reason: "success", Files: 1, HashedFiles: 1, HashedBytes: 7}
	for _, stage := range []string{"package_versions", "library_import", "cpu_prerequisites", "cuda_prerequisites", "dependency_inventory", "dependency_hashing", "complete"} {
		point := workers.ProbeCheckpoint{Stage: stage}
		if stage == "dependency_hashing" || stage == "complete" {
			point.Files, point.HashedFiles, point.HashedBytes = 1, 1, 7
		}
		child.Checkpoints = append(child.Checkpoints, point)
	}
	probe := &workers.Probe{Mode: "snapshot", Stage: "complete", Reason: "success", BudgetMS: 60000, Child: child}
	if !probe.Valid() {
		t.Fatal("invalid healthy fixture")
	}
	inspection := workers.Inspection{Role: "recognition", Python: tools.Processing.Worker, Compatible: true, EngineAvailable: true, Probe: probe}
	receipt := workers.Receipt{Kind: "worker-environment", Version: contracts.Version, Inspection: inspection}
	receipt.Inspection.Files = []library.PinnedFile{tools.Processing.Worker}
	raw, _ := json.Marshal(receipt)
	name := filepath.Join(a.Workspace.Control, "environment-fixture.json")
	if err := os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	pin, err := workers.Hash(a.ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	prepared := workers.Prepared{Python: tools.Processing.Worker, Receipt: pin, Inspection: inspection}
	payload := workerPayload{Election: workers.Election{Role: "recognition", Mode: "reuse", ExpectedRevision: settingRevision(before)}, Tools: tools}
	raw, _ = json.Marshal(payload)
	pending, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "workers.setup", raw)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, pending.ID, a.Session, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the generic capture-payload guard; bounded checkpoints need no exception.
	if _, err := a.Catalog.CheckpointWork(a.ctx, claim, "unsafe-history", "running", json.RawMessage(`{"probe":{"observations":[{"raw":"copied capture"}]}}`), time.Minute); err == nil {
		t.Fatal("raw observations entered the durable result")
	}
	raw, _ = json.Marshal(prepared)
	if _, err := a.Catalog.CheckpointWork(a.ctx, claim, "environment-prepared", "running", raw, time.Minute); err != nil {
		t.Fatalf("healthy prepared checkpoint rejected: %v", err)
	}
	if _, err := a.publishWorkerSetup(a.ctx, claim, payload, prepared); err != nil {
		t.Fatal(err)
	}
	final, err := a.Catalog.Work(a.ctx, claim.ID)
	if err != nil || final.State != "succeeded" {
		t.Fatal("healthy setup did not publish", err)
	}
	var result struct {
		Probe *workers.Probe `json:"probe"`
	}
	if json.Unmarshal(final.Result, &result) != nil || result.Probe == nil || !result.Probe.Valid() {
		t.Fatal("success lost its bounded probe", string(final.Result))
	}
}
