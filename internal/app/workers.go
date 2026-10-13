// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/workers"
	"github.com/shruggietech/insonic/internal/workspace"
)

func workerOperation(op string) bool {
	switch op {
	case "workers.catalog", "workers.inspect", "workers.plan", "workers.setup", "processing.readiness", "models.defaults", "models.prepare":
		return true
	}
	return false
}
func workerRequestValid(req contracts.Request) bool {
	if !workerOperation(req.Operation) || req.ItemID != "" || req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 {
		return false
	}
	if req.Operation == "workers.catalog" || req.Operation == "models.defaults" {
		return len(req.Data) == 0
	}
	return req.Operation == "processing.readiness" || len(req.Data) > 0
}

type workerPayload struct {
	Election workers.Election `json:"election"`
	Tools    ProcessingTools  `json:"tools"`
}

func (a *App) workerService() workers.Service { return workers.Service{Directory: a.Workspace.Control} }
func (a *App) workerDispatch(ctx context.Context, req contracts.Request) (any, error) {
	s := a.workerService()
	switch req.Operation {
	case "workers.catalog":
		candidates := []string{}
		if c, e := ReadProcessingTools(a.Workspace); e == nil {
			for _, p := range []string{c.Processing.RecognitionPython.Path, c.Processing.DiarizationPython.Path} {
				if p != "" {
					candidates = append(candidates, p)
				}
			}
		}
		for _, name := range []string{"python3", "python"} {
			if p, e := exec.LookPath(name); e == nil && filepath.IsAbs(p) && !strings.Contains(strings.ToLower(p), "windowsapps") {
				candidates = append(candidates, p)
			}
		}
		return map[string]any{"catalog": workers.Inventory(), "platform": runtime.GOOS + "_" + runtime.GOARCH, "cpus": runtime.NumCPU(), "recommended_threads": min(4, max(1, runtime.NumCPU())), "recommended_device": "cpu", "candidates": candidates, "candidate_verification": "not_checked", "models": models.Defaults()}, nil
	case "models.defaults":
		return a.defaultModels(ctx), nil
	case "models.prepare":
		return a.defaultModelPrepare(ctx, req)
	case "processing.readiness":
		var input ReadinessInput
		if len(req.Data) > 0 && strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return a.processingReadiness(ctx, input)
	case "workers.inspect":
		var input struct {
			Role       string `json:"role"`
			PythonPath string `json:"python_path,omitempty"`
		}
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if input.PythonPath == "" {
			if c, e := ReadProcessingTools(a.Workspace); e == nil {
				input.PythonPath = c.Processing.RecognitionPython.Path
				if input.Role == "diarization" {
					input.PythonPath = c.Processing.DiarizationPython.Path
				}
			}
		}
		return s.Inspect(ctx, input.Role, input.PythonPath, false)
	case "workers.plan", "workers.setup":
		var input workers.Election
		if strictPayload(req.Data, &input) != nil || input.Validate() != nil {
			return nil, contracts.Fail("invalid_request")
		}
		p, e := workers.Selected(input.Role)
		if e != nil {
			return nil, e
		}
		if input.ProfileDigest != "" && input.ProfileDigest != p.Digest() {
			return nil, contracts.Fail("incompatible_version")
		}
		input.ProfileDigest = p.Digest()
		if req.Operation == "workers.setup" {
			if prior, err := a.Catalog.Work(ctx, req.RequestID); err == nil {
				var saved workerPayload
				if prior.Kind != "workers.setup" || strictPayload(prior.Payload, &saved) != nil || documentHash(configurationBytes(saved.Election)) != documentHash(configurationBytes(input)) {
					return nil, contracts.Fail("conflict")
				}
				return a.modelWorkView(prior), nil
			} else if typed, ok := err.(*contracts.Error); !ok || typed.Code != "not_found" {
				return nil, err
			}
		}
		a.settingsMu.Lock()
		raw, e := settingRead(filepath.Join(a.Workspace.Control, "processing-tools.json"))
		a.settingsMu.Unlock()
		if e != nil {
			return nil, e
		}
		revision := settingRevision(raw)
		c, e := ReadProcessingTools(a.Workspace)
		if req.Operation == "workers.plan" {
			input.ExpectedRevision = revision
			plan := map[string]any{"election": input, "profile": p, "destination": filepath.Join(a.Workspace.Control, "workers"), "required_space_bytes": p.MinSpaceBytes, "download_bytes": p.DownloadBytes, "recommended_threads": min(4, max(1, runtime.NumCPU())), "recommended_device": "cpu", "effects": "Only successful verified work publishes the elected role; accepted environments are not updated in place."}
			if input.Mode == "managed" {
				version, err := s.PlatformPrerequisite(ctx, p)
				if version != "" {
					plan["observed_glibc"] = version
				}
				if err != nil {
					plan["prerequisite"] = err.Error()
				}
			}
			if free, err := workers.FreeSpace(a.Workspace.Control); err == nil {
				plan["available_space_bytes"] = free
				plan["space_sufficient"] = free >= uint64(p.MinSpaceBytes)
			}
			if input.Mode == "reuse" {
				plan["download_bytes"] = 0
				plan["inspection"], _ = s.Inspect(ctx, input.Role, input.PythonPath, false)
			}
			if input.Mode == "managed" && e == nil {
				python, receipt := c.Processing.RecognitionPython, c.Processing.RecognitionEnvironment
				if input.Role == "diarization" {
					python, receipt = c.Processing.DiarizationPython, c.Processing.DiarizationEnvironment
				}
				if existing, ok := workers.ExistingManaged(ctx, receipt, python, p); ok {
					plan["verified_existing_environment"] = existing
					plan["download_bytes"] = 0
					plan["required_space_bytes"] = 0
					plan["space_sufficient"] = true
				}
			}
			if e != nil || c.Processing.Worker.Path == "" {
				plan["prerequisite"] = "Install the current package or configure verified Cueson/media and worker tools before publishing setup."
			}
			return plan, nil
		}
		if !hashString.MatchString(input.ExpectedRevision) || input.ExpectedRevision != revision {
			return nil, contracts.Fail("conflict")
		}
		if e != nil {
			return nil, e
		}
		if c.Processing.Worker.Path == "" {
			return nil, contracts.Fail("engine_unavailable")
		}
		payload := workerPayload{Election: input, Tools: c}
		if len(raw) == 0 {
			payload.Tools.PackageCompanions = true
		}
		encoded, _ := json.Marshal(payload)
		w, e := a.Catalog.EnqueueWork(ctx, req.RequestID, "workers.setup", encoded)
		if e == nil {
			a.recoverWork()
		}
		return a.modelWorkView(w), e
	}
	return nil, contracts.Fail("invalid_request")
}

func (a *App) executeWorkerSetup(ctx context.Context, claim catalog.Work) (any, error) {
	var payload workerPayload
	if strictPayload(claim.Payload, &payload) != nil || payload.Election.Validate() != nil {
		return nil, contracts.Fail("invalid_request")
	}
	// Recovery after a successful settings rename uses its prepared checkpoint.
	var prepared workers.Prepared
	if claim.Phase == "environment-prepared" && strictPayload(claim.Result, &prepared) == nil && workers.VerifyReceipt(ctx, prepared.Receipt, prepared.Python) == nil {
		return a.publishWorkerSetup(ctx, claim, payload, prepared)
	}
	s := a.workerService()
	s.Directory = filepath.Join(a.Workspace.Control, "workers")
	s.Progress = func(phase string, data map[string]any) error {
		raw, _ := json.Marshal(data)
		_, e := a.Catalog.CheckpointWork(ctx, claim, phase, "running", raw, leaseTTL)
		return e
	}
	transfer := models.NewService(&artifact.Service{Workspace: a.Workspace}, a.Catalog, nil)
	s.Fetch = func(ctx context.Context, asset workers.Asset) (string, error) {
		return transfer.DownloadAsset(ctx, claim.ID, models.File{Role: asset.Name, URL: asset.URL, SHA256: asset.SHA256, Size: asset.Size}, func(n int64) error {
			return s.Progress("asset-download", map[string]any{"current_asset": asset.Name, "downloaded_bytes": n, "total_bytes": asset.Size})
		})
	}
	var e error
	if payload.Election.Mode == "managed" {
		p, err := workers.Selected(payload.Election.Role)
		if err != nil {
			return nil, err
		}
		python, receipt := payload.Tools.Processing.RecognitionPython, payload.Tools.Processing.RecognitionEnvironment
		if payload.Election.Role == "diarization" {
			python, receipt = payload.Tools.Processing.DiarizationPython, payload.Tools.Processing.DiarizationEnvironment
		}
		if existing, ok := workers.ExistingManaged(ctx, receipt, python, p); ok {
			prepared = existing
		}
	}
	if prepared.Python.Path == "" {
		prepared, e = s.Prepare(ctx, payload.Election)
	}
	if e != nil {
		return nil, e
	}
	raw, _ := json.Marshal(prepared)
	if _, e = a.Catalog.CheckpointWork(ctx, claim, "environment-prepared", "running", raw, leaseTTL); e != nil {
		return nil, e
	}
	return a.publishWorkerSetup(ctx, claim, payload, prepared)
}

func (a *App) publishWorkerSetup(ctx context.Context, claim catalog.Work, payload workerPayload, prepared workers.Prepared) (any, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if e := workers.VerifyReceipt(ctx, prepared.Receipt, prepared.Python); e != nil {
		return nil, e
	}
	// Publication and cancellation share workMu. Once published, complete the
	// work before releasing it so a later cancel cannot relabel accepted setup.
	a.workMu.Lock()
	defer a.workMu.Unlock()
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if _, e := a.Catalog.RenewWork(ctx, claim, 30*time.Second); e != nil {
		return nil, e
	}
	c := payload.Tools
	c.Processing.Threads = min(4, max(1, runtime.NumCPU()))
	if payload.Election.Role == "recognition" {
		c.Processing.RecognitionPython = prepared.Python
		c.Processing.RecognitionEnvironment = prepared.Receipt
	} else {
		c.Processing.DiarizationPython = prepared.Python
		c.Processing.DiarizationEnvironment = prepared.Receipt
	}
	value, _ := json.Marshal(c)
	input, _ := json.Marshal(map[string]any{"section": "processing_tools", "revision": payload.Election.ExpectedRevision, "value": json.RawMessage(value)})
	result, e := a.settingsSet(contracts.Request{Data: input})
	if e != nil {
		return nil, e
	}
	out := map[string]any{"role": payload.Election.Role, "python": prepared.Python, "environment_receipt": prepared.Receipt, "processing_tools_revision": result.(map[string]any)["revision"], "verification": "dependency_bytes_and_library_probe", "inference": "not_run"}
	raw, _ := json.Marshal(out)
	_, e = a.Catalog.CheckpointWork(ctx, claim, "complete", "succeeded", raw, leaseTTL)
	return out, e
}

type defaultModelInput struct {
	Choice               string `json:"choice"`
	CredentialID         string `json:"credential_id,omitempty"`
	LocalDirectory       string `json:"local_directory,omitempty"`
	DestinationProfileID string `json:"destination_profile_id,omitempty"`
	Acquire              bool   `json:"acquire,omitempty"`
}

func (a *App) defaultModels(ctx context.Context) []any {
	out := []any{}
	for _, c := range models.Defaults() {
		item := map[string]any{"choice": c, "model_id": models.InstallationID(c.Manifest), "total_bytes": c.Bytes()}
		if r, e := a.modelReadiness(models.InstallationID(c.Manifest)); e == nil {
			item["readiness"] = r
		}
		out = append(out, item)
	}
	return out
}
func (a *App) defaultModelPrepare(ctx context.Context, req contracts.Request) (any, error) {
	var input defaultModelInput
	if strictPayload(req.Data, &input) != nil || input.LocalDirectory != "" && !filepath.IsAbs(input.LocalDirectory) {
		return nil, contracts.Fail("invalid_request")
	}
	c, e := models.Default(input.Choice, input.CredentialID)
	if e != nil {
		return nil, e
	}
	// An omitted credential reuses the exact previously elected protected reference.
	// Token rotation replaces that secret, without changing the bundle identity.
	if input.CredentialID == "" {
		if old, err := a.Catalog.BaseModel(ctx, models.InstallationID(c.Manifest)); err == nil {
			var prior models.Manifest
			if models.DecodeManifest(old.Manifest, &prior) != nil || !sameDefaultBundle(prior, c.Manifest) {
				return nil, contracts.Fail("conflict")
			}
			c.Manifest = prior
		} else {
			all, err := a.Catalog.Works(ctx)
			if err != nil {
				return nil, err
			}
			for _, work := range all {
				if work.Kind != "models.acquire" {
					continue
				}
				var prior models.Request
				if strictPayload(work.Payload, &prior) == nil && sameDefaultBundle(prior.Manifest, c.Manifest) {
					c.Manifest = prior.Manifest
					break
				}
			}
		}
	}
	destination, e := a.selectedStorageProfile(ctx, input.DestinationProfileID)
	if e != nil {
		return nil, e
	}

	frozen := false
	acquisitionID, e := a.acquisitionWorkID(ctx, c.Manifest, "")
	if e != nil {
		return nil, e
	}
	if existing, err := a.Catalog.Work(ctx, acquisitionID); err == nil {
		var prior models.Request
		if existing.Kind != "models.acquire" || strictPayload(existing.Payload, &prior) != nil {
			return nil, contracts.Fail("conflict")
		}
		if prior.Destination != nil {
			destination = *prior.Destination
		}
		input.LocalDirectory = prior.LocalDirectory
		frozen = true
	}
	download, scratch, reuse := c.Bytes(), c.Bytes(), int64(0)
	state := "not_acquired"
	service, serviceErr := a.modelService()
	if serviceErr == nil && service.Verify(ctx, models.InstallationID(c.Manifest)) == nil {
		download, scratch, reuse, state = 0, 0, c.Bytes(), "verified_available"
	} else if input.LocalDirectory != "" {
		if e = models.VerifyLocal(ctx, input.LocalDirectory, c.Manifest); e != nil {
			return nil, e
		}
		download, reuse, state = 0, c.Bytes(), "verified_local"
		// Publication stages one role at a time, including local reuse.
		scratch = 0
		for _, f := range c.Manifest.Files {
			scratch = max(scratch, f.Size)
		}
	}
	view := map[string]any{"choice": c, "model_id": models.InstallationID(c.Manifest), "manifest_digest": c.Manifest.Digest(), "total_bytes": c.Bytes(), "download_bytes": download, "verified_reuse_bytes": reuse, "state": state, "scratch_required_bytes": scratch, "destination": destination, "destination_frozen": frozen, "storage_capacity": "unknown", "local_reuse": input.LocalDirectory != "", "local_directory": input.LocalDirectory}
	if free, e := workers.FreeSpace(a.Workspace.Control); e == nil {
		// A downloaded source and its publication stage can coexist.
		required := scratch + download
		view["scratch_required_bytes"] = required
		view["scratch_available_bytes"] = free
		view["scratch_space_sufficient"] = free >= uint64(required)
		if input.Acquire && free < uint64(required) {
			return view, contracts.Fail("insufficient_space")
		}
	}
	if destination.Adapter == "filesystem" {
		required := c.Bytes()
		if state == "verified_available" {
			required = 0
		}
		path, _ := destination.Configuration["root"].(string)
		if !filepath.IsAbs(path) {
			path = filepath.Join(a.Workspace.Control, path)
		}
		if free, e := workers.FreeSpace(path); e == nil {
			view["storage_capacity"] = "observed_filesystem_free_space"
			view["storage_available_bytes"] = free
			view["storage_space_sufficient"] = free >= uint64(required)
			if input.Acquire && free < uint64(required) {
				return view, contracts.Fail("insufficient_space")
			}
		}
	}
	if !input.Acquire {
		return view, nil
	}
	if serviceErr != nil {
		return view, serviceErr
	}
	selected, e := models.ResolveManifest(c.Manifest, "")
	if e != nil {
		return nil, e
	}
	work, e := a.enqueueModelAcquisitionTo(ctx, selected, false, &destination, input.LocalDirectory)
	if e != nil {
		return view, e
	}
	verified := service.Verify(ctx, selected.Target.ID) == nil
	if work.State == "failed" || work.State == "cancelled" || work.State == "interrupted" || work.State == "succeeded" && !verified {
		work, e = a.enqueueModelAcquisitionTo(ctx, selected, true, &destination, input.LocalDirectory)
	}
	if e != nil {
		return view, e
	}
	a.recoverWork()
	view["work"] = a.modelWorkView(work)
	view["acquisition_work_id"] = work.ID
	return view, nil
}
func (a *App) selectedStorageProfile(ctx context.Context, id string) (workspaceProfile workspace.Profile, e error) {
	if id == "" || id == a.Workspace.Config.Profiles.Storage.ID {
		return a.Workspace.Config.Profiles.Storage, nil
	}
	if !contracts.ValidID(id) {
		return workspaceProfile, contracts.Fail("invalid_request")
	}
	p, e := a.Catalog.Profile(ctx, id, 0)
	if e != nil {
		return workspaceProfile, e
	}
	return artifact.WorkspaceProfile(p)
}

func sameDefaultBundle(a, b models.Manifest) bool {
	// Credential references route transport only; every upstream identity, role,
	// size, license and compatibility declaration still has to match.
	a.Files = append([]models.File(nil), a.Files...)
	b.Files = append([]models.File(nil), b.Files...)
	for i := range a.Files {
		a.Files[i].CredentialID = ""
	}
	for i := range b.Files {
		b.Files[i].CredentialID = ""
	}
	return a.Digest() == b.Digest()
}
